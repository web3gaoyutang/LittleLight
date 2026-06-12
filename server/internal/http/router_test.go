package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

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
	server := newTestServerWithAuth(localTestAuthConfig())

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

func TestRoutesWechatMockLoginIsolatesUsersByOpenID(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())

	firstLogin := performRequest(t, server, http.MethodPost, "/api/v1/auth/wechat/mock", map[string]any{
		"code":     "teacher-a",
		"nickName": "A 老师",
	})
	assertStatus(t, firstLogin, http.StatusOK)
	var firstSession domain.WechatSession
	decodeResponse(t, firstLogin, &firstSession)

	secondLogin := performRequest(t, server, http.MethodPost, "/api/v1/auth/wechat/mock", map[string]any{
		"code":     "teacher-b",
		"nickName": "B 老师",
	})
	assertStatus(t, secondLogin, http.StatusOK)
	var secondSession domain.WechatSession
	decodeResponse(t, secondLogin, &secondSession)
	if firstSession.UserID == secondSession.UserID || firstSession.OpenID == secondSession.OpenID {
		t.Fatalf("expected isolated mock users, got first=%+v second=%+v", firstSession, secondSession)
	}

	updateFirst := performRequestWithBearer(t, server, http.MethodPut, "/api/v1/me", map[string]any{
		"name":           "A 老师改名",
		"school":         "A 学校",
		"stage":          "小学",
		"subject":        "语文",
		"proStatus":      "free",
		"reminderPolicy": "normal",
	}, firstSession.SessionToken)
	assertStatus(t, updateFirst, http.StatusOK)

	firstProfileResponse := performRequestWithBearer(t, server, http.MethodGet, "/api/v1/me", nil, firstSession.SessionToken)
	assertStatus(t, firstProfileResponse, http.StatusOK)
	var firstProfile domain.UserProfile
	decodeResponse(t, firstProfileResponse, &firstProfile)
	if firstProfile.Name != "A 老师改名" {
		t.Fatalf("first profile was not updated: %+v", firstProfile)
	}

	secondProfileResponse := performRequestWithBearer(t, server, http.MethodGet, "/api/v1/me", nil, secondSession.SessionToken)
	assertStatus(t, secondProfileResponse, http.StatusOK)
	var secondProfile domain.UserProfile
	decodeResponse(t, secondProfileResponse, &secondProfile)
	if secondProfile.Name != "B 老师" {
		t.Fatalf("second profile leaked first user update: %+v", secondProfile)
	}
}

func TestRoutesRequireSessionWhenDevHeaderDisabled(t *testing.T) {
	server := newTestServerWithAuth(AuthConfig{
		SessionSecret: "unit-secret",
		AllowDevUser:  false,
		AllowMockAuth: true,
	})

	response := performRequest(t, server, http.MethodGet, "/api/v1/me", nil)
	assertStatus(t, response, http.StatusUnauthorized)

	login := performRequest(t, server, http.MethodPost, "/api/v1/auth/wechat/mock", map[string]any{
		"code":     "unit-test-login",
		"nickName": "Token 老师",
	})
	assertStatus(t, login, http.StatusOK)
	var session domain.WechatSession
	decodeResponse(t, login, &session)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request.Header.Set("Authorization", "Bearer "+session.SessionToken)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	assertStatus(t, response, http.StatusOK)
}

func TestRoutesLogoutRevokesSession(t *testing.T) {
	server := newTestServerWithAuth(AuthConfig{
		SessionSecret: "unit-secret",
		AllowDevUser:  false,
		AllowMockAuth: true,
	})

	login := performRequest(t, server, http.MethodPost, "/api/v1/auth/wechat/mock", map[string]any{
		"code":     "logout-session",
		"nickName": "退出测试老师",
	})
	assertStatus(t, login, http.StatusOK)
	var session domain.WechatSession
	decodeResponse(t, login, &session)

	response := performRequestWithBearer(t, server, http.MethodGet, "/api/v1/me", nil, session.SessionToken)
	assertStatus(t, response, http.StatusOK)

	response = performRequestWithBearer(t, server, http.MethodPost, "/api/v1/auth/logout", nil, session.SessionToken)
	assertStatus(t, response, http.StatusOK)
	var result map[string]bool
	decodeResponse(t, response, &result)
	if !result["ok"] {
		t.Fatalf("expected logout ok response, got %+v", result)
	}

	response = performRequestWithBearer(t, server, http.MethodGet, "/api/v1/me", nil, session.SessionToken)
	assertStatus(t, response, http.StatusUnauthorized)
}

func TestRoutesRequireExplicitAuthByDefault(t *testing.T) {
	server := newTestServer()

	response := performRequest(t, server, http.MethodGet, "/api/v1/me", nil)
	assertStatus(t, response, http.StatusUnauthorized)

	response = performRequest(t, server, http.MethodPost, "/api/v1/auth/wechat/mock", map[string]any{
		"code": "default-mock-disabled",
	})
	assertStatus(t, response, http.StatusNotFound)
}

func TestRoutesDevAuthRequiresExplicitUserIDHeader(t *testing.T) {
	server := newTestServerWithAuth(AuthConfig{AllowDevUser: true})

	response := performRequest(t, server, http.MethodGet, "/api/v1/me", nil)
	assertStatus(t, response, http.StatusUnauthorized)

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/me", nil, demoUserID)
	assertStatus(t, response, http.StatusOK)
}

func TestRoutesDevAuthRejectsUnknownUserIDHeader(t *testing.T) {
	server := newTestServerWithAuth(AuthConfig{AllowDevUser: true, AllowMockAuth: true})
	unknownUserID := domain.ID("00000000-0000-0000-0000-000000000099")

	response := performRequestWithUser(t, server, http.MethodGet, "/api/v1/me", nil, unknownUserID)
	assertStatus(t, response, http.StatusUnauthorized)

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/courses", map[string]any{
		"title":     "不应创建",
		"className": "一班",
		"weekday":   1,
		"startTime": "09:00",
		"endTime":   "09:45",
	}, unknownUserID)
	assertStatus(t, response, http.StatusUnauthorized)

	login := performRequest(t, server, http.MethodPost, "/api/v1/auth/wechat/mock", map[string]any{
		"code":     "known-dev-header",
		"nickName": "已登录老师",
	})
	assertStatus(t, login, http.StatusOK)
	var session domain.WechatSession
	decodeResponse(t, login, &session)

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/me", nil, session.UserID)
	assertStatus(t, response, http.StatusOK)
}

