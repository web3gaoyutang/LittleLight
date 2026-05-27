package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

const demoUserID = domain.ID("00000000-0000-0000-0000-000000000001")

type userContextKey struct{}

func withUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := domain.ID(strings.TrimSpace(r.Header.Get("X-User-ID")))
		if userID == "" {
			userID = demoUserID
		}
		ctx := context.WithValue(r.Context(), userContextKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func currentUserID(r *http.Request) domain.ID {
	if value, ok := r.Context().Value(userContextKey{}).(domain.ID); ok && value != "" {
		return value
	}
	return demoUserID
}
