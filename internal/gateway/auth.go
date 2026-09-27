package gateway

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
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

func NewOIDCVerifier(ctx context.Context, issuer, audience string) (mcpauth.TokenVerifier, error) {
	if issuer == "" || audience == "" {
		return nil, errors.New("issuer and audience are required")
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: audience})
	return func(ctx context.Context, raw string, _ *http.Request) (*mcpauth.TokenInfo, error) {
		tok, err := verifier.Verify(ctx, raw)
		if err != nil {
			return nil, mcpauth.ErrInvalidToken
		}
		var claims struct {
			Subject     string   `json:"sub"`
			Scope       string   `json:"scope"`
			Permissions []string `json:"permissions"`
		}
		if err := tok.Claims(&claims); err != nil || claims.Subject == "" {
			return nil, mcpauth.ErrInvalidToken
		}
		scopes := append([]string{}, claims.Permissions...)
		scopes = append(scopes, strings.Fields(claims.Scope)...)
		return &mcpauth.TokenInfo{
			UserID: claims.Subject, Scopes: scopes, Expiration: tok.Expiry,
		}, nil
	}, nil
}