func TestRoutesCORSAllowsConfiguredOriginsOnly(t *testing.T) {
	server := newTestServerWithAuth(AuthConfig{
		AllowDevUser: true,
		CORSOrigins:  []string{"https://h5.example.com", "https://admin.example.com"},
	})

	request, response := newJSONRequest(t, http.MethodGet, "/api/v1/me", nil)
	request.Header.Set("X-User-ID", string(demoUserID))
	request.Header.Set("Origin", "https://h5.example.com")
	server.ServeHTTP(response, request)
	assertStatus(t, response, http.StatusOK)
	if response.Header().Get("Access-Control-Allow-Origin") != "https://h5.example.com" {
		t.Fatalf("expected allowed CORS origin, got %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
	if !strings.Contains(response.Header().Values("Vary")[0], "Origin") {
		t.Fatalf("expected Vary Origin header, got %v", response.Header().Values("Vary"))
	}

	request, response = newJSONRequest(t, http.MethodGet, "/api/v1/me", nil)
	request.Header.Set("X-User-ID", string(demoUserID))
	request.Header.Set("Origin", "https://evil.example.com")
	server.ServeHTTP(response, request)
	assertStatus(t, response, http.StatusOK)
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("unexpected CORS origin for disallowed requester: %q", response.Header().Get("Access-Control-Allow-Origin"))
	}

	request = httptest.NewRequest(http.MethodOptions, "/api/v1/me", nil)
	request.Header.Set("Origin", "https://admin.example.com")
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	assertStatus(t, response, http.StatusNoContent)
	if response.Header().Get("Access-Control-Allow-Origin") != "https://admin.example.com" {
		t.Fatalf("expected preflight CORS origin, got %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestRoutesWechatLoginRequiresConfiguredWechat(t *testing.T) {
	server := newTestServer()

	response := performRequest(t, server, http.MethodPost, "/api/v1/auth/wechat", map[string]any{
		"code":     "wx-code",
		"nickName": "微信老师",
	})
	assertStatus(t, response, http.StatusServiceUnavailable)
}

func TestRoutesRateLimitsLoginAttemptsByClient(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())

	for i := 0; i < authLoginRateLimit; i++ {
		response := performRequest(t, server, http.MethodPost, "/api/v1/auth/wechat/mock", map[string]any{
			"code":     "limited-login",
			"nickName": "登录限流老师",
		})
		assertStatus(t, response, http.StatusOK)
	}

	response := performRequest(t, server, http.MethodPost, "/api/v1/auth/wechat/mock", map[string]any{
		"code":     "limited-login",
		"nickName": "登录限流老师",
	})
	assertStatus(t, response, http.StatusTooManyRequests)
	if response.Header().Get("Retry-After") == "" {
		t.Fatalf("rate limited responses should include Retry-After")
	}
}

func TestRoutesRejectInvalidLoginInput(t *testing.T) {
	wechatServer := newTestServerWithAuth(AuthConfig{AllowMockAuth: true})
	tests := []struct {
		name   string
		path   string
		body   map[string]any
		status int
	}{
		{
			name:   "wechat login requires code",
			path:   "/api/v1/auth/wechat",
			body:   map[string]any{"nickName": "微信老师"},
			status: http.StatusBadRequest,
		},
		{
			name:   "wechat login rejects long code before provider exchange",
			path:   "/api/v1/auth/wechat",
			body:   map[string]any{"code": strings.Repeat("x", 513)},
			status: http.StatusBadRequest,
		},
		{
			name:   "wechat login rejects long nickName",
			path:   "/api/v1/auth/wechat",
			body:   map[string]any{"code": "wx-code", "nickName": strings.Repeat("林", 81)},
			status: http.StatusBadRequest,
		},
		{
			name:   "wechat login rejects long avatarUrl",
			path:   "/api/v1/auth/wechat",
			body:   map[string]any{"code": "wx-code", "avatarUrl": "https://example.com/" + strings.Repeat("a", 500)},
			status: http.StatusBadRequest,
		},
		{
			name:   "mock login rejects long code",
			path:   "/api/v1/auth/wechat/mock",
			body:   map[string]any{"code": strings.Repeat("x", 513)},
			status: http.StatusBadRequest,
		},
		{
			name:   "mock login rejects long nickName",
			path:   "/api/v1/auth/wechat/mock",
			body:   map[string]any{"code": "dev-login", "nickName": strings.Repeat("林", 81)},
			status: http.StatusBadRequest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performRequest(t, wechatServer, http.MethodPost, test.path, test.body)
			assertStatus(t, response, test.status)
		})
	}
}

func TestRoutesRejectInvalidInput(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())

	tests := []struct {
		name   string
		path   string
		method string
		body   map[string]any
	}{
		{
			name:   "course missing fields and invalid clock",
			method: http.MethodPost,
			path:   "/api/v1/courses",
			body: map[string]any{
				"title":     "",
				"className": "三年级一班",
				"weekday":   9,
				"startTime": "9点",
				"endTime":   "10:15",
			},
		},
		{
			name:   "course end before start",
			method: http.MethodPost,
			path:   "/api/v1/courses",
			body: map[string]any{
				"title":     "时间倒挂",
				"className": "三年级一班",
				"weekday":   3,
				"startTime": "10:15",
				"endTime":   "09:30",
			},
		},
		{
			name:   "course note too long",
			method: http.MethodPost,
			path:   "/api/v1/courses",
			body: map[string]any{
				"title":     "备注过长",
				"className": "三年级一班",
				"weekday":   3,
				"startTime": "09:30",
				"endTime":   "10:15",
				"note":      strings.Repeat("长", 1001),
			},
		},
		{
			name:   "reminder status not allowed on create",
			method: http.MethodPost,
			path:   "/api/v1/reminders",
			body: map[string]any{
				"title":    "不允许创建时指定完成",
				"remindAt": "2026-05-27T17:20:00+08:00",
				"status":   "done",
			},
		},
		{
			name:   "reminder snooze requires until",
			method: http.MethodPost,
			path:   "/api/v1/reminders/reminder_lin/snooze",
			body:   map[string]any{},
		},
		{
			name:   "reminder snooze requires future until",
			method: http.MethodPost,
			path:   "/api/v1/reminders/reminder_lin/snooze",
			body: map[string]any{
				"until": "2000-01-01T10:00:00+08:00",
			},
		},
		{
			name:   "parent invalid risk",
			method: http.MethodPost,
			path:   "/api/v1/parents",
			body: map[string]any{
				"studentName":  "小林",
				"className":    "一班",
				"parentName":   "小林家长",
				"relationship": "家长",
				"riskLevel":    "critical",
			},
		},
		{
			name:   "communication summary too long",
			method: http.MethodPost,
			path:   "/api/v1/communication-records",
			body: map[string]any{
				"student":   "小林",
				"channel":   "微信",
				"reason":    "测试",
				"summary":   strings.Repeat("长", 2001),
				"riskLevel": "low",
			},
		},
		{
			name:   "favorite type invalid",
			method: http.MethodPost,
			path:   "/api/v1/me/favorites",
			body: map[string]any{
				"type":    "unknown",
				"title":   "标题",
				"content": "内容",
			},
		},
		{
			name:   "profile name too long",
			method: http.MethodPut,
			path:   "/api/v1/me",
			body: map[string]any{
				"name": strings.Repeat("林", 41),
			},
		},
		{
			name:   "ai action metadata too large",
			method: http.MethodPost,
			path:   "/api/v1/ai/generations/ai_generation_seed_1/actions",
			body: map[string]any{
				"action": "review_confirmed",
				"metadata": map[string]any{
					"detail": strings.Repeat("x", 501),
				},
			},
		},
		{
			name:   "ai action metadata too deep",
			method: http.MethodPost,
			path:   "/api/v1/ai/generations/ai_generation_seed_1/actions",
			body: map[string]any{
				"action": "review_confirmed",
				"metadata": map[string]any{
					"a": map[string]any{
						"b": map[string]any{
							"c": map[string]any{
								"d": map[string]any{
									"e": "too deep",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performRequestWithUser(t, server, test.method, test.path, test.body, demoUserID)
			assertStatus(t, response, http.StatusBadRequest)
		})
	}

	response := performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/praise", map[string]any{
		"content": "",
	}, demoUserID)
	assertStatus(t, response, http.StatusBadRequest)

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/parent-drafts", map[string]any{
		"issue": strings.Repeat("太长", 601),
	}, demoUserID)
	assertStatus(t, response, http.StatusBadRequest)
}

func TestRoutesRejectInvalidQueryInput(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := domain.ID("00000000-0000-0000-0000-000000000001")

	tests := []struct {
		name string
		path string
	}{
		{name: "dashboard invalid day", path: "/api/v1/dashboard?day=not-a-date"},
		{name: "dashboard impossible day", path: "/api/v1/dashboard?day=2026-99-99"},
		{name: "reminders invalid day", path: "/api/v1/reminders?day=2026-02-31"},
		{name: "courses weekday non integer", path: "/api/v1/courses?weekday=abc"},
		{name: "courses weekday out of range", path: "/api/v1/courses?weekday=9"},
		{name: "parents invalid limit", path: "/api/v1/parents?limit=abc"},
		{name: "parents limit too large", path: "/api/v1/parents?limit=101"},
		{name: "parents invalid offset", path: "/api/v1/parents?offset=-1"},
		{name: "parents query too long", path: "/api/v1/parents?q=" + strings.Repeat("a", 101)},
		{name: "favorites invalid type", path: "/api/v1/me/favorites?type=unknown"},
		{name: "ai generations invalid scenario", path: "/api/v1/ai/generations?scenario=unknown"},
		{name: "healing entries invalid type", path: "/api/v1/healing/entries?type=unknown"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performRequestWithUser(t, server, http.MethodGet, test.path, nil, userID)
			assertStatus(t, response, http.StatusBadRequest)
		})
	}
}

func TestRoutesRejectCrossUserReferences(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())

	loginA := performRequest(t, server, http.MethodPost, "/api/v1/auth/wechat/mock", map[string]any{
		"code":     "owner-a",
		"nickName": "Owner A",
	})
	assertStatus(t, loginA, http.StatusOK)
	var sessionA domain.WechatSession
	decodeResponse(t, loginA, &sessionA)

	loginB := performRequest(t, server, http.MethodPost, "/api/v1/auth/wechat/mock", map[string]any{
		"code":     "owner-b",
		"nickName": "Owner B",
	})
	assertStatus(t, loginB, http.StatusOK)
	var sessionB domain.WechatSession
	decodeResponse(t, loginB, &sessionB)

	parentResponse := performRequestWithBearer(t, server, http.MethodPost, "/api/v1/parents", map[string]any{
		"studentName":  "隔离学生",
		"className":    "一班",
		"parentName":   "隔离家长",
		"relationship": "家长",
		"riskLevel":    "low",
	}, sessionA.SessionToken)
	assertStatus(t, parentResponse, http.StatusOK)
	var parent domain.ParentProfile
	decodeResponse(t, parentResponse, &parent)

	response := performRequestWithBearer(t, server, http.MethodPost, "/api/v1/communication-records", map[string]any{
		"parentId":  parent.ID,
		"student":   "隔离学生",
		"channel":   "微信",
		"reason":    "测试",
		"summary":   "不应跨用户引用",
		"riskLevel": "low",
	}, sessionB.SessionToken)
	assertStatus(t, response, http.StatusBadRequest)

	response = performRequestWithBearer(t, server, http.MethodPost, "/api/v1/reminders", map[string]any{
		"title":    "跨用户提醒",
		"remindAt": "2026-05-27T17:20:00+08:00",
		"parentId": parent.ID,
	}, sessionB.SessionToken)
	assertStatus(t, response, http.StatusBadRequest)
}

