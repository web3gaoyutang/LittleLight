package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
	"github.com/web3gaoyutang/littlelight/server/internal/repository"
)

const (
	demoUserID       = domain.ID("00000000-0000-0000-0000-000000000001")
	mockWechatOpenID = "mock-openid-littlelight-teacher"
)

type userContextKey struct{}
type sessionTokenHashContextKey struct{}

type AuthConfig struct {
	SessionSecret string
	WechatAppID   string
	WechatSecret  string
	AllowDevUser  bool
	AllowMockAuth bool
	CORSOrigins   []string
}

var (
	errMissingAuth = errors.New("missing auth token")
	errInvalidAuth = errors.New("invalid auth token")
	errExpiredAuth = errors.New("expired auth token")
)

func withUser(store repository.Store, config AuthConfig) func(http.Handler) http.Handler {
	secret := strings.TrimSpace(config.SessionSecret)
	if secret == "" {
		secret = "littlelight-local-session-secret"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, tokenHash, err := authenticatedUserID(r, store, secret, config.AllowDevUser)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey{}, userID)
			if tokenHash != "" {
				ctx = context.WithValue(ctx, sessionTokenHashContextKey{}, tokenHash)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func currentUserID(r *http.Request) domain.ID {
	if value, ok := r.Context().Value(userContextKey{}).(domain.ID); ok && value != "" {
		return value
	}
	panic("authenticated user missing from request context")
}

func currentSessionTokenHash(r *http.Request) string {
	value, _ := r.Context().Value(sessionTokenHashContextKey{}).(string)
	return value
}

func authenticatedUserID(r *http.Request, store repository.Store, secret string, allowDevUser bool) (domain.ID, string, error) {
	if token := bearerToken(r.Header.Get("Authorization")); token != "" {
		return authenticatedSessionToken(r, store, secret, token)
	}
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		return authenticatedSessionToken(r, store, secret, token)
	}
	if allowDevUser {
		userID := domain.ID(strings.TrimSpace(r.Header.Get("X-User-ID")))
		if userID == "" {
			userID = domain.ID(strings.TrimSpace(r.URL.Query().Get("userId")))
		}
		if userID != "" {
			if _, err := store.UserProfile(r.Context(), userID); err != nil {
				return "", "", errInvalidAuth
			}
			return userID, "", nil
		}
	}
	return "", "", errMissingAuth
}

func authenticatedSessionToken(r *http.Request, store repository.Store, secret string, token string) (domain.ID, string, error) {
	userID, err := verifySessionToken(token, secret)
	if err != nil {
		return "", "", err
	}
	tokenHash := sessionTokenHash(token)
	session, err := store.AuthSessionByTokenHash(r.Context(), tokenHash)
	if err != nil {
		return "", "", errInvalidAuth
	}
	if session.UserID != userID {
		return "", "", errInvalidAuth
	}
	return userID, tokenHash, nil
}

func bearerToken(value string) string {
	parts := strings.Fields(value)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func signSessionToken(userID domain.ID, secret string, expiresAt time.Time) (string, error) {
	if strings.TrimSpace(secret) == "" {
		secret = "littlelight-local-session-secret"
	}
	nonce, err := secureNonce()
	if err != nil {
		return "", err
	}
	payload := fmt.Sprintf("%s.%d.%s", userID, expiresAt.Unix(), nonce)
	signature := tokenSignature(payload, secret)
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "." + signature)), nil
}

func verifySessionToken(token string, secret string) (domain.ID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return "", errInvalidAuth
	}
	parts := strings.Split(string(raw), ".")
	if len(parts) != 4 {
		return "", errInvalidAuth
	}
	payload := strings.Join(parts[:3], ".")
	if !hmac.Equal([]byte(parts[3]), []byte(tokenSignature(payload, secret))) {
		return "", errInvalidAuth
	}
	expiresUnix, err := parseUnix(parts[1])
	if err != nil || expiresUnix <= time.Now().Unix() {
		return "", errExpiredAuth
	}
	userID := domain.ID(strings.TrimSpace(parts[0]))
	if userID == "" {
		return "", errInvalidAuth
	}
	return userID, nil
}

func secureNonce() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(data[:]), nil
}

func sessionTokenHash(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func tokenSignature(payload string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func parseUnix(value string) (int64, error) {
	var result int64
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return 0, err
	}
	return result, nil
}
