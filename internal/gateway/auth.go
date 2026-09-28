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

// NewIntegratedOIDCVerifier is deliberately specific to the bundled ZITADEL
// deployment. ZITADEL DCR issues opaque bearer access tokens for this flow,
// so the resource server validates each token online through ZITADEL UserInfo.
// The backchannel URL is private to the Docker identity network; the public
// issuer is still advertised through protected-resource/OIDC metadata.
func NewIntegratedOIDCVerifier(userInfoURL, forwardedHost string) (mcpauth.TokenVerifier, error) {
	if strings.TrimSpace(userInfoURL) == "" {
		return nil, errors.New("integrated userinfo URL is required")
	}
	u, err := url.Parse(userInfoURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("integrated userinfo URL must be http(s)")
	}
	client := &http.Client{Timeout: 5 * time.Second}

	return func(ctx context.Context, raw string, _ *http.Request) (*mcpauth.TokenInfo, error) {
		if strings.TrimSpace(raw) == "" {
			return nil, mcpauth.ErrInvalidToken
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
		if err != nil {
			return nil, mcpauth.ErrInvalidToken
		}
		req.Header.Set("Authorization", "Bearer "+raw)
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

		var info struct {
			Subject string `json:"sub"`
		}
		dec := json.NewDecoder(io.LimitReader(resp.Body, 64<<10))
		if err := dec.Decode(&info); err != nil || strings.TrimSpace(info.Subject) == "" {
			return nil, mcpauth.ErrInvalidToken
		}

		// RequireBearerToken needs a non-expired TokenInfo. UserInfo validates
		// the opaque token online for every request, so this local expiration is
		// intentionally short and never replaces issuer-side revocation checks.
		return &mcpauth.TokenInfo{
			UserID:     info.Subject,
			Scopes:     []string{"openid"},
			Expiration: time.Now().Add(time.Minute),
		}, nil
	}, nil
}