func TestRoutesDeleteParentKeepsHistoricalCommunicationRecord(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := demoUserID

	parentResponse := performRequestWithUser(t, server, http.MethodPost, "/api/v1/parents", map[string]any{
		"studentName":  "历史接口学生",
		"className":    "一班",
		"parentName":   "历史接口家长",
		"relationship": "母亲",
		"riskLevel":    "low",
	}, userID)
	assertStatus(t, parentResponse, http.StatusOK)
	var parent domain.ParentProfile
	decodeResponse(t, parentResponse, &parent)

	recordResponse := performRequestWithUser(t, server, http.MethodPost, "/api/v1/communication-records", map[string]any{
		"parentId":  parent.ID,
		"student":   parent.StudentName,
		"channel":   "微信",
		"reason":    "历史接口沟通",
		"summary":   "家长删除后仍保留",
		"riskLevel": "low",
	}, userID)
	assertStatus(t, recordResponse, http.StatusOK)
	var record domain.CommunicationRecord
	decodeResponse(t, recordResponse, &record)
	if record.ParentID == nil || *record.ParentID != parent.ID {
		t.Fatalf("expected record to link parent before delete, got %+v", record)
	}

	deleteResponse := performRequestWithUser(t, server, http.MethodDelete, "/api/v1/parents/"+string(parent.ID), nil, userID)
	assertStatus(t, deleteResponse, http.StatusOK)

	detailResponse := performRequestWithUser(t, server, http.MethodGet, "/api/v1/communication-records/"+string(record.ID), nil, userID)
	assertStatus(t, detailResponse, http.StatusOK)
	var detail map[string]any
	decodeResponse(t, detailResponse, &detail)
	if _, ok := detail["parentId"]; ok {
		t.Fatalf("expected deleted parent link to be omitted, got %+v", detail)
	}
	if detail["student"] != parent.StudentName || detail["summary"] != "家长删除后仍保留" {
		t.Fatalf("expected historical record details to remain, got %+v", detail)
	}
}

func TestRoutesRejectCrossUserObjectAccess(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	owner := loginMockUser(t, server, "direct-owner", "直接对象归属老师")
	intruder := loginMockUser(t, server, "direct-intruder", "越权访问老师")

	courseResponse := performRequestWithBearer(t, server, http.MethodPost, "/api/v1/courses", map[string]any{
		"title":     "归属隔离课程",
		"className": "隔离班",
		"weekday":   2,
		"startTime": "09:00",
		"endTime":   "09:45",
	}, owner.SessionToken)
	assertStatus(t, courseResponse, http.StatusOK)
	var course domain.Course
	decodeResponse(t, courseResponse, &course)

	parentResponse := performRequestWithBearer(t, server, http.MethodPost, "/api/v1/parents", map[string]any{
		"studentName":  "归属隔离学生",
		"className":    "隔离班",
		"parentName":   "归属隔离家长",
		"relationship": "家长",
		"riskLevel":    "low",
	}, owner.SessionToken)
	assertStatus(t, parentResponse, http.StatusOK)
	var parent domain.ParentProfile
	decodeResponse(t, parentResponse, &parent)

	reminderResponse := performRequestWithBearer(t, server, http.MethodPost, "/api/v1/reminders", map[string]any{
		"title":    "归属隔离提醒",
		"remindAt": "2026-05-27T17:20:00+08:00",
		"parentId": parent.ID,
		"courseId": course.ID,
	}, owner.SessionToken)
	assertStatus(t, reminderResponse, http.StatusOK)
	var reminder domain.Reminder
	decodeResponse(t, reminderResponse, &reminder)

	recordResponse := performRequestWithBearer(t, server, http.MethodPost, "/api/v1/communication-records", map[string]any{
		"parentId":  parent.ID,
		"student":   parent.StudentName,
		"channel":   "微信",
		"reason":    "归属隔离沟通",
		"summary":   "这条沟通记录只能归属当前老师",
		"riskLevel": "low",
	}, owner.SessionToken)
	assertStatus(t, recordResponse, http.StatusOK)
	var record domain.CommunicationRecord
	decodeResponse(t, recordResponse, &record)

	healingResponse := performRequestWithBearer(t, server, http.MethodPost, "/api/v1/healing/entries", map[string]any{
		"type":    "treehole",
		"mood":    "steady",
		"content": "归属隔离树洞",
	}, owner.SessionToken)
	assertStatus(t, healingResponse, http.StatusOK)
	var healing domain.HealingEntry
	decodeResponse(t, healingResponse, &healing)

	aiResponse := performRequestWithBearer(t, server, http.MethodPost, "/api/v1/ai/parent-drafts", map[string]any{
		"issue":       "归属隔离 AI 草稿",
		"parentStyle": "直接",
		"tone":        "温和",
		"studentName": parent.StudentName,
	}, owner.SessionToken)
	assertStatus(t, aiResponse, http.StatusOK)
	var drafts []domain.AIDraft
	decodeResponse(t, aiResponse, &drafts)
	if len(drafts) == 0 || drafts[0].GenerationID == "" {
		t.Fatalf("expected owner AI generation id, got %+v", drafts)
	}

	favoriteResponse := performRequestWithBearer(t, server, http.MethodPost, "/api/v1/me/favorites", map[string]any{
		"type":    "communication_template",
		"title":   "归属隔离收藏",
		"content": "只属于创建老师",
	}, owner.SessionToken)
	assertStatus(t, favoriteResponse, http.StatusOK)
	var favorite domain.Favorite
	decodeResponse(t, favoriteResponse, &favorite)

	tests := []struct {
		name   string
		method string
		path   string
		body   map[string]any
	}{
		{
			name:   "get course",
			method: http.MethodGet,
			path:   "/api/v1/courses/" + string(course.ID),
		},
		{
			name:   "update course",
			method: http.MethodPut,
			path:   "/api/v1/courses/" + string(course.ID),
			body: map[string]any{
				"title":     "越权改课",
				"className": "隔离班",
				"weekday":   2,
				"startTime": "09:00",
				"endTime":   "09:45",
			},
		},
		{
			name:   "delete course",
			method: http.MethodDelete,
			path:   "/api/v1/courses/" + string(course.ID),
		},
		{
			name:   "get reminder",
			method: http.MethodGet,
			path:   "/api/v1/reminders/" + string(reminder.ID),
		},
		{
			name:   "update reminder",
			method: http.MethodPut,
			path:   "/api/v1/reminders/" + string(reminder.ID),
			body: map[string]any{
				"title":    "越权改提醒",
				"remindAt": "2026-05-27T18:20:00+08:00",
			},
		},
		{
			name:   "complete reminder",
			method: http.MethodPost,
			path:   "/api/v1/reminders/" + string(reminder.ID) + "/complete",
		},
		{
			name:   "snooze reminder",
			method: http.MethodPost,
			path:   "/api/v1/reminders/" + string(reminder.ID) + "/snooze",
			body: map[string]any{
				"until": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			name:   "delete reminder",
			method: http.MethodDelete,
			path:   "/api/v1/reminders/" + string(reminder.ID),
		},
		{
			name:   "get parent",
			method: http.MethodGet,
			path:   "/api/v1/parents/" + string(parent.ID),
		},
		{
			name:   "update parent",
			method: http.MethodPut,
			path:   "/api/v1/parents/" + string(parent.ID),
			body: map[string]any{
				"studentName":  "越权学生",
				"className":    "隔离班",
				"parentName":   "越权家长",
				"relationship": "家长",
				"riskLevel":    "low",
			},
		},
		{
			name:   "delete parent",
			method: http.MethodDelete,
			path:   "/api/v1/parents/" + string(parent.ID),
		},
		{
			name:   "get communication record",
			method: http.MethodGet,
			path:   "/api/v1/communication-records/" + string(record.ID),
		},
		{
			name:   "update communication record",
			method: http.MethodPut,
			path:   "/api/v1/communication-records/" + string(record.ID),
			body: map[string]any{
				"student":   parent.StudentName,
				"channel":   "微信",
				"reason":    "越权沟通",
				"summary":   "不应改到别人的沟通记录",
				"riskLevel": "low",
			},
		},
		{
			name:   "complete communication follow-up",
			method: http.MethodPost,
			path:   "/api/v1/communication-records/" + string(record.ID) + "/complete-follow-up",
		},
		{
			name:   "delete communication record",
			method: http.MethodDelete,
			path:   "/api/v1/communication-records/" + string(record.ID),
		},
		{
			name:   "get healing entry",
			method: http.MethodGet,
			path:   "/api/v1/healing/entries/" + string(healing.ID),
		},
		{
			name:   "delete healing entry",
			method: http.MethodDelete,
			path:   "/api/v1/healing/entries/" + string(healing.ID),
		},
		{
			name:   "get ai generation",
			method: http.MethodGet,
			path:   "/api/v1/ai/generations/" + string(drafts[0].GenerationID),
		},
		{
			name:   "delete ai generation",
			method: http.MethodDelete,
			path:   "/api/v1/ai/generations/" + string(drafts[0].GenerationID),
		},
		{
			name:   "append ai action",
			method: http.MethodPost,
			path:   "/api/v1/ai/generations/" + string(drafts[0].GenerationID) + "/actions",
			body: map[string]any{
				"action":  "copy",
				"draftId": drafts[0].ID,
			},
		},
		{
			name:   "delete favorite",
			method: http.MethodDelete,
			path:   "/api/v1/me/favorites/" + string(favorite.ID),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performRequestWithBearer(t, server, test.method, test.path, test.body, intruder.SessionToken)
			assertStatus(t, response, http.StatusNotFound)
		})
	}

	response := performRequestWithBearer(t, server, http.MethodGet, "/api/v1/courses?weekday=2", nil, intruder.SessionToken)
	assertStatus(t, response, http.StatusOK)
	var courses []domain.Course
	decodeResponse(t, response, &courses)
	for _, item := range courses {
		if item.ID == course.ID {
			t.Fatalf("intruder course list leaked owner course: %+v", courses)
		}
	}

	response = performRequestWithBearer(t, server, http.MethodGet, "/api/v1/parents?q=%E5%BD%92%E5%B1%9E%E9%9A%94%E7%A6%BB", nil, intruder.SessionToken)
	assertStatus(t, response, http.StatusOK)
	var parents ListResponse[domain.ParentProfile]
	decodeResponse(t, response, &parents)
	if len(parents.Items) != 0 {
		t.Fatalf("intruder parent search leaked owner data: %+v", parents)
	}

	response = performRequestWithBearer(t, server, http.MethodGet, "/api/v1/communication-records?q=%E5%BD%92%E5%B1%9E%E9%9A%94%E7%A6%BB", nil, intruder.SessionToken)
	assertStatus(t, response, http.StatusOK)
	var records ListResponse[domain.CommunicationRecord]
	decodeResponse(t, response, &records)
	if len(records.Items) != 0 {
		t.Fatalf("intruder communication search leaked owner data: %+v", records)
	}

	response = performRequestWithBearer(t, server, http.MethodGet, "/api/v1/healing/entries?q=%E5%BD%92%E5%B1%9E%E9%9A%94%E7%A6%BB", nil, intruder.SessionToken)
	assertStatus(t, response, http.StatusOK)
	var healingEntries ListResponse[domain.HealingEntry]
	decodeResponse(t, response, &healingEntries)
	if len(healingEntries.Items) != 0 {
		t.Fatalf("intruder healing search leaked owner data: %+v", healingEntries)
	}

	response = performRequestWithBearer(t, server, http.MethodGet, "/api/v1/ai/generations?q=%E5%BD%92%E5%B1%9E%E9%9A%94%E7%A6%BB", nil, intruder.SessionToken)
	assertStatus(t, response, http.StatusOK)
	var generations ListResponse[domain.AIGeneration]
	decodeResponse(t, response, &generations)
	if len(generations.Items) != 0 {
		t.Fatalf("intruder AI generation search leaked owner data: %+v", generations)
	}

	response = performRequestWithBearer(t, server, http.MethodGet, "/api/v1/courses/"+string(course.ID), nil, owner.SessionToken)
	assertStatus(t, response, http.StatusOK)
	response = performRequestWithBearer(t, server, http.MethodGet, "/api/v1/parents/"+string(parent.ID), nil, owner.SessionToken)
	assertStatus(t, response, http.StatusOK)
	response = performRequestWithBearer(t, server, http.MethodGet, "/api/v1/communication-records/"+string(record.ID), nil, owner.SessionToken)
	assertStatus(t, response, http.StatusOK)
}

