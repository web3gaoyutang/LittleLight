package httpapi

import (
	"net/http"
	"testing"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
	"github.com/web3gaoyutang/littlelight/server/internal/repository"
	"github.com/web3gaoyutang/littlelight/server/internal/service"
)

func TestDictationStreamRequiresConfiguredProvider(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())

	response := performRequestWithUser(t, server, http.MethodGet, "/api/v1/dictation/stream", nil, demoUserID)
	assertStatus(t, response, http.StatusServiceUnavailable)
}

func TestDictationStreamRequiresAuth(t *testing.T) {
	store := repository.NewMemoryStore()
	server := NewServer(store, service.NewAIService(service.AIOptions{Provider: "mock"}), nil)
	server.ConfigureAuth(localTestAuthConfig())
	server.ConfigureDictation(service.NewDictationService(service.DictationOptions{
		AppID:     "xf-app",
		APIKey:    "xf-key",
		APISecret: "xf-secret",
	}))

	response := performRequest(t, server.Routes(), http.MethodGet, "/api/v1/dictation/stream", nil)
	assertStatus(t, response, http.StatusUnauthorized)
}

func TestDictationSessionConcurrency(t *testing.T) {
	server := &Server{
		dictation: service.NewDictationService(service.DictationOptions{
			AppID:                "xf-app",
			APIKey:               "xf-key",
			APISecret:            "xf-secret",
			MaxConcurrentPerUser: 1,
		}),
		dictationUsers: map[domain.ID]int{},
	}
	userID := domain.ID("user-one")

	if !server.acquireDictationSession(userID) {
		t.Fatalf("first dictation session should be allowed")
	}
	if server.acquireDictationSession(userID) {
		t.Fatalf("second concurrent dictation session should be rejected")
	}
	server.releaseDictationSession(userID)
	if !server.acquireDictationSession(userID) {
		t.Fatalf("released dictation session should allow a new one")
	}
}
