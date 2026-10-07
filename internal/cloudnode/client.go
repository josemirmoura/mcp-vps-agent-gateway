package cloudnode

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	nodeSigningProtocol = "PORTICO-NODE-v1"
	maxResponseBytes    = 2 << 20
)

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Portico Cloud HTTP %d: %s", e.StatusCode, e.Message)
}

func IsStatus(err error, status int) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == status
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewClient(rawBaseURL string, httpClient *http.Client) (*Client, error) {
	baseURL, err := url.Parse(strings.TrimSpace(rawBaseURL))
	if err != nil {
		return nil, err
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, errors.New("Portico Cloud URL must include scheme and host")
	}
	if baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, errors.New("Portico Cloud URL must not include credentials, query, or fragment")
	}
	if baseURL.Path != "" && baseURL.Path != "/" {
		return nil, errors.New("Portico Cloud URL must not include a path prefix")
	}
	if baseURL.Scheme != "https" && !(baseURL.Scheme == "http" && isLoopbackHost(baseURL.Hostname())) {
		return nil, errors.New("Portico Cloud URL requires HTTPS except for loopback development")
	}
	baseURL.Path = ""
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{baseURL: baseURL, httpClient: httpClient}, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *Client) Enroll(
	ctx context.Context,
	token, nodeName, platform, publicKey string,
) (EnrollmentResponse, error) {
	request := EnrollmentRequest{
		Token:             token,
		Name:              nodeName,
		Platform:          platform,
		PublicKey:         publicKey,
		ProtocolVersion:   ProtocolVersion,
		CapabilityVersion: ProtocolVersion,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return EnrollmentResponse{}, err
	}

	var response EnrollmentResponse
	if _, err := c.doJSON(ctx, http.MethodPost, "/api/v1/nodes/enroll", body, nil, &response); err != nil {
		return EnrollmentResponse{}, err
	}
	if response.NodeID == "" || response.WorkspaceID == "" || response.Fingerprint == "" {
		return EnrollmentResponse{}, errors.New("Portico Cloud returned incomplete enrollment data")
	}
	return response, nil
}

func (c *Client) Heartbeat(ctx context.Context, identity Identity) error {
	body, err := json.Marshal(map[string]any{
		"status":             "healthy",
		"protocol_version":   ProtocolVersion,
		"capability_version": ProtocolVersion,
	})
	if err != nil {
		return err
	}
	path := "/api/v1/nodes/" + url.PathEscape(identity.NodeID) + "/heartbeat"
	_, err = c.doJSON(ctx, http.MethodPost, path, body, &identity, nil)
	return err
}

func (c *Client) LeaseTask(ctx context.Context, identity Identity) (Task, bool, error) {
	path := "/api/v1/nodes/" + url.PathEscape(identity.NodeID) + "/tasks/lease"
	var task Task
	status, err := c.doJSON(ctx, http.MethodPost, path, []byte("{}"), &identity, &task)
	if err != nil {
		return Task{}, false, err
	}
	if status == http.StatusNoContent {
		return Task{}, false, nil
	}
	if task.ID == "" || task.LeaseID == "" || task.DestinationNodeID != identity.NodeID {
		return Task{}, false, errors.New("Portico Cloud returned an invalid task lease")
	}
	return task, true, nil
}

func (c *Client) RenewTaskLease(
	ctx context.Context,
	identity Identity,
	taskID, leaseID string,
) (time.Time, error) {
	body, err := json.Marshal(map[string]string{"lease_id": leaseID})
	if err != nil {
		return time.Time{}, err
	}
	path := "/api/v1/nodes/" + url.PathEscape(identity.NodeID) +
		"/tasks/" + url.PathEscape(taskID) + "/renew"
	var response struct {
		LeaseExpiresAt string `json:"lease_expires_at"`
	}
	if _, err := c.doJSON(ctx, http.MethodPost, path, body, &identity, &response); err != nil {
		return time.Time{}, err
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, response.LeaseExpiresAt)
	if err != nil {
		return time.Time{}, errors.New("Portico Cloud returned an invalid lease expiry")
	}
	return expiresAt, nil
}

func (c *Client) CompleteTask(
	ctx context.Context,
	identity Identity,
	taskID string,
	completion Completion,
) error {
	body, err := json.Marshal(completion)
	if err != nil {
		return err
	}
	path := "/api/v1/nodes/" + url.PathEscape(identity.NodeID) +
		"/tasks/" + url.PathEscape(taskID) + "/complete"
	_, err = c.doJSON(ctx, http.MethodPost, path, body, &identity, nil)
	return err
}

func (c *Client) doJSON(
	ctx context.Context,
	method, path string,
	body []byte,
	identity *Identity,
	out any,
) (int, error) {
	endpoint := *c.baseURL
	endpoint.Path = path

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if identity != nil {
		if err := signRequest(req, body, *identity); err != nil {
			return 0, err
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return resp.StatusCode, err
	}
	if len(responseBody) > maxResponseBytes {
		return resp.StatusCode, errors.New("Portico Cloud response body exceeded limit")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(responseBody))
		if len(message) > 1024 {
			message = message[:1024]
		}
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return resp.StatusCode, &APIError{StatusCode: resp.StatusCode, Message: message}
	}
	if resp.StatusCode == http.StatusNoContent || out == nil {
		return resp.StatusCode, nil
	}
	if len(bytes.TrimSpace(responseBody)) == 0 {
		return resp.StatusCode, errors.New("Portico Cloud returned an empty JSON response")
	}
	if err := json.Unmarshal(responseBody, out); err != nil {
		return resp.StatusCode, err
	}
	return resp.StatusCode, nil
}

func signRequest(req *http.Request, body []byte, identity Identity) error {
	privateKey, err := identity.Private()
	if err != nil {
		return err
	}

	nonceBytes := make([]byte, 24)
	if _, err := rand.Read(nonceBytes); err != nil {
		return err
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
	timestamp := time.Now().UTC().Unix()

	message := canonicalMessage(req.Method, req.URL.Path, timestamp, nonce, body)
	signature := ed25519.Sign(privateKey, message)

	req.Header.Set("X-Portico-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Portico-Nonce", nonce)
	req.Header.Set("X-Portico-Signature", base64.RawURLEncoding.EncodeToString(signature))
	return nil
}

func canonicalMessage(method, path string, timestamp int64, nonce string, body []byte) []byte {
	bodyHash := sha256.Sum256(body)
	return []byte(strings.Join([]string{
		nodeSigningProtocol,
		strings.ToUpper(method),
		path,
		strconv.FormatInt(timestamp, 10),
		nonce,
		hex.EncodeToString(bodyHash[:]),
	}, "\n"))
}
