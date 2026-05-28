package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/web3gaoyutang/littlelight/server/internal/domain"
	"github.com/web3gaoyutang/littlelight/server/internal/repository"
	"github.com/web3gaoyutang/littlelight/server/internal/service"
)

func TestRoutesHealthAndReadiness(t *testing.T) {
	server := newTestServer()

	response := performRequest(t, server, http.MethodGet, "/healthz", nil)
	assertStatus(t, response, http.StatusOK)
	var health map[string]any
	decodeResponse(t, response, &health)
	if health["status"] != "ok" {
		t.Fatalf("expected healthy response, got %+v", health)
	}

	response = performRequest(t, server, http.MethodGet, "/readyz", nil)
	assertStatus(t, response, http.StatusOK)
	var ready map[string]any
	decodeResponse(t, response, &ready)
	if ready["status"] != "ok" {
		t.Fatalf("expected ready response, got %+v", ready)
	}
}

func TestRoutesWechatMockLoginAndProfileHeader(t *testing.T) {
	server := newTestServer()

	response := performRequest(t, server, http.MethodPost, "/api/v1/auth/wechat/mock", map[string]any{
		"code":     "unit-test-login",
		"nickName": "HTTP 测试老师",
	})
	assertStatus(t, response, http.StatusOK)

	var session domain.WechatSession
	decodeResponse(t, response, &session)
	if session.UserID == "" || session.SessionToken == "" || session.OpenID == "" {
		t.Fatalf("mock login returned incomplete session: %+v", session)
	}
	if session.Profile.Name != "HTTP 测试老师" {
		t.Fatalf("mock login should update profile name, got %+v", session.Profile)
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/me", nil, session.UserID)
	assertStatus(t, response, http.StatusOK)
	var profile domain.UserProfile
	decodeResponse(t, response, &profile)
	if profile.Name != session.Profile.Name {
		t.Fatalf("profile lookup should use logged-in user context, got %+v", profile)
	}
}

func TestRoutesBusinessFlowAndAIGenerationAudit(t *testing.T) {
	server := newTestServer()
	userID := domain.ID("00000000-0000-0000-0000-000000000001")

	day := "2026-05-27"
	response := performRequestWithUser(t, server, http.MethodPost, "/api/v1/courses", map[string]any{
		"title":     "HTTP 测试课程",
		"className": "三年级一班",
		"location":  "302",
		"weekday":   3,
		"startTime": "09:30",
		"endTime":   "10:15",
		"note":      "handler test",
	}, userID)
	assertStatus(t, response, http.StatusOK)
	var course domain.Course
	decodeResponse(t, response, &course)
	if course.ID == "" || course.Title != "HTTP 测试课程" {
		t.Fatalf("unexpected course response: %+v", course)
	}

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/reminders", map[string]any{
		"title":    "HTTP 测试待办",
		"category": "todo",
		"remindAt": "2026-05-27T17:20:00+08:00",
		"note":     "handler test",
	}, userID)
	assertStatus(t, response, http.StatusOK)
	var reminder domain.Reminder
	decodeResponse(t, response, &reminder)
	if reminder.ID == "" || reminder.Status != "pending" {
		t.Fatalf("unexpected reminder response: %+v", reminder)
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/dashboard?day="+day, nil, userID)
	assertStatus(t, response, http.StatusOK)
	var dashboard domain.DashboardSummary
	decodeResponse(t, response, &dashboard)
	if dashboard.TodayLabel != day || dashboard.CoursesCount == 0 || dashboard.RemindersCount == 0 {
		t.Fatalf("dashboard did not include created data: %+v", dashboard)
	}

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/parents", map[string]any{
		"studentName":        "HTTP 学生",
		"className":          "三年级一班",
		"parentName":         "HTTP 家长",
		"relationship":       "母亲",
		"contact":            "13800000000",
		"communicationStyle": "容易焦虑",
		"riskLevel":          "medium",
		"importantNotes":     "需要先共情",
		"nextAction":         "明天跟进",
	}, userID)
	assertStatus(t, response, http.StatusOK)
	var parent domain.ParentProfile
	decodeResponse(t, response, &parent)
	if parent.ID == "" || parent.StudentName != "HTTP 学生" {
		t.Fatalf("unexpected parent response: %+v", parent)
	}

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/parent-drafts", map[string]any{
		"issue":       "最近作业启动较慢",
		"parentStyle": "容易焦虑",
		"tone":        "温和",
		"studentName": parent.StudentName,
	}, userID)
	assertStatus(t, response, http.StatusOK)
	var drafts []domain.AIDraft
	decodeResponse(t, response, &drafts)
	if len(drafts) < 3 || drafts[0].Safety != "teacher_review_required" {
		t.Fatalf("unexpected AI drafts: %+v", drafts)
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/ai/generations?scenario=parent_drafts", nil, userID)
	assertStatus(t, response, http.StatusOK)
	var generations []domain.AIGeneration
	decodeResponse(t, response, &generations)
	if len(generations) == 0 || generations[0].Scenario != "parent_drafts" {
		t.Fatalf("expected AI generation audit record, got %+v", generations)
	}
}