func TestRoutesDeleteAIGenerationRemovesSensitiveRecordAndActions(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := domain.ID("00000000-0000-0000-0000-000000000001")

	response := performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/praise", map[string]any{
		"persona": "温柔前辈",
		"content": "这是一条需要清理的敏感 AI 输入",
		"mood":    "tired",
	}, userID)
	assertStatus(t, response, http.StatusOK)
	var draft domain.AIDraft
	decodeResponse(t, response, &draft)
	if draft.GenerationID == "" {
		t.Fatalf("expected generation id on AI draft: %+v", draft)
	}

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(draft.GenerationID)+"/actions", map[string]any{
		"action":  "copy",
		"draftId": draft.ID,
		"note":    "删除前的使用动作",
	}, userID)
	assertStatus(t, response, http.StatusOK)

	response = performRequestWithUser(t, server, http.MethodDelete, "/api/v1/ai/generations/"+string(draft.GenerationID), nil, userID)
	assertStatus(t, response, http.StatusOK)
	var deleted map[string]bool
	decodeResponse(t, response, &deleted)
	if !deleted["ok"] {
		t.Fatalf("expected ok delete response, got %+v", deleted)
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/ai/generations/"+string(draft.GenerationID), nil, userID)
	assertStatus(t, response, http.StatusNotFound)

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(draft.GenerationID)+"/actions", map[string]any{
		"action":  "copy",
		"draftId": draft.ID,
	}, userID)
	assertStatus(t, response, http.StatusNotFound)
}

func TestRoutesDeleteCurrentAccountRemovesOwnedDataAndSession(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	session := loginMockUser(t, server, "delete-account", "待删除老师")

	response := performRequestWithBearer(t, server, http.MethodPost, "/api/v1/courses", map[string]any{
		"title":     "删除账号课程",
		"className": "一班",
		"weekday":   1,
		"startTime": "09:00",
		"endTime":   "09:45",
	}, session.SessionToken)
	assertStatus(t, response, http.StatusOK)

	response = performRequestWithBearer(t, server, http.MethodPost, "/api/v1/me/favorites", map[string]any{
		"type":    "communication_template",
		"title":   "删除账号收藏",
		"content": "账号删除后应消失",
	}, session.SessionToken)
	assertStatus(t, response, http.StatusOK)

	response = performRequestWithBearer(t, server, http.MethodDelete, "/api/v1/me", nil, session.SessionToken)
	assertStatus(t, response, http.StatusOK)
	var deleted map[string]bool
	decodeResponse(t, response, &deleted)
	if !deleted["ok"] {
		t.Fatalf("expected ok delete response, got %+v", deleted)
	}

	response = performRequestWithBearer(t, server, http.MethodGet, "/api/v1/me", nil, session.SessionToken)
	assertStatus(t, response, http.StatusUnauthorized)

	recreated := loginMockUser(t, server, "delete-account", "重新登录老师")
	if recreated.UserID == session.UserID {
		t.Fatalf("expected deleted account to be recreated as a new user")
	}
	response = performRequestWithBearer(t, server, http.MethodGet, "/api/v1/courses?weekday=1", nil, recreated.SessionToken)
	assertStatus(t, response, http.StatusOK)
	var courses []domain.Course
	decodeResponse(t, response, &courses)
	if len(courses) != 0 {
		t.Fatalf("expected recreated account to have no deleted courses, got %+v", courses)
	}

	response = performRequestWithBearer(t, server, http.MethodGet, "/api/v1/me/favorites?q=%E5%88%A0%E9%99%A4%E8%B4%A6%E5%8F%B7", nil, recreated.SessionToken)
	assertStatus(t, response, http.StatusOK)
	var favorites ListResponse[domain.Favorite]
	decodeResponse(t, response, &favorites)
	if len(favorites.Items) != 0 {
		t.Fatalf("expected recreated account to have no deleted favorites, got %+v", favorites)
	}
}

func TestRoutesExportCurrentAccountDataIsScopedToSessionUser(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	owner := loginMockUser(t, server, "export-owner", "导出老师")
	other := loginMockUser(t, server, "export-other", "其他老师")

	response := performRequestWithBearer(t, server, http.MethodPost, "/api/v1/courses", map[string]any{
		"title":     "导出账号课程",
		"className": "导出班",
		"weekday":   1,
		"startTime": "09:00",
		"endTime":   "09:45",
	}, owner.SessionToken)
	assertStatus(t, response, http.StatusOK)

	response = performRequestWithBearer(t, server, http.MethodPost, "/api/v1/me/favorites", map[string]any{
		"type":    "communication_template",
		"title":   "导出账号收藏",
		"content": "只有导出老师能看到",
	}, owner.SessionToken)
	assertStatus(t, response, http.StatusOK)

	response = performRequestWithBearer(t, server, http.MethodPost, "/api/v1/ai/praise", map[string]any{
		"content": "导出账号 AI 输入",
	}, owner.SessionToken)
	assertStatus(t, response, http.StatusOK)
	var praise domain.AIDraft
	decodeResponse(t, response, &praise)

	response = performRequestWithBearer(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(praise.GenerationID)+"/actions", map[string]any{
		"action":  "review_confirmed",
		"draftId": praise.ID,
		"note":    "导出前复核",
	}, owner.SessionToken)
	assertStatus(t, response, http.StatusOK)

	response = performRequestWithBearer(t, server, http.MethodPost, "/api/v1/courses", map[string]any{
		"title":     "其他账号课程",
		"className": "其他班",
		"weekday":   1,
		"startTime": "10:00",
		"endTime":   "10:45",
	}, other.SessionToken)
	assertStatus(t, response, http.StatusOK)

	response = performRequestWithBearer(t, server, http.MethodGet, "/api/v1/me/export", nil, owner.SessionToken)
	assertStatus(t, response, http.StatusOK)
	if !strings.Contains(response.Header().Get("Content-Disposition"), "littlelight-account-export.json") {
		t.Fatalf("expected download content disposition, got %q", response.Header().Get("Content-Disposition"))
	}
	var data domain.AccountExport
	decodeResponse(t, response, &data)
	if data.Profile.ID != owner.UserID || data.Profile.Name != "导出老师" || data.ExportedAt.IsZero() {
		t.Fatalf("unexpected export profile: %+v", data)
	}
	if !exportHasCourse(data, "导出账号课程") || exportHasCourse(data, "其他账号课程") {
		t.Fatalf("export should include only owner courses, got %+v", data.Courses)
	}
	if !exportHasFavorite(data, "导出账号收藏") {
		t.Fatalf("expected owner favorite in export, got %+v", data.Favorites)
	}
	if len(data.AIGenerations) == 0 || len(data.AIGenerations[0].Actions) == 0 {
		t.Fatalf("expected ai generations with actions in export, got %+v", data.AIGenerations)
	}

	response = performRequestWithBearer(t, server, http.MethodGet, "/api/v1/me/export", nil, other.SessionToken)
	assertStatus(t, response, http.StatusOK)
	var otherData domain.AccountExport
	decodeResponse(t, response, &otherData)
	if exportHasCourse(otherData, "导出账号课程") || !exportHasCourse(otherData, "其他账号课程") {
		t.Fatalf("other export should not leak owner data, got %+v", otherData.Courses)
	}
}

