package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
	"github.com/web3gaoyutang/littlelight/server/internal/service"
	"nhooyr.io/websocket"
)

const dictationRateLimit = 120

func (s *Server) dictationStream(w http.ResponseWriter, r *http.Request) {
	userID, _, err := authenticatedUserID(r, s.store, s.auth.SessionSecret, s.auth.AllowDevUser)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
		return
	}
	if s.dictation == nil || !s.dictation.Configured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error":  "dictation unavailable",
			"detail": "讯飞语音听写服务尚未配置",
		})
		return
	}
	if !s.allowDictationRate(w, r, userID) {
		return
	}
	if !s.acquireDictationSession(userID) {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"error":  "dictation session already active",
			"detail": "当前账号已有语音听写正在进行",
		})
		return
	}
	defer s.releaseDictationSession(userID)

	options, err := dictationSessionOptionsFromRequest(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "validation failed", "detail": err.Error()})
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("dictation websocket accept failed: %v", err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "dictation session closed")

	if err := s.dictation.Run(r.Context(), conn, options); err != nil {
		if errors.Is(err, context.Canceled) {
			_ = conn.Close(websocket.StatusNormalClosure, "dictation canceled")
			return
		}
		log.Printf("dictation session failed user=%s: %v", userID, err)
		_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"type":"error","code":"DICTATION_FAILED","message":"语音听写暂时不可用，请稍后重试。"}`))
		_ = conn.Close(websocket.StatusInternalError, "dictation failed")
		return
	}
	_ = conn.Close(websocket.StatusNormalClosure, "dictation complete")
}

func dictationSessionOptionsFromRequest(r *http.Request) (service.DictationSessionOptions, error) {
	language := strings.TrimSpace(r.URL.Query().Get("language"))
	sampleRate := 16000
	if raw := strings.TrimSpace(r.URL.Query().Get("sampleRate")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return service.DictationSessionOptions{}, errors.New("sampleRate must be 16000 or 8000")
		}
		sampleRate = parsed
	}
	return service.DictationSessionOptions{Language: language, SampleRate: sampleRate}, nil
}

func (s *Server) acquireDictationSession(userID domain.ID) bool {
	options := s.dictation.Options()
	limit := options.MaxConcurrentPerUser
	if limit <= 0 {
		limit = 1
	}
	s.dictationMu.Lock()
	defer s.dictationMu.Unlock()
	if s.dictationUsers == nil {
		s.dictationUsers = map[domain.ID]int{}
	}
	if s.dictationUsers[userID] >= limit {
		return false
	}
	s.dictationUsers[userID]++
	return true
}

func (s *Server) releaseDictationSession(userID domain.ID) {
	s.dictationMu.Lock()
	defer s.dictationMu.Unlock()
	if s.dictationUsers == nil {
		return
	}
	count := s.dictationUsers[userID]
	if count <= 1 {
		delete(s.dictationUsers, userID)
		return
	}
	s.dictationUsers[userID] = count - 1
}

func (s *Server) allowDictationRate(w http.ResponseWriter, r *http.Request, userID domain.ID) bool {
	options := s.dictation.Options()
	limit := options.DailyLimitPerUser
	if limit <= 0 {
		limit = dictationRateLimit
	}
	key := "dictation:" + string(userID) + ":" + r.URL.Query().Get("language")
	ok, retryAfter := s.rateLimiter.Allow(key, limit, rateLimitWindow)
	if ok {
		return true
	}
	w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
	writeJSON(w, http.StatusTooManyRequests, map[string]any{
		"error":  "rate limited",
		"detail": "语音听写请求过于频繁，请稍后再试",
	})
	return false
}
