package gateway

import (
	"context"
	"crypto/subtle"
	"errors"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

type AuthConfig struct {
	Mode                 string
	StaticToken          string
	StaticSubject        string
	Verifier             mcpauth.TokenVerifier
	RequiredScopes       []string
	ResourceMetadataURL  string
	ResourceIdentifier   string
	AuthorizationServers []string
	InstanceID           string
	InstanceName         string
	MaxConcurrentRequests int
}

func (a AuthConfig) ProtectedResourceMetadataHandler() http.Handler {
	if a.ResourceIdentifier == "" || len(a.AuthorizationServers) == 0 {
		return nil
	}
	return mcpauth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:               a.ResourceIdentifier,
		AuthorizationServers:   a.AuthorizationServers,
		ScopesSupported:        a.RequiredScopes,
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "MCP VPS Agent Gateway",
	})
}

func (a AuthConfig) Wrap(next http.Handler) http.Handler {
	if a.Mode == "" || a.Mode == "none" {
		subject := a.StaticSubject
		if subject == "" {
			subject = "local-dev"
		}
		return subjectMiddleware(next, subject)
	}

	verifier := a.Verifier
	if verifier == nil && a.Mode == "static" {
		verifier = staticVerifier(a.StaticToken, a.StaticSubject)
	}
	if verifier == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "authentication verifier is not configured", http.StatusServiceUnavailable)
		})
	}

	protected := mcpauth.RequireBearerToken(verifier, &mcpauth.RequireBearerTokenOptions{
		Scopes:              a.RequiredScopes,
		ResourceMetadataURL: a.ResourceMetadataURL,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := mcpauth.TokenInfoFromContext(r.Context())
		if info == nil || info.UserID == "" {
			http.Error(w, "authenticated token has no subject", http.StatusUnauthorized)
			return
		}
		subjectMiddleware(next, info.UserID).ServeHTTP(w, r)
	}))
	return protected
}

func subjectMiddleware(next http.Handler, subject string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), subjectKey{}, subject)))
	})
}

func staticVerifier(token, subject string) mcpauth.TokenVerifier {
	if subject == "" {
		subject = "static-user"
	}
	return func(_ context.Context, got string, _ *http.Request) (*mcpauth.TokenInfo, error) {
		if token == "" || len(got) != len(token) || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			return nil, mcpauth.ErrInvalidToken
		}
		return &mcpauth.TokenInfo{
			UserID: subject, Scopes: []string{"read", "write"},
			Expiration: time.Now().Add(24 * time.Hour),
		}, nil
	}
}

// NewIntegratedOIDCVerifier validates opaque ZITADEL access tokens through a
// private RFC 7662 introspection backchannel. The introspection client belongs
// to a dedicated resource project. ZITADEL itself rejects tokens whose aud
// does not contain that client or project, and we independently require the
// expected issuer, resource-project audience and advertised scopes.
func NewIntegratedOIDCVerifier(introspectionURL, forwardedHost, clientID, clientSecret, expectedIssuer, expectedAudience string) (mcpauth.TokenVerifier, error) {
	if strings.TrimSpace(introspectionURL) == "" {
		return nil, errors.New("integrated introspection URL is required")
	}
	u, err := url.Parse(introspectionURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("integrated introspection URL must be http(s)")
	}
	if clientID == "" || clientSecret == "" {
		return nil, errors.New("integrated introspection client credentials are required")
	}
	expectedIssuer = strings.TrimRight(strings.TrimSpace(expectedIssuer), "/")
	if expectedIssuer == "" {
		return nil, errors.New("integrated expected issuer is required")
	}
	if strings.TrimSpace(expectedAudience) == "" {
		return nil, errors.New("integrated expected audience is required")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	return func(ctx context.Context, raw string, _ *http.Request) (*mcpauth.TokenInfo, error) {
		if strings.TrimSpace(raw) == "" {
			return nil, mcpauth.ErrInvalidToken
		}

		form := url.Values{"token": []string{raw}}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, introspectionURL, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, mcpauth.ErrInvalidToken
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(clientID, clientSecret)
		if forwardedHost != "" {
			req.Host = forwardedHost
			req.Header.Set("X-Forwarded-Proto", "https")
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, mcpauth.ErrInvalidToken
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			return nil, mcpauth.ErrInvalidToken
		}

		var token struct {
			Active     bool            `json:"active"`
			Subject    string          `json:"sub"`
			Scope      string          `json:"scope"`
			Expiration int64           `json:"exp"`
			Issuer     string          `json:"iss"`
			Audience   json.RawMessage `json:"aud"`
		}
		dec := json.NewDecoder(io.LimitReader(resp.Body, 64<<10))
		if err := dec.Decode(&token); err != nil {
			return nil, mcpauth.ErrInvalidToken
		}
		if !token.Active || strings.TrimSpace(token.Subject) == "" {
			return nil, mcpauth.ErrInvalidToken
		}
		if strings.TrimRight(token.Issuer, "/") != expectedIssuer {
			return nil, mcpauth.ErrInvalidToken
		}
		if token.Expiration <= time.Now().Unix() {
			return nil, mcpauth.ErrInvalidToken
		}
		if !audienceContains(token.Audience, expectedAudience) {
			return nil, mcpauth.ErrInvalidToken
		}

		return &mcpauth.TokenInfo{
			UserID:     token.Subject,
			Scopes:     strings.Fields(token.Scope),
			Expiration: time.Unix(token.Expiration, 0),
		}, nil
	}, nil
}

func audienceContains(raw json.RawMessage, expected string) bool {
	if len(raw) == 0 {
		return false
	}
	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		return one == expected
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err != nil {
		return false
	}
	for _, candidate := range many {
		if candidate == expected {
			return true
		}
	}
	return false
}