func TestRoutesProductBoundaryEndpoints(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := domain.ID("00000000-0000-0000-0000-000000000001")

	response := performRequestWithUser(t, server, http.MethodGet, "/api/v1/billing/entitlements", nil, userID)
	assertStatus(t, response, http.StatusOK)
	var entitlements domain.Entitlements
	decodeResponse(t, response, &entitlements)
	if entitlements.Plan == "" || entitlements.CheckoutStatus == "" || len(entitlements.Features) == 0 {
		t.Fatalf("unexpected entitlements: %+v", entitlements)
	}

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/billing/checkout", map[string]any{
		"plan":     "yearly",
		"provider": "wechat",
		"mock":     true,
	}, userID)
	assertStatus(t, response, http.StatusOK)
	var checkout domain.BillingCheckoutResult
	decodeResponse(t, response, &checkout)
	if checkout.Status != "paid" || checkout.Plan != "yearly" || checkout.Provider != "wechat" || checkout.AmountCents != 20000 {
		t.Fatalf("checkout should return a simulated paid WeChat order, got %+v", checkout)
	}
	if checkout.Profile.ProStatus != "pro" || checkout.Entitlements.Plan != "pro" {
		t.Fatalf("checkout should activate Pro entitlements, got %+v", checkout)
	}

	monthlyLogin := performRequest(t, server, http.MethodPost, "/api/v1/auth/wechat/mock", map[string]any{
		"code":     "monthly-checkout",
		"nickName": "月费测试老师",
	})
	assertStatus(t, monthlyLogin, http.StatusOK)
	var monthlySession domain.WechatSession
	decodeResponse(t, monthlyLogin, &monthlySession)
	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/billing/checkout", map[string]any{
		"plan":     "monthly",
		"provider": "wechat",
		"mock":     true,
	}, monthlySession.UserID)
	assertStatus(t, response, http.StatusOK)
	decodeResponse(t, response, &checkout)
	if checkout.Plan != "monthly" || checkout.AmountCents != 2000 || checkout.Profile.ProStatus != "pro" {
		t.Fatalf("monthly checkout should activate Pro at 20 yuan, got %+v", checkout)
	}

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/billing/checkout", map[string]any{
		"plan":     "weekly",
		"provider": "wechat",
		"mock":     true,
	}, userID)
	assertStatus(t, response, http.StatusBadRequest)

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/billing/checkout", map[string]any{
		"plan":     "monthly",
		"provider": "wechat",
		"mock":     false,
	}, userID)
	assertStatus(t, response, http.StatusBadRequest)

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/billing/entitlements", nil, userID)
	assertStatus(t, response, http.StatusOK)
	decodeResponse(t, response, &entitlements)
	if entitlements.Plan != "pro" || entitlements.Status != "pro" {
		t.Fatalf("entitlements should stay Pro after simulated checkout, got %+v", entitlements)
	}

	response = performRequestWithUser(t, server, http.MethodPut, "/api/v1/notifications/settings", map[string]any{
		"reminderPolicy": "normal",
	}, userID)
	assertStatus(t, response, http.StatusOK)
	var settings domain.NotificationSettings
	decodeResponse(t, response, &settings)
	if settings.ReminderPolicy != "normal" || settings.ProviderStatus == "" {
		t.Fatalf("unexpected notification settings: %+v", settings)
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/sync/status", nil, userID)
	assertStatus(t, response, http.StatusOK)
	var syncStatus domain.SyncStatus
	decodeResponse(t, response, &syncStatus)
	if syncStatus.CloudSyncStatus == "" || syncStatus.ObjectStorageStatus == "" {
		t.Fatalf("unexpected sync status: %+v", syncStatus)
	}
}

func TestRoutesBusinessFlowAndAIGenerationAudit(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
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
	if drafts[0].GenerationID == "" {
		t.Fatalf("expected AI drafts to include generation id: %+v", drafts)
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/ai/generations?scenario=parent_drafts", nil, userID)
	assertStatus(t, response, http.StatusOK)
	var generations ListResponse[domain.AIGeneration]
	decodeResponse(t, response, &generations)
	if len(generations.Items) == 0 || generations.Items[0].Scenario != "parent_drafts" {
		t.Fatalf("expected AI generation audit record, got %+v", generations)
	}

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(drafts[0].GenerationID)+"/actions", map[string]any{
		"action":  "copy",
		"draftId": drafts[0].ID,
		"note":    "尝试绕过复核直接复制",
		"metadata": map[string]any{
			"surface": "route-test",
		},
	}, userID)
	assertStatus(t, response, http.StatusBadRequest)

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(drafts[0].GenerationID)+"/actions", map[string]any{
		"action":  "review_confirmed",
		"draftId": drafts[0].ID,
		"note":    "教师确认复核",
		"metadata": map[string]any{
			"surface": "route-test",
			"safety":  drafts[0].Safety,
		},
	}, userID)
	assertStatus(t, response, http.StatusOK)
	var reviewAction domain.AIAction
	decodeResponse(t, response, &reviewAction)
	if reviewAction.Action != "review_confirmed" || reviewAction.DraftID != string(drafts[0].ID) {
		t.Fatalf("unexpected AI review action response: %+v", reviewAction)
	}

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(drafts[0].GenerationID)+"/actions", map[string]any{
		"action":  "copy",
		"draftId": drafts[0].ID,
		"note":    "复核后复制",
		"metadata": map[string]any{
			"surface": "route-test",
		},
	}, userID)
	assertStatus(t, response, http.StatusOK)
	var action domain.AIAction
	decodeResponse(t, response, &action)
	if action.ID == "" || action.GenerationID != drafts[0].GenerationID || action.Action != "copy" {
		t.Fatalf("unexpected AI action response: %+v", action)
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/ai/generations/"+string(drafts[0].GenerationID), nil, userID)
	assertStatus(t, response, http.StatusOK)
	var generationDetail domain.AIGeneration
	decodeResponse(t, response, &generationDetail)
	if len(generationDetail.Actions) != 2 || generationDetail.Actions[0].Action != "review_confirmed" || generationDetail.Actions[1].Action != "copy" {
		t.Fatalf("expected AI action history on generation detail, got %+v", generationDetail)
	}
	detailDrafts, ok := generationDetail.Output["drafts"].([]any)
	if !ok || len(detailDrafts) == 0 {
		t.Fatalf("expected generation detail output drafts, got %+v", generationDetail.Output)
	}
	detailDraft, ok := detailDrafts[0].(map[string]any)
	if !ok {
		t.Fatalf("expected generation detail draft object, got %+v", detailDrafts[0])
	}
	if detailDraft["generationId"] != string(drafts[0].GenerationID) || detailDraft["safetyLevel"] == "" || detailDraft["safetyReason"] == "" {
		t.Fatalf("expected persisted draft metadata, got %+v", detailDraft)
	}

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(drafts[0].GenerationID)+"/actions", map[string]any{
		"action": "send_without_review",
	}, userID)
	assertStatus(t, response, http.StatusBadRequest)
}

func TestRoutesRateLimitsAIGenerationPerUser(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userA := domain.ID("00000000-0000-0000-0000-000000000001")
	userB := loginMockUser(t, server, "ai-rate-user-b", "频控用户B").UserID

	for i := 0; i < aiGenerationRateLimit; i++ {
		response := performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/praise", map[string]any{
			"content": "今天完成了一次课堂复盘",
		}, userA)
		assertStatus(t, response, http.StatusOK)
	}

	response := performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/praise", map[string]any{
		"content": "这一次请求应该被频控拦截",
	}, userA)
	assertStatus(t, response, http.StatusTooManyRequests)
	if response.Header().Get("Retry-After") == "" {
		t.Fatalf("rate limited AI responses should include Retry-After")
	}

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/praise", map[string]any{
		"content": "另一个用户不应该被同一个限流桶影响",
	}, userB)
	assertStatus(t, response, http.StatusOK)
}