func TestRoutesInvalidJSONAndNotFound(t *testing.T) {
	server := newTestServer()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/courses", bytes.NewBufferString("{"))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	assertStatus(t, response, http.StatusBadRequest)

	response = performRequest(t, server, http.MethodGet, "/api/v1/courses/missing-course", nil)
	assertStatus(t, response, http.StatusNotFound)
}

func TestRoutesMatchOpenAPIContract(t *testing.T) {
	server := newTestServer()
	routes, ok := server.(chi.Routes)
	if !ok {
		t.Fatalf("test server does not expose chi routes")
	}

	registered := map[string]bool{}
	if err := chi.Walk(routes, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		registered[method+" "+route] = true
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	documented := parseOpenAPIOperations(t)
	assertOperationSetEqual(t, "OpenAPI operation missing backend route", documented, registered)
	assertOperationSetEqual(t, "Backend route missing OpenAPI operation", registered, documented)
}

func newTestServer() http.Handler {
	store := repository.NewMemoryStore()
	ai := service.NewAIService()
	return NewServer(store, ai, nil, DependencyCheck{Name: "test", Check: func(context.Context) error { return nil }}).Routes()
}

func performRequest(t *testing.T, handler http.Handler, method string, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return performRequestWithUser(t, handler, method, path, body, "")
}

func performRequestWithUser(t *testing.T, handler http.Handler, method string, path string, body any, userID domain.ID) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(data)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	if userID != "" {
		request.Header.Set("X-User-ID", string(userID))
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if response.Code != expected {
		t.Fatalf("expected status %d, got %d body=%s", expected, response.Code, response.Body.String())
	}
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v body=%s", err, response.Body.String())
	}
}

func parseOpenAPIOperations(t *testing.T) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "openapi.yaml"))
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}
	operations := map[string]bool{}
	var currentPath string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "  /") && strings.HasSuffix(trimmed, ":") {
			currentPath = strings.TrimSuffix(trimmed, ":")
			continue
		}
		if currentPath == "" || !strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "      ") || !strings.HasSuffix(trimmed, ":") {
			continue
		}
		method := strings.ToUpper(strings.TrimSuffix(trimmed, ":"))
		switch method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
			operations[method+" "+currentPath] = true
		}
	}
	if len(operations) == 0 {
		t.Fatalf("no operations parsed from OpenAPI contract")
	}
	return operations
}

func assertOperationSetEqual(t *testing.T, message string, expected map[string]bool, actual map[string]bool) {
	t.Helper()
	missing := make([]string, 0)
	for operation := range expected {
		if !actual[operation] {
			missing = append(missing, operation)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("%s:\n%s", message, strings.Join(missing, "\n"))
	}
}