func TestRoutesAIReviewGateBlocksOutputUseUntilConfirmed(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := domain.ID("00000000-0000-0000-0000-000000000001")

	response := performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/praise", map[string]any{
		"content": "我想轻生，不知道还能不能撑下去",
	}, userID)
	assertStatus(t, response, http.StatusOK)
	var draft domain.AIDraft
	decodeResponse(t, response, &draft)
	if draft.GenerationID == "" || draft.Safety != "crisis_support_required" || !draft.ReviewRequired {
		t.Fatalf("expected high-risk draft with generation id, got %+v", draft)
	}

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(draft.GenerationID)+"/actions", map[string]any{
		"action":  "save_healing",
		"draftId": draft.ID,
		"note":    "直接保存高风险内容",
	}, userID)
	assertStatus(t, response, http.StatusBadRequest)

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(draft.GenerationID)+"/actions", map[string]any{
		"action":  "review_confirmed",
		"draftId": draft.ID,
		"note":    "教师确认安全复核",
		"metadata": map[string]any{
			"surface":       "route-test",
			"safety":        draft.Safety,
			"safetyLevel":   draft.SafetyLevel,
			"safetySignals": draft.SafetySignals,
		},
	}, userID)
	assertStatus(t, response, http.StatusOK)

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(draft.GenerationID)+"/actions", map[string]any{
		"action":  "save_healing",
		"draftId": draft.ID,
		"note":    "复核后保存安全提示记录",
	}, userID)
	assertStatus(t, response, http.StatusOK)
	var action domain.AIAction
	decodeResponse(t, response, &action)
	if action.Action != "save_healing" || action.GenerationID != draft.GenerationID {
		t.Fatalf("unexpected reviewed save action: %+v", action)
	}
}

func TestRoutesAIReviewGateRequiresMatchingDraft(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := domain.ID("00000000-0000-0000-0000-000000000001")

	response := performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/parent-drafts", map[string]any{
		"issue":       "家长询问孩子最近课堂专注度下降，希望老师给出建议",
		"parentStyle": "容易焦虑",
		"tone":        "温和",
	}, userID)
	assertStatus(t, response, http.StatusOK)
	var drafts []domain.AIDraft
	decodeResponse(t, response, &drafts)
	if len(drafts) < 2 || drafts[0].GenerationID == "" || drafts[0].ID == drafts[1].ID {
		t.Fatalf("expected multiple identified drafts, got %+v", drafts)
	}
	generationID := drafts[0].GenerationID

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(generationID)+"/actions", map[string]any{
		"action": "review_confirmed",
		"note":   "缺少草稿 ID 的复核不应解锁任何输出",
	}, userID)
	assertStatus(t, response, http.StatusBadRequest)

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(generationID)+"/actions", map[string]any{
		"action":  "review_confirmed",
		"draftId": "not-in-generation",
		"note":    "不存在的草稿 ID 不应被接受",
	}, userID)
	assertStatus(t, response, http.StatusBadRequest)

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(generationID)+"/actions", map[string]any{
		"action":  "review_confirmed",
		"draftId": drafts[0].ID,
		"note":    "只复核第一条草稿",
	}, userID)
	assertStatus(t, response, http.StatusOK)

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(generationID)+"/actions", map[string]any{
		"action":  "copy",
		"draftId": drafts[1].ID,
		"note":    "第二条草稿尚未复核，不应被第一条解锁",
	}, userID)
	assertStatus(t, response, http.StatusBadRequest)

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/"+string(generationID)+"/actions", map[string]any{
		"action":  "copy",
		"draftId": drafts[0].ID,
		"note":    "第一条草稿复核后可以使用",
	}, userID)
	assertStatus(t, response, http.StatusOK)
}

func TestRoutesAIActionAllowsLegacySingleOutputWithDraftID(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := domain.ID("00000000-0000-0000-0000-000000000001")

	response := performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/ai_generation_seed_1/actions", map[string]any{
		"action":  "copy",
		"draftId": "legacy-single-output",
		"note":    "历史单条输出允许用非空草稿标识写审计",
	}, userID)
	assertStatus(t, response, http.StatusOK)

	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/ai/generations/ai_generation_seed_1/actions", map[string]any{
		"action": "copy",
		"note":   "历史输出也不允许空草稿标识",
	}, userID)
	assertStatus(t, response, http.StatusBadRequest)
}

func TestRoutesInvalidJSONAndNotFound(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())

	request := httptest.NewRequest(http.MethodPost, "/api/v1/courses", bytes.NewBufferString("{"))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-ID", string(demoUserID))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	assertStatus(t, response, http.StatusBadRequest)

	request = httptest.NewRequest(http.MethodPost, "/api/v1/courses", bytes.NewBufferString(`{"title":"未知字段","className":"三年级一班","weekday":3,"startTime":"09:30","endTime":"10:15","unexpected":"nope"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-ID", string(demoUserID))
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	assertStatus(t, response, http.StatusBadRequest)
	if !strings.Contains(response.Body.String(), "unknown field") {
		t.Fatalf("expected unknown field error, got %s", response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/courses", bytes.NewBufferString(`{"title":"尾随 JSON","className":"三年级一班","weekday":3,"startTime":"09:30","endTime":"10:15"} {}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-ID", string(demoUserID))
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	assertStatus(t, response, http.StatusBadRequest)

	request = httptest.NewRequest(http.MethodPost, "/api/v1/ai/praise", bytes.NewBufferString(`{"content":"`+strings.Repeat("很长", 600000)+`"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-ID", string(demoUserID))
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	assertStatus(t, response, http.StatusBadRequest)

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/courses/missing-course", nil, demoUserID)
	assertStatus(t, response, http.StatusNotFound)
}

func TestRoutesListSearchAndPagination(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := domain.ID("00000000-0000-0000-0000-000000000001")

	response := performRequestWithUser(t, server, http.MethodGet, "/api/v1/parents?q=%E9%99%88%E5%AD%90%E9%BB%98&limit=1", nil, userID)
	assertStatus(t, response, http.StatusOK)
	var parents ListResponse[domain.ParentProfile]
	decodeResponse(t, response, &parents)
	if len(parents.Items) != 1 || parents.Items[0].StudentName != "陈子默" {
		t.Fatalf("unexpected searched parents: %+v", parents)
	}
	if parents.PageInfo.Limit != 1 || parents.PageInfo.Count != 1 || parents.PageInfo.Offset != 0 || parents.PageInfo.HasMore {
		t.Fatalf("unexpected parent pageInfo: %+v", parents.PageInfo)
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/parents?limit=1&offset=0", nil, userID)
	assertStatus(t, response, http.StatusOK)
	decodeResponse(t, response, &parents)
	if len(parents.Items) != 1 || !parents.PageInfo.HasMore || parents.PageInfo.NextOffset != 1 {
		t.Fatalf("expected precise parent hasMore metadata, got %+v", parents)
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/ai/generations?scenario=praise&limit=1&offset=0", nil, userID)
	assertStatus(t, response, http.StatusOK)
	var generations ListResponse[domain.AIGeneration]
	decodeResponse(t, response, &generations)
	if len(generations.Items) != 1 || generations.Items[0].Scenario != "praise" {
		t.Fatalf("unexpected paged ai generations: %+v", generations)
	}
	if generations.PageInfo.Limit != 1 || generations.PageInfo.Count != 1 || generations.PageInfo.Offset != 0 {
		t.Fatalf("unexpected ai generation pageInfo: %+v", generations.PageInfo)
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/me/favorites?q=%E5%BF%99%E7%A2%8C&limit=1&offset=0", nil, userID)
	assertStatus(t, response, http.StatusOK)
	var favorites ListResponse[domain.Favorite]
	decodeResponse(t, response, &favorites)
	if len(favorites.Items) != 1 || favorites.Items[0].Title != "忙碌后恢复" {
		t.Fatalf("unexpected searched favorites: %+v", favorites)
	}
	if favorites.PageInfo.Limit != 1 || favorites.PageInfo.Count != 1 || favorites.PageInfo.Offset != 0 {
		t.Fatalf("unexpected favorite pageInfo: %+v", favorites.PageInfo)
	}
}

func TestRoutesSnoozeReminderUsesExplicitFutureTime(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := domain.ID("00000000-0000-0000-0000-000000000001")
	until := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)

	response := performRequestWithUser(t, server, http.MethodPost, "/api/v1/reminders/reminder_lin/snooze", map[string]any{
		"until": until.Format(time.RFC3339),
	}, userID)
	assertStatus(t, response, http.StatusOK)

	var reminder domain.Reminder
	decodeResponse(t, response, &reminder)
	if reminder.Status != "snoozed" {
		t.Fatalf("expected reminder to be snoozed, got %+v", reminder)
	}
	if !reminder.RemindAt.UTC().Equal(until) {
		t.Fatalf("expected reminder until %s, got %s", until, reminder.RemindAt.UTC())
	}
}

func TestRoutesDashboardCacheInvalidatesCurrentUserOnly(t *testing.T) {
	dashboardCache := newFakeDashboardCache()
	server := newTestServerWithDashboardCache(localTestAuthConfig(), dashboardCache)
	ctx := context.Background()
	day := "2026-05-27"
	parsedDay, err := time.Parse("2006-01-02", day)
	if err != nil {
		t.Fatalf("parse day: %v", err)
	}
	otherDay := parsedDay.AddDate(0, 0, 1)
	userA := demoUserID
	userB := loginMockUser(t, server, "dashboard-cache-user-b", "缓存用户B").UserID

	dashboardCache.Set(ctx, userA, parsedDay, domain.DashboardSummary{TodayLabel: "stale-a", CoursesCount: 999})
	dashboardCache.Set(ctx, userA, otherDay, domain.DashboardSummary{TodayLabel: "stale-a-other", CoursesCount: 888})
	dashboardCache.Set(ctx, userB, parsedDay, domain.DashboardSummary{TodayLabel: "cached-b", CoursesCount: 7})

	response := performRequestWithUser(t, server, http.MethodPost, "/api/v1/courses", map[string]any{
		"title":     "缓存失效课程",
		"className": "三年级一班",
		"weekday":   3,
		"startTime": "09:30",
		"endTime":   "10:15",
	}, userA)
	assertStatus(t, response, http.StatusOK)

	if len(dashboardCache.invalidatedUsers) != 1 || dashboardCache.invalidatedUsers[0] != userA {
		t.Fatalf("expected current user invalidation, got %+v", dashboardCache.invalidatedUsers)
	}
	if _, ok := dashboardCache.Get(ctx, userA, parsedDay); ok {
		t.Fatalf("expected current user's cached dashboard for selected day to be invalidated")
	}
	if _, ok := dashboardCache.Get(ctx, userA, otherDay); ok {
		t.Fatalf("expected all current user dashboard days to be invalidated")
	}
	if _, ok := dashboardCache.Get(ctx, userB, parsedDay); !ok {
		t.Fatalf("expected other user's cached dashboard to remain")
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/dashboard?day="+day, nil, userB)
	assertStatus(t, response, http.StatusOK)
	var cachedForOtherUser domain.DashboardSummary
	decodeResponse(t, response, &cachedForOtherUser)
	if cachedForOtherUser.TodayLabel != "cached-b" || cachedForOtherUser.CoursesCount != 7 {
		t.Fatalf("expected other user cache hit, got %+v", cachedForOtherUser)
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/dashboard?day="+day, nil, userA)
	assertStatus(t, response, http.StatusOK)
	var refreshedForCurrentUser domain.DashboardSummary
	decodeResponse(t, response, &refreshedForCurrentUser)
	if refreshedForCurrentUser.TodayLabel == "stale-a" || refreshedForCurrentUser.CoursesCount == 999 {
		t.Fatalf("expected current user dashboard to be recomputed, got %+v", refreshedForCurrentUser)
	}

	dashboardCache.Set(ctx, userA, parsedDay, domain.DashboardSummary{TodayLabel: "stale-after-record", FollowUpsCount: 99})
	dashboardCache.Set(ctx, userB, parsedDay, domain.DashboardSummary{TodayLabel: "cached-b-after-record", CoursesCount: 8})
	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/communication-records", map[string]any{
		"parentId":   "parent_lin",
		"student":    "林晓晓",
		"channel":    "微信",
		"reason":     "缓存失效跟进",
		"summary":    "沟通记录会影响首页待跟进统计",
		"riskLevel":  "medium",
		"followUpAt": parsedDay.Format(time.RFC3339),
	}, userA)
	assertStatus(t, response, http.StatusOK)
	if len(dashboardCache.invalidatedUsers) != 2 || dashboardCache.invalidatedUsers[1] != userA {
		t.Fatalf("expected communication record to invalidate current user dashboard, got %+v", dashboardCache.invalidatedUsers)
	}
	if _, ok := dashboardCache.Get(ctx, userA, parsedDay); ok {
		t.Fatalf("expected current user's dashboard cache to be invalidated after communication record change")
	}
	if cached, ok := dashboardCache.Get(ctx, userB, parsedDay); !ok || cached.TodayLabel != "cached-b-after-record" {
		t.Fatalf("expected other user's dashboard cache to remain after communication record change, got %+v ok=%v", cached, ok)
	}

	var createdRecord domain.CommunicationRecord
	decodeResponse(t, response, &createdRecord)
	dashboardCache.Set(ctx, userA, parsedDay, domain.DashboardSummary{TodayLabel: "stale-after-follow-up", FollowUpsCount: 1})
	dashboardCache.Set(ctx, userB, parsedDay, domain.DashboardSummary{TodayLabel: "cached-b-after-follow-up", CoursesCount: 9})
	response = performRequestWithUser(t, server, http.MethodPost, "/api/v1/communication-records/"+string(createdRecord.ID)+"/complete-follow-up", nil, userA)
	assertStatus(t, response, http.StatusOK)
	var completedRecord domain.CommunicationRecord
	decodeResponse(t, response, &completedRecord)
	if completedRecord.FollowUpStatus != "done" || completedRecord.FollowedUpAt == nil {
		t.Fatalf("expected completed follow-up response, got %+v", completedRecord)
	}
	if len(dashboardCache.invalidatedUsers) != 3 || dashboardCache.invalidatedUsers[2] != userA {
		t.Fatalf("expected follow-up completion to invalidate current user dashboard, got %+v", dashboardCache.invalidatedUsers)
	}
	if _, ok := dashboardCache.Get(ctx, userA, parsedDay); ok {
		t.Fatalf("expected current user's dashboard cache to be invalidated after follow-up completion")
	}
	if cached, ok := dashboardCache.Get(ctx, userB, parsedDay); !ok || cached.TodayLabel != "cached-b-after-follow-up" {
		t.Fatalf("expected other user's dashboard cache to remain after follow-up completion, got %+v ok=%v", cached, ok)
	}
}

func TestRoutesDashboardCacheInvalidatesReminderDays(t *testing.T) {
	dashboardCache := newFakeDashboardCache()
	server := newTestServerWithDashboardCache(localTestAuthConfig(), dashboardCache)
	ctx := context.Background()
	userID := demoUserID
	day := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	nextDay := day.AddDate(0, 0, 1)

	dashboardCache.Set(ctx, userID, day, domain.DashboardSummary{TodayLabel: "stale-day"})
	dashboardCache.Set(ctx, userID, nextDay, domain.DashboardSummary{TodayLabel: "stale-next-day"})
	response := performRequestWithUser(t, server, http.MethodPost, "/api/v1/reminders", map[string]any{
		"title":    "按天失效提醒",
		"remindAt": day.Add(9 * time.Hour).Format(time.RFC3339),
	}, userID)
	assertStatus(t, response, http.StatusOK)
	var reminder domain.Reminder
	decodeResponse(t, response, &reminder)

	if len(dashboardCache.invalidatedUsers) != 0 {
		t.Fatalf("expected reminder create to avoid user-wide invalidation, got %+v", dashboardCache.invalidatedUsers)
	}
	if len(dashboardCache.invalidatedDays) != 1 || dashboardCache.invalidatedDays[0] != dashboardCache.key(userID, day) {
		t.Fatalf("expected reminder create to invalidate selected day, got %+v", dashboardCache.invalidatedDays)
	}
	if _, ok := dashboardCache.Get(ctx, userID, nextDay); !ok {
		t.Fatalf("expected other cached dashboard days to remain after reminder create")
	}

	dashboardCache.Set(ctx, userID, day, domain.DashboardSummary{TodayLabel: "stale-day-after-update"})
	response = performRequestWithUser(t, server, http.MethodPut, "/api/v1/reminders/"+string(reminder.ID), map[string]any{
		"title":    "跨天提醒",
		"remindAt": nextDay.Add(10 * time.Hour).Format(time.RFC3339),
		"status":   "pending",
	}, userID)
	assertStatus(t, response, http.StatusOK)

	if len(dashboardCache.invalidatedUsers) != 0 {
		t.Fatalf("expected reminder update to avoid user-wide invalidation, got %+v", dashboardCache.invalidatedUsers)
	}
	if len(dashboardCache.invalidatedDays) != 3 {
		t.Fatalf("expected create plus old/new day invalidations, got %+v", dashboardCache.invalidatedDays)
	}
	if dashboardCache.invalidatedDays[1] != dashboardCache.key(userID, day) || dashboardCache.invalidatedDays[2] != dashboardCache.key(userID, nextDay) {
		t.Fatalf("expected reminder update to invalidate old and new days, got %+v", dashboardCache.invalidatedDays)
	}
}

func TestRoutesImportTemplateDownloads(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := domain.ID("00000000-0000-0000-0000-000000000001")

	response := performRequestWithUser(t, server, http.MethodGet, "/api/v1/courses/imports/template", nil, userID)
	assertStatus(t, response, http.StatusOK)
	if !strings.Contains(response.Header().Get("Content-Type"), "text/csv") || !strings.Contains(response.Body.String(), "课程名称") {
		t.Fatalf("unexpected course template response: headers=%v body=%s", response.Header(), response.Body.String())
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/parents/imports/template", nil, userID)
	assertStatus(t, response, http.StatusOK)
	if !strings.Contains(response.Header().Get("Content-Disposition"), "littlelight-parent-template.csv") || !strings.Contains(response.Body.String(), "学生姓名") {
		t.Fatalf("unexpected parent template response: headers=%v body=%s", response.Header(), response.Body.String())
	}
}

func TestRoutesImportPreviewDoesNotPersistRows(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := domain.ID("00000000-0000-0000-0000-000000000001")
	body, contentType := multipartBody(t, "file", "courses.csv", "课程名称,班级,星期,开始时间,结束时间\n预览课程,一班,三,09:00,09:45\n")

	request := httptest.NewRequest(http.MethodPost, "/api/v1/courses/imports?preview=true", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("X-User-ID", string(userID))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	assertStatus(t, response, http.StatusOK)
	var result service.ImportResult
	decodeResponse(t, response, &result)
	if result.Imported != 0 || len(result.Preview) != 1 || result.Preview[0].Status != "ready" {
		t.Fatalf("unexpected preview result: %+v", result)
	}

	response = performRequestWithUser(t, server, http.MethodGet, "/api/v1/courses?weekday=3", nil, userID)
	assertStatus(t, response, http.StatusOK)
	var courses []domain.Course
	decodeResponse(t, response, &courses)
	for _, course := range courses {
		if course.Title == "预览课程" {
			t.Fatalf("preview import should not persist course: %+v", courses)
		}
	}
}

func TestRoutesImportPreviewMarksDuplicateRows(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := domain.ID("00000000-0000-0000-0000-000000000001")
	body, contentType := multipartBody(t, "file", "courses.csv", "课程名称,班级,星期,开始时间,结束时间\n重复课程,一班,三,09:00,09:45\n重复课程,一班,三,09:00,09:45\n")

	request := httptest.NewRequest(http.MethodPost, "/api/v1/courses/imports?preview=true", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("X-User-ID", string(userID))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	assertStatus(t, response, http.StatusOK)
	var result service.ImportResult
	decodeResponse(t, response, &result)
	if result.Skipped != 1 || len(result.Preview) != 2 || result.Preview[1].Status != "duplicate" {
		t.Fatalf("expected duplicate row in preview, got %+v", result)
	}
}

func TestRoutesImportPreviewKeepsDuplicateRowNumbersAfterInvalidRows(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := domain.ID("00000000-0000-0000-0000-000000000001")
	body, contentType := multipartBody(t, "file", "courses.csv", "课程名称,班级,星期,开始时间,结束时间\n,一班,三,09:00,09:45\n重复课程,一班,三,09:00,09:45\n重复课程,一班,三,09:00,09:45\n")

	request := httptest.NewRequest(http.MethodPost, "/api/v1/courses/imports?preview=true", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("X-User-ID", string(userID))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	assertStatus(t, response, http.StatusOK)
	var result service.ImportResult
	decodeResponse(t, response, &result)
	if result.Skipped != 2 || len(result.Preview) != 3 {
		t.Fatalf("expected invalid and duplicate rows in preview, got %+v", result)
	}
	if result.Preview[0].Row != 2 || result.Preview[0].Status != "invalid" {
		t.Fatalf("expected first data row to be invalid, got %+v", result.Preview)
	}
	if result.Preview[2].Row != 4 || result.Preview[2].Status != "duplicate" {
		t.Fatalf("expected duplicate to keep original row 4, got %+v", result.Preview)
	}
	if result.FailureCSV == "" || !strings.Contains(result.FailureCSV, "第 4 行与本次文件中的其他课程重复") {
		t.Fatalf("expected failure csv to include duplicate row reason, got %+v", result)
	}
}

func TestRoutesImportRejectsInvalidUploads(t *testing.T) {
	server := newTestServerWithAuth(localTestAuthConfig())
	userID := domain.ID("00000000-0000-0000-0000-000000000001")
	unsupported := multipartBodyBytes(t, "file", "parents.txt", []byte("not,a,supported,file\n"))
	empty := multipartBodyBytes(t, "file", "parents.csv", nil)
	tooLarge := multipartBodyBytes(t, "file", "parents.csv", []byte(strings.Repeat("x", 5<<20)))

	tests := []struct {
		name        string
		body        *bytes.Buffer
		contentType string
	}{
		{
			name:        "unsupported file type",
			body:        unsupported.body,
			contentType: unsupported.contentType,
		},
		{
			name:        "empty file",
			body:        empty.body,
			contentType: empty.contentType,
		},
		{
			name:        "too large file",
			body:        tooLarge.body,
			contentType: tooLarge.contentType,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/parents/imports?preview=true", test.body)
			request.Header.Set("Content-Type", test.contentType)
			request.Header.Set("X-User-ID", string(userID))
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			assertStatus(t, response, http.StatusBadRequest)
		})
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/parents/imports?preview=true", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-User-ID", string(userID))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	assertStatus(t, response, http.StatusBadRequest)
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

func newTestServerWithAuth(config AuthConfig) http.Handler {
	store := repository.NewMemoryStore()
	ai := service.NewAIService()
	server := NewServer(store, ai, nil, DependencyCheck{Name: "test", Check: func(context.Context) error { return nil }})
	server.ConfigureAuth(config)
	return server.Routes()
}

func newTestServerWithDashboardCache(config AuthConfig, dashboardCache dashboardCache) http.Handler {
	store := repository.NewMemoryStore()
	ai := service.NewAIService()
	server := &Server{
		store:          store,
		ai:             ai,
		dashboardCache: dashboardCache,
		readiness: []DependencyCheck{{
			Name:  "test",
			Check: func(context.Context) error { return nil },
		}},
	}
	server.ConfigureAuth(config)
	return server.Routes()
}

type fakeDashboardCache struct {
	entries          map[string]domain.DashboardSummary
	invalidatedUsers []domain.ID
	invalidatedDays  []string
}

func newFakeDashboardCache() *fakeDashboardCache {
	return &fakeDashboardCache{entries: map[string]domain.DashboardSummary{}}
}

func (c *fakeDashboardCache) Get(ctx context.Context, userID domain.ID, day time.Time) (domain.DashboardSummary, bool) {
	data, ok := c.entries[c.key(userID, day)]
	return data, ok
}

func (c *fakeDashboardCache) Set(ctx context.Context, userID domain.ID, day time.Time, summary domain.DashboardSummary) {
	c.entries[c.key(userID, day)] = summary
}

func (c *fakeDashboardCache) Invalidate(ctx context.Context, userID domain.ID, day time.Time) {
	c.invalidatedDays = append(c.invalidatedDays, c.key(userID, day))
	delete(c.entries, c.key(userID, day))
}

func (c *fakeDashboardCache) InvalidateUser(ctx context.Context, userID domain.ID) {
	c.invalidatedUsers = append(c.invalidatedUsers, userID)
	prefix := string(userID) + "|"
	for key := range c.entries {
		if strings.HasPrefix(key, prefix) {
			delete(c.entries, key)
		}
	}
}

func (c *fakeDashboardCache) key(userID domain.ID, day time.Time) string {
	return string(userID) + "|" + day.Format("2006-01-02")
}

func localTestAuthConfig() AuthConfig {
	return AuthConfig{
		AllowDevUser:  true,
		AllowMockAuth: true,
	}
}

func performRequest(t *testing.T, handler http.Handler, method string, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return performRequestWithUser(t, handler, method, path, body, "")
}

func performRequestWithUser(t *testing.T, handler http.Handler, method string, path string, body any, userID domain.ID) *httptest.ResponseRecorder {
	t.Helper()
	request, response := newJSONRequest(t, method, path, body)
	if userID != "" {
		request.Header.Set("X-User-ID", string(userID))
	}
	handler.ServeHTTP(response, request)
	return response
}

func performRequestWithBearer(t *testing.T, handler http.Handler, method string, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	request, response := newJSONRequest(t, method, path, body)
	request.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(response, request)
	return response
}

func newJSONRequest(t *testing.T, method string, path string, body any) (*http.Request, *httptest.ResponseRecorder) {
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
	return request, httptest.NewRecorder()
}

func loginMockUser(t *testing.T, handler http.Handler, code string, nickName string) domain.WechatSession {
	t.Helper()
	response := performRequest(t, handler, http.MethodPost, "/api/v1/auth/wechat/mock", map[string]any{
		"code":     code,
		"nickName": nickName,
	})
	assertStatus(t, response, http.StatusOK)
	var session domain.WechatSession
	decodeResponse(t, response, &session)
	if session.UserID == "" || session.SessionToken == "" {
		t.Fatalf("mock login returned incomplete session: %+v", session)
	}
	return session
}

func multipartBody(t *testing.T, field string, filename string, content string) (*bytes.Buffer, string) {
	t.Helper()
	result := multipartBodyBytes(t, field, filename, []byte(content))
	return result.body, result.contentType
}

type multipartResult struct {
	body        *bytes.Buffer
	contentType string
}

func multipartBodyBytes(t *testing.T, field string, filename string, content []byte) multipartResult {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return multipartResult{body: &body, contentType: writer.FormDataContentType()}
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

func exportHasCourse(data domain.AccountExport, title string) bool {
	for _, course := range data.Courses {
		if course.Title == title {
			return true
		}
	}
	return false
}

func exportHasFavorite(data domain.AccountExport, title string) bool {
	for _, favorite := range data.Favorites {
		if favorite.Title == title {
			return true
		}
	}
	return false
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
