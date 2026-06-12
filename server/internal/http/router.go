package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/web3gaoyutang/littlelight/server/internal/domain"
	"github.com/web3gaoyutang/littlelight/server/internal/platform/cache"
	"github.com/web3gaoyutang/littlelight/server/internal/repository"
	"github.com/web3gaoyutang/littlelight/server/internal/service"
)

const maxJSONBodyBytes = 1 << 20
const maxAIActionMetadataBytes = 4 << 10
const maxAIActionMetadataDepth = 4
const maxAIActionMetadataArrayItems = 20
const maxAIActionMetadataStringRunes = 500
const maxSearchQueryRunes = 100
const authLoginRateLimit = 10
const aiGenerationRateLimit = 20
const rateLimitWindow = time.Minute

type Server struct {
	store          repository.Store
	ai             *service.AIService
	dashboardCache dashboardCache
	readiness      []DependencyCheck
	auth           AuthConfig
	rateLimiter    *fixedWindowLimiter
}

type fixedWindowLimiter struct {
	mu      sync.Mutex
	buckets map[string]rateBucket
	now     func() time.Time
}

type rateBucket struct {
	count int
	reset time.Time
}

type dashboardCache interface {
	Get(context.Context, domain.ID, time.Time) (domain.DashboardSummary, bool)
	Set(context.Context, domain.ID, time.Time, domain.DashboardSummary)
	Invalidate(context.Context, domain.ID, time.Time)
	InvalidateUser(context.Context, domain.ID)
}

type DependencyCheck struct {
	Name  string
	Check func(context.Context) error
}

type PageInfo struct {
	Limit      int  `json:"limit"`
	Offset     int  `json:"offset"`
	Count      int  `json:"count"`
	NextOffset int  `json:"nextOffset,omitempty"`
	HasMore    bool `json:"hasMore"`
}

type ListResponse[T any] struct {
	Items    []T      `json:"items"`
	PageInfo PageInfo `json:"pageInfo"`
}

func NewServer(store repository.Store, ai *service.AIService, dashboardCache *cache.DashboardCache, readiness ...DependencyCheck) *Server {
	return &Server{
		store:          store,
		ai:             ai,
		dashboardCache: dashboardCache,
		readiness:      readiness,
		rateLimiter:    newFixedWindowLimiter(time.Now),
	}
}

func (s *Server) ConfigureAuth(config AuthConfig) {
	s.auth = config
}

func newFixedWindowLimiter(now func() time.Time) *fixedWindowLimiter {
	if now == nil {
		now = time.Now
	}
	return &fixedWindowLimiter{buckets: map[string]rateBucket{}, now: now}
}

func (l *fixedWindowLimiter) Allow(key string, limit int, window time.Duration) (bool, time.Duration) {
	if l == nil || limit <= 0 || window <= 0 {
		return true, 0
	}
	key = strings.TrimSpace(key)
	if key == "" {
		key = "unknown"
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	bucket := l.buckets[key]
	if bucket.reset.IsZero() || !now.Before(bucket.reset) {
		bucket = rateBucket{reset: now.Add(window)}
	}
	bucket.count++
	l.buckets[key] = bucket
	if bucket.count <= limit {
		return true, 0
	}
	retryAfter := bucket.reset.Sub(now)
	if retryAfter < 0 {
		retryAfter = 0
	}
	return false, retryAfter
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors(s.auth.CORSOrigins))

	r.Get("/healthz", s.health)
	r.Get("/readyz", s.ready)
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/wechat", s.wechatLogin)
		r.Post("/auth/wechat/mock", s.wechatMockLogin)
		r.Group(func(r chi.Router) {
			r.Use(withUser(s.store, s.auth))
			r.Post("/auth/logout", s.logout)
			r.Get("/me", s.userProfile)
			r.Put("/me", s.updateUserProfile)
			r.Delete("/me", s.deleteUserProfile)
			r.Get("/me/export", s.exportUserData)
			r.Get("/me/favorites", s.favorites)
			r.Post("/me/favorites", s.createFavorite)
			r.Delete("/me/favorites/{id}", s.deleteFavorite)
			r.Get("/billing/entitlements", s.billingEntitlements)
			r.Post("/billing/checkout", s.billingCheckout)
			r.Get("/notifications/settings", s.notificationSettings)
			r.Put("/notifications/settings", s.updateNotificationSettings)
			r.Get("/sync/status", s.syncStatus)
			r.Get("/dashboard", s.dashboard)
			r.Get("/courses", s.courses)
			r.Post("/courses", s.createCourse)
			r.Get("/courses/imports/template", s.courseImportTemplate)
			r.Post("/courses/imports", s.importCourses)
			r.Get("/courses/{id}", s.course)
			r.Put("/courses/{id}", s.updateCourse)
			r.Delete("/courses/{id}", s.deleteCourse)
			r.Get("/reminders", s.reminders)
			r.Post("/reminders", s.createReminder)
			r.Get("/reminders/{id}", s.reminder)
			r.Put("/reminders/{id}", s.updateReminder)
			r.Delete("/reminders/{id}", s.deleteReminder)
			r.Post("/reminders/{id}/complete", s.completeReminder)
			r.Post("/reminders/{id}/snooze", s.snoozeReminder)
			r.Get("/parents", s.parents)
			r.Post("/parents", s.createParent)
			r.Get("/parents/imports/template", s.parentImportTemplate)
			r.Post("/parents/imports", s.importParents)
			r.Get("/parents/{id}", s.parent)
			r.Put("/parents/{id}", s.updateParent)
			r.Delete("/parents/{id}", s.deleteParent)
			r.Get("/communication-records", s.communicationRecords)
			r.Post("/communication-records", s.createCommunicationRecord)
			r.Get("/communication-records/{id}", s.communicationRecord)
			r.Put("/communication-records/{id}", s.updateCommunicationRecord)
			r.Post("/communication-records/{id}/complete-follow-up", s.completeCommunicationFollowUp)
			r.Delete("/communication-records/{id}", s.deleteCommunicationRecord)
			r.Get("/ai/generations", s.aiGenerations)
			r.Get("/ai/generations/{id}", s.aiGeneration)
			r.Delete("/ai/generations/{id}", s.deleteAIGeneration)
			r.Post("/ai/generations/{id}/actions", s.createAIAction)
			r.Post("/ai/parent-drafts", s.parentDrafts)
			r.Post("/ai/praise", s.praise)
			r.Get("/healing/entries", s.healingEntries)
			r.Post("/healing/entries", s.createHealingEntry)
			r.Get("/healing/entries/{id}", s.healingEntry)
			r.Delete("/healing/entries/{id}", s.deleteHealingEntry)
		})
	})
	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().Format(time.RFC3339)})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dependencies := map[string]string{}
	ready := true
	for _, dependency := range s.readiness {
		if dependency.Name == "" || dependency.Check == nil {
			continue
		}
		if err := dependency.Check(ctx); err != nil {
			ready = false
			dependencies[dependency.Name] = err.Error()
			continue
		}
		dependencies[dependency.Name] = "ok"
	}
	status := http.StatusOK
	payloadStatus := "ok"
	if !ready {
		status = http.StatusServiceUnavailable
		payloadStatus = "unavailable"
	}
	writeJSON(w, status, map[string]any{"status": payloadStatus, "dependencies": dependencies, "time": time.Now().Format(time.RFC3339)})
}

func (s *Server) wechatMockLogin(w http.ResponseWriter, r *http.Request) {
	if !s.auth.AllowMockAuth {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "mock login disabled"})
		return
	}
	if !s.allowAuthRate(w, r) {
		return
	}
	var payload struct {
		Code      string `json:"code"`
		NickName  string `json:"nickName"`
		AvatarURL string `json:"avatarUrl"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateWechatLoginInput(w, payload.Code, payload.NickName, payload.AvatarURL, false) {
		return
	}
	openID := mockWechatOpenID
	if code := strings.TrimSpace(payload.Code); code != "" && code != "dev" {
		openID = "mock-openid-" + sanitizeOpenIDSeed(code)
	}
	s.writeWechatSession(w, r, openID, strings.TrimSpace(payload.NickName), strings.TrimSpace(payload.AvatarURL))
}

func (s *Server) wechatLogin(w http.ResponseWriter, r *http.Request) {
	if !s.allowAuthRate(w, r) {
		return
	}
	var payload struct {
		Code      string `json:"code"`
		NickName  string `json:"nickName"`
		AvatarURL string `json:"avatarUrl"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateWechatLoginInput(w, payload.Code, payload.NickName, payload.AvatarURL, true) {
		return
	}
	code := strings.TrimSpace(payload.Code)
	openID, err := s.exchangeWechatCode(r.Context(), code)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errWechatNotConfigured) {
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	s.writeWechatSession(w, r, openID, strings.TrimSpace(payload.NickName), strings.TrimSpace(payload.AvatarURL))
}

func (s *Server) writeWechatSession(w http.ResponseWriter, r *http.Request, openID string, nickName string, avatarURL string) {
	profile, err := s.store.FindOrCreateUserByWechatOpenID(r.Context(), openID, nickName, avatarURL)
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	sessionToken, err := signSessionToken(profile.ID, s.auth.SessionSecret, expiresAt)
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	if _, err := s.store.CreateAuthSession(r.Context(), domain.AuthSession{
		UserID:    profile.ID,
		TokenHash: sessionTokenHash(sessionToken),
		ExpiresAt: expiresAt,
	}); err != nil {
		writeResult(w, nil, err)
		return
	}
	writeJSON(w, http.StatusOK, domain.WechatSession{
		UserID:       profile.ID,
		SessionToken: sessionToken,
		OpenID:       openID,
		Profile:      profile,
		ExpiresAt:    expiresAt,
	})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	tokenHash := currentSessionTokenHash(r)
	if tokenHash == "" {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	err := s.store.RevokeAuthSession(r.Context(), tokenHash, currentUserID(r))
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

var errWechatNotConfigured = errors.New("wechat login is not configured")

func (s *Server) exchangeWechatCode(ctx context.Context, code string) (string, error) {
	appID := strings.TrimSpace(s.auth.WechatAppID)
	secret := strings.TrimSpace(s.auth.WechatSecret)
	if appID == "" || secret == "" {
		return "", errWechatNotConfigured
	}
	endpoint := "https://api.weixin.qq.com/sns/oauth2/access_token?" + url.Values{
		"appid":      []string{appID},
		"secret":     []string{secret},
		"code":       []string{code},
		"grant_type": []string{"authorization_code"},
	}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("wechat code exchange failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("wechat code exchange failed: %s", response.Status)
	}
	var result struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("wechat code exchange invalid response: %w", err)
	}
	if result.ErrCode != 0 {
		message := strings.TrimSpace(result.ErrMsg)
		if message == "" {
			message = "unknown wechat error"
		}
		return "", fmt.Errorf("wechat code exchange rejected: %s", message)
	}
	openID := strings.TrimSpace(result.OpenID)
	if openID == "" {
		return "", errors.New("wechat code exchange returned empty openid")
	}
	return openID, nil
}

func (s *Server) userProfile(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.UserProfile(r.Context(), currentUserID(r))
	writeResult(w, data, err)
}

func (s *Server) updateUserProfile(w http.ResponseWriter, r *http.Request) {
	var payload domain.UserProfile
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateUserProfile(w, payload) {
		return
	}
	data, err := s.store.UpdateUserProfile(r.Context(), currentUserID(r), payload)
	writeResult(w, data, err)
}

func (s *Server) deleteUserProfile(w http.ResponseWriter, r *http.Request) {
	err := s.store.DeleteUser(r.Context(), currentUserID(r))
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

func (s *Server) exportUserData(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.ExportUserData(r.Context(), currentUserID(r))
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="littlelight-account-export.json"`)
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) favorites(w http.ResponseWriter, r *http.Request) {
	options, queryOptions, ok := parseListOptions(w, r)
	if !ok {
		return
	}
	favoriteType, ok := optionalEnumQuery(w, r, "type", "communication_template", "ai_praise", "class_feedback")
	if !ok {
		return
	}
	data, err := s.store.Favorites(r.Context(), currentUserID(r), favoriteType, queryOptions)
	writeListResult(w, data, options, err)
}

func (s *Server) createFavorite(w http.ResponseWriter, r *http.Request) {
	var payload domain.Favorite
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateFavorite(w, payload) {
		return
	}
	data, err := s.store.CreateFavorite(r.Context(), currentUserID(r), payload)
	writeResult(w, data, err)
}

func (s *Server) deleteFavorite(w http.ResponseWriter, r *http.Request) {
	err := s.store.DeleteFavorite(r.Context(), currentUserID(r), pathID(r))
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

func (s *Server) billingEntitlements(w http.ResponseWriter, r *http.Request) {
	profile, err := s.store.UserProfile(r.Context(), currentUserID(r))
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	writeJSON(w, http.StatusOK, entitlementsForProfile(profile))
}

func (s *Server) billingCheckout(w http.ResponseWriter, r *http.Request) {
	var payload domain.BillingCheckoutRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	plan := strings.TrimSpace(payload.Plan)
	if !oneOf(plan, "monthly", "yearly") {
		validationError(w, "plan must be monthly or yearly")
		return
	}
	provider := strings.TrimSpace(payload.Provider)
	if provider == "" {
		provider = "wechat"
	}
	if provider != "wechat" {
		validationError(w, "provider must be wechat")
		return
	}
	if !payload.Mock {
		validationError(w, "mock must be true until real WeChat Pay is configured")
		return
	}
	userID := currentUserID(r)
	profile, err := s.store.UserProfile(r.Context(), userID)
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	profile.ProStatus = "pro"
	updated, err := s.store.UpdateUserProfile(r.Context(), userID, profile)
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	entitlements := entitlementsForProfile(updated)
	writeJSON(w, http.StatusOK, domain.BillingCheckoutResult{
		OrderID:      fmt.Sprintf("mock_wx_%s_%d", plan, time.Now().UnixMilli()),
		Plan:         plan,
		Provider:     provider,
		AmountCents:  billingPlanAmountCents(plan),
		Currency:     "CNY",
		Status:       "paid",
		Message:      "模拟微信支付成功，当前账号已开通微光 Pro。",
		Profile:      updated,
		Entitlements: entitlements,
	})
}

func (s *Server) notificationSettings(w http.ResponseWriter, r *http.Request) {
	profile, err := s.store.UserProfile(r.Context(), currentUserID(r))
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	writeJSON(w, http.StatusOK, notificationSettingsForProfile(profile))
}

func (s *Server) updateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	var payload domain.NotificationSettings
	if !decodeJSON(w, r, &payload) {
		return
	}
	if payload.ReminderPolicy != "" && !oneOf(payload.ReminderPolicy, "normal", "low_interrupt") {
		validationError(w, "reminderPolicy must be normal or low_interrupt")
		return
	}
	userID := currentUserID(r)
	profile, err := s.store.UserProfile(r.Context(), userID)
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	if payload.ReminderPolicy != "" {
		profile.ReminderPolicy = payload.ReminderPolicy
	}
	updated, err := s.store.UpdateUserProfile(r.Context(), userID, profile)
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	writeJSON(w, http.StatusOK, notificationSettingsForProfile(updated))
}

func (s *Server) syncStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, domain.SyncStatus{
		CloudSyncStatus:     "local_persistence",
		ObjectStorageStatus: "not_configured",
		Message:             "当前数据已按账号保存在后端数据库；对象存储和跨端云同步状态页尚未接入。",
	})
}

func entitlementsForProfile(profile domain.UserProfile) domain.Entitlements {
	status := strings.TrimSpace(profile.ProStatus)
	if status == "" {
		status = "free"
	}
	plan := "free"
	if status == "trial" || status == "pro" {
		plan = "pro"
	}
	return domain.Entitlements{
		Plan:            plan,
		Status:          status,
		Features:        []string{"AI 助手高频使用", "课表与待办同步", "家校沟通素材库", "AI 生成记录审计", "账号数据导出", "优先体验新功能"},
		CheckoutStatus:  "ready",
		CheckoutMessage: "当前为模拟微信支付流程，不会发起真实扣款；接入正式商户号后可替换为真实微信支付。",
	}
}

func billingPlanAmountCents(plan string) int {
	if plan == "yearly" {
		return 20000
	}
	return 2000
}

func notificationSettingsForProfile(profile domain.UserProfile) domain.NotificationSettings {
	policy := strings.TrimSpace(profile.ReminderPolicy)
	if policy == "" {
		policy = "low_interrupt"
	}
	return domain.NotificationSettings{
		ReminderPolicy:     policy,
		ProviderStatus:     "not_configured",
		PermissionRequired: true,
		Message:            "服务端已保存提醒偏好；系统推送 provider 与客户端通知权限尚未接入。",
	}
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	day, ok := parseDayQuery(w, r)
	if !ok {
		return
	}
	if s.dashboardCache != nil {
		if data, ok := s.dashboardCache.Get(r.Context(), userID, day); ok {
			writeJSON(w, http.StatusOK, data)
			return
		}
	}
	data, err := s.store.Dashboard(r.Context(), userID, day)
	if err == nil && s.dashboardCache != nil {
		s.dashboardCache.Set(r.Context(), userID, day, data)
	}
	writeResult(w, data, err)
}

func (s *Server) courses(w http.ResponseWriter, r *http.Request) {
	weekday, ok := parseWeekdayQuery(w, r)
	if !ok {
		return
	}
	data, err := s.store.CoursesByDay(r.Context(), currentUserID(r), weekday)
	writeResult(w, data, err)
}

func (s *Server) course(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.Course(r.Context(), currentUserID(r), pathID(r))
	writeResult(w, data, err)
}

func (s *Server) createCourse(w http.ResponseWriter, r *http.Request) {
	var payload domain.Course
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateCourse(w, payload) {
		return
	}
	userID := currentUserID(r)
	data, err := s.store.CreateCourse(r.Context(), userID, payload)
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, data, err)
}

func (s *Server) updateCourse(w http.ResponseWriter, r *http.Request) {
	var payload domain.Course
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateCourse(w, payload) {
		return
	}
	userID := currentUserID(r)
	data, err := s.store.UpdateCourse(r.Context(), userID, pathID(r), payload)
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, data, err)
}

func (s *Server) deleteCourse(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	err := s.store.DeleteCourse(r.Context(), userID, pathID(r))
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

func (s *Server) importCourses(w http.ResponseWriter, r *http.Request) {
	file, filename, ok := readUpload(w, r)
	if !ok {
		return
	}
	courses, result := service.ParseCoursesImport(filename, file)
	userID := currentUserID(r)
	courses = s.filterDuplicateImportCourses(r, userID, courses, &result)
	if isPreviewImport(r) {
		writeJSON(w, http.StatusOK, result)
		return
	}
	for _, course := range courses {
		if _, err := s.store.CreateCourse(r.Context(), userID, course); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("%s 导入失败：%v", course.Title, err))
			continue
		}
		result.Imported++
	}
	if result.Imported > 0 {
		s.invalidateDashboard(r, userID)
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) courseImportTemplate(w http.ResponseWriter, r *http.Request) {
	writeCSVDownload(w, "littlelight-course-template.csv", service.CourseImportTemplateCSV())
}

func (s *Server) reminders(w http.ResponseWriter, r *http.Request) {
	day, ok := parseDayQuery(w, r)
	if !ok {
		return
	}
	data, err := s.store.RemindersByDay(r.Context(), currentUserID(r), day)
	writeResult(w, data, err)
}

func (s *Server) reminder(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.Reminder(r.Context(), currentUserID(r), pathID(r))
	writeResult(w, data, err)
}

func (s *Server) createReminder(w http.ResponseWriter, r *http.Request) {
	var payload domain.Reminder
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateReminder(w, payload, false) {
		return
	}
	if payload.RemindAt.IsZero() {
		payload.RemindAt = time.Now().Add(30 * time.Minute)
	}
	userID := currentUserID(r)
	if !s.validateReminderReferences(w, r, userID, payload) {
		return
	}
	data, err := s.store.CreateReminder(r.Context(), userID, payload)
	if err == nil {
		s.invalidateDashboardDay(r, userID, data.RemindAt)
	}
	writeResult(w, data, err)
}

func (s *Server) updateReminder(w http.ResponseWriter, r *http.Request) {
	var payload domain.Reminder
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateReminder(w, payload, true) {
		return
	}
	if payload.RemindAt.IsZero() {
		payload.RemindAt = time.Now().Add(30 * time.Minute)
	}
	userID := currentUserID(r)
	if !s.validateReminderReferences(w, r, userID, payload) {
		return
	}
	previous, previousErr := s.store.Reminder(r.Context(), userID, pathID(r))
	data, err := s.store.UpdateReminder(r.Context(), userID, pathID(r), payload)
	if err == nil {
		if previousErr == nil {
			s.invalidateDashboardDay(r, userID, previous.RemindAt)
		}
		s.invalidateDashboardDay(r, userID, data.RemindAt)
	}
	writeResult(w, data, err)
}

func (s *Server) completeReminder(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	previous, previousErr := s.store.Reminder(r.Context(), userID, pathID(r))
	err := s.store.CompleteReminder(r.Context(), userID, pathID(r))
	if err == nil {
		if previousErr == nil {
			s.invalidateDashboardDay(r, userID, previous.RemindAt)
		} else {
			s.invalidateDashboard(r, userID)
		}
	}
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

func (s *Server) deleteReminder(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	previous, previousErr := s.store.Reminder(r.Context(), userID, pathID(r))
	err := s.store.DeleteReminder(r.Context(), userID, pathID(r))
	if err == nil {
		if previousErr == nil {
			s.invalidateDashboardDay(r, userID, previous.RemindAt)
		} else {
			s.invalidateDashboard(r, userID)
		}
	}
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

func (s *Server) snoozeReminder(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Until time.Time `json:"until"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if payload.Until.IsZero() {
		validationError(w, "until is required")
		return
	}
	if !payload.Until.After(time.Now()) {
		validationError(w, "until must be in the future")
		return
	}
	userID := currentUserID(r)
	previous, previousErr := s.store.Reminder(r.Context(), userID, pathID(r))
	data, err := s.store.SnoozeReminder(r.Context(), userID, pathID(r), payload.Until)
	if err == nil {
		if previousErr == nil {
			s.invalidateDashboardDay(r, userID, previous.RemindAt)
		}
		s.invalidateDashboardDay(r, userID, data.RemindAt)
	}
	writeResult(w, data, err)
}

func (s *Server) parents(w http.ResponseWriter, r *http.Request) {
	options, queryOptions, ok := parseListOptions(w, r)
	if !ok {
		return
	}
	data, err := s.store.Parents(r.Context(), currentUserID(r), queryOptions)
	writeListResult(w, data, options, err)
}

func (s *Server) parent(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.Parent(r.Context(), currentUserID(r), pathID(r))
	writeResult(w, data, err)
}

func (s *Server) createParent(w http.ResponseWriter, r *http.Request) {
	var payload domain.ParentProfile
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateParent(w, payload) {
		return
	}
	userID := currentUserID(r)
	data, err := s.store.CreateParent(r.Context(), userID, payload)
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, data, err)
}

func (s *Server) updateParent(w http.ResponseWriter, r *http.Request) {
	var payload domain.ParentProfile
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateParent(w, payload) {
		return
	}
	userID := currentUserID(r)
	data, err := s.store.UpdateParent(r.Context(), userID, pathID(r), payload)
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, data, err)
}

func (s *Server) deleteParent(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	err := s.store.DeleteParent(r.Context(), userID, pathID(r))
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

func (s *Server) importParents(w http.ResponseWriter, r *http.Request) {
	file, filename, ok := readUpload(w, r)
	if !ok {
		return
	}
	parents, result := service.ParseParentsImport(filename, file)
	userID := currentUserID(r)
	parents = s.filterDuplicateImportParents(r, userID, parents, &result)
	if isPreviewImport(r) {
		writeJSON(w, http.StatusOK, result)
		return
	}
	for _, parent := range parents {
		if _, err := s.store.CreateParent(r.Context(), userID, parent); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("%s 导入失败：%v", parent.StudentName, err))
			continue
		}
		result.Imported++
	}
	if result.Imported > 0 {
		s.invalidateDashboard(r, userID)
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) parentImportTemplate(w http.ResponseWriter, r *http.Request) {
	writeCSVDownload(w, "littlelight-parent-template.csv", service.ParentImportTemplateCSV())
}

func (s *Server) filterDuplicateImportCourses(r *http.Request, userID domain.ID, courses []domain.Course, result *service.ImportResult) []domain.Course {
	if result == nil || len(courses) == 0 {
		return courses
	}
	existing := map[string]bool{}
	for weekday := 0; weekday <= 6; weekday++ {
		items, err := s.store.CoursesByDay(r.Context(), userID, weekday)
		if err != nil {
			continue
		}
		for _, item := range items {
			existing[courseImportKey(item)] = true
		}
	}
	seen := map[string]bool{}
	filtered := make([]domain.Course, 0, len(courses))
	for index, course := range courses {
		row := result.ReadyRow(index)
		key := courseImportKey(course)
		switch {
		case seen[key]:
			result.MarkDuplicate(row, fmt.Sprintf("第 %d 行与本次文件中的其他课程重复，已跳过", row))
		case existing[key]:
			result.MarkDuplicate(row, fmt.Sprintf("第 %d 行课程已存在，已跳过", row))
		default:
			seen[key] = true
			filtered = append(filtered, course)
		}
	}
	return filtered
}

func (s *Server) filterDuplicateImportParents(r *http.Request, userID domain.ID, parents []domain.ParentProfile, result *service.ImportResult) []domain.ParentProfile {
	if result == nil || len(parents) == 0 {
		return parents
	}
	existing := map[string]bool{}
	for offset := 0; ; offset += 100 {
		items, err := s.store.Parents(r.Context(), userID, repository.ListOptions{Limit: 100, Offset: offset})
		if err != nil || len(items) == 0 {
			break
		}
		for _, item := range items {
			existing[parentImportKey(item)] = true
		}
		if len(items) < 100 {
			break
		}
	}
	seen := map[string]bool{}
	filtered := make([]domain.ParentProfile, 0, len(parents))
	for index, parent := range parents {
		row := result.ReadyRow(index)
		key := parentImportKey(parent)
		switch {
		case seen[key]:
			result.MarkDuplicate(row, fmt.Sprintf("第 %d 行与本次文件中的其他家长档案重复，已跳过", row))
		case existing[key]:
			result.MarkDuplicate(row, fmt.Sprintf("第 %d 行家长档案已存在，已跳过", row))
		default:
			seen[key] = true
			filtered = append(filtered, parent)
		}
	}
	return filtered
}

func (s *Server) communicationRecords(w http.ResponseWriter, r *http.Request) {
	var parentID *domain.ID
	if raw := r.URL.Query().Get("parentId"); raw != "" {
		id := domain.ID(raw)
		parentID = &id
	}
	options, queryOptions, ok := parseListOptions(w, r)
	if !ok {
		return
	}
	data, err := s.store.CommunicationRecords(r.Context(), currentUserID(r), parentID, queryOptions)
	writeListResult(w, data, options, err)
}

func (s *Server) communicationRecord(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.CommunicationRecord(r.Context(), currentUserID(r), pathID(r))
	writeResult(w, data, err)
}

func (s *Server) createCommunicationRecord(w http.ResponseWriter, r *http.Request) {
	var payload domain.CommunicationRecord
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateCommunicationRecord(w, payload) {
		return
	}
	userID := currentUserID(r)
	if !s.validateCommunicationRecordReferences(w, r, userID, payload) {
		return
	}
	data, err := s.store.CreateCommunicationRecord(r.Context(), userID, payload)
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, data, err)
}

func (s *Server) updateCommunicationRecord(w http.ResponseWriter, r *http.Request) {
	var payload domain.CommunicationRecord
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateCommunicationRecord(w, payload) {
		return
	}
	userID := currentUserID(r)
	if !s.validateCommunicationRecordReferences(w, r, userID, payload) {
		return
	}
	data, err := s.store.UpdateCommunicationRecord(r.Context(), userID, pathID(r), payload)
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, data, err)
}

func (s *Server) completeCommunicationFollowUp(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	data, err := s.store.CompleteCommunicationFollowUp(r.Context(), userID, pathID(r))
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, data, err)
}

func (s *Server) deleteCommunicationRecord(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	err := s.store.DeleteCommunicationRecord(r.Context(), userID, pathID(r))
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

func (s *Server) validateReminderReferences(w http.ResponseWriter, r *http.Request, userID domain.ID, reminder domain.Reminder) bool {
	if reminder.ParentID != nil && *reminder.ParentID != "" {
		if _, err := s.store.Parent(r.Context(), userID, *reminder.ParentID); err != nil {
			return validationError(w, "parentId is not available for current user")
		}
	}
	if reminder.CourseID != nil && *reminder.CourseID != "" {
		if _, err := s.store.Course(r.Context(), userID, *reminder.CourseID); err != nil {
			return validationError(w, "courseId is not available for current user")
		}
	}
	return true
}

func (s *Server) validateCommunicationRecordReferences(w http.ResponseWriter, r *http.Request, userID domain.ID, record domain.CommunicationRecord) bool {
	if record.ParentID == nil || *record.ParentID == "" {
		return true
	}
	if _, err := s.store.Parent(r.Context(), userID, *record.ParentID); err != nil {
		return validationError(w, "parentId is not available for current user")
	}
	return true
}

func (s *Server) parentDrafts(w http.ResponseWriter, r *http.Request) {
	if !s.allowAIRate(w, r) {
		return
	}
	var payload service.ParentDraftRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateParentDraftRequest(w, payload) {
		return
	}
	data, err := s.ai.GenerateParentDrafts(r.Context(), payload)
	if err == nil {
		generation, createErr := s.store.CreateAIGeneration(r.Context(), currentUserID(r), domain.AIGeneration{
			Scenario:    "parent_drafts",
			Input:       map[string]any{"issue": payload.Issue, "parentStyle": payload.ParentStyle, "tone": payload.Tone, "studentName": payload.StudentName},
			Output:      map[string]any{"drafts": data},
			SafetyLabel: service.SafetyLabelFromDrafts(data, "teacher_review_required"),
		})
		if createErr != nil {
			err = createErr
		} else {
			for index := range data {
				data[index].GenerationID = generation.ID
			}
			err = s.store.UpdateAIGenerationOutput(r.Context(), currentUserID(r), generation.ID, map[string]any{"drafts": data})
		}
	}
	writeResult(w, data, err)
}

func (s *Server) praise(w http.ResponseWriter, r *http.Request) {
	if !s.allowAIRate(w, r) {
		return
	}
	var payload service.PraiseRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validatePraiseRequest(w, payload) {
		return
	}
	data, err := s.ai.GeneratePraise(r.Context(), payload)
	if err == nil {
		generation, createErr := s.store.CreateAIGeneration(r.Context(), currentUserID(r), domain.AIGeneration{
			Scenario:    "praise",
			Input:       map[string]any{"persona": payload.Persona, "content": payload.Content, "mood": payload.Mood},
			Output:      map[string]any{"draft": data},
			SafetyLabel: data.Safety,
		})
		if createErr != nil {
			err = createErr
		} else {
			data.GenerationID = generation.ID
			err = s.store.UpdateAIGenerationOutput(r.Context(), currentUserID(r), generation.ID, map[string]any{"draft": data})
		}
	}
	writeResult(w, data, err)
}

func (s *Server) aiGenerations(w http.ResponseWriter, r *http.Request) {
	options, queryOptions, ok := parseListOptions(w, r)
	if !ok {
		return
	}
	scenario, ok := optionalEnumQuery(w, r, "scenario", "parent_drafts", "praise")
	if !ok {
		return
	}
	data, err := s.store.AIGenerations(r.Context(), currentUserID(r), scenario, queryOptions)
	writeListResult(w, data, options, err)
}

func (s *Server) aiGeneration(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.AIGeneration(r.Context(), currentUserID(r), pathID(r))
	writeResult(w, data, err)
}

func (s *Server) deleteAIGeneration(w http.ResponseWriter, r *http.Request) {
	err := s.store.DeleteAIGeneration(r.Context(), currentUserID(r), pathID(r))
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

func (s *Server) createAIAction(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Action   string         `json:"action"`
		DraftID  string         `json:"draftId"`
		Note     string         `json:"note"`
		Metadata map[string]any `json:"metadata"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	action := domain.AIAction{
		GenerationID: pathID(r),
		Action:       payload.Action,
		DraftID:      payload.DraftID,
		Note:         payload.Note,
		Metadata:     payload.Metadata,
	}
	if !validateAIAction(w, action) {
		return
	}
	userID := currentUserID(r)
	generation, err := s.store.AIGeneration(r.Context(), userID, action.GenerationID)
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	if !s.allowAIAction(w, generation, action) {
		return
	}
	data, err := s.store.CreateAIAction(r.Context(), userID, action)
	writeResult(w, data, err)
}

func (s *Server) allowAIAction(w http.ResponseWriter, generation domain.AIGeneration, action domain.AIAction) bool {
	if !actionDraftExists(generation, action.DraftID) {
		return validationError(w, "draftId must reference an output draft for this generation")
	}
	if !usesAIOutput(action.Action) {
		return true
	}
	if !generationRequiresReview(generation, action.DraftID) {
		return true
	}
	if hasConfirmedAIReview(generation.Actions, action.DraftID) {
		return true
	}
	return validationError(w, "review_confirmed action is required before using this AI output")
}

func (s *Server) allowAuthRate(w http.ResponseWriter, r *http.Request) bool {
	key := "auth:" + clientAddress(r)
	return s.allowRate(w, key, authLoginRateLimit, rateLimitWindow, "too many login attempts")
}

func (s *Server) allowAIRate(w http.ResponseWriter, r *http.Request) bool {
	key := "ai:" + string(currentUserID(r))
	return s.allowRate(w, key, aiGenerationRateLimit, rateLimitWindow, "too many AI generation requests")
}

func (s *Server) allowRate(w http.ResponseWriter, key string, limit int, window time.Duration, message string) bool {
	if s.rateLimiter == nil {
		return true
	}
	allowed, retryAfter := s.rateLimiter.Allow(key, limit, window)
	if allowed {
		return true
	}
	seconds := int(retryAfter.Seconds())
	if retryAfter > 0 && seconds == 0 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": "rate limited", "detail": message})
	return false
}

func clientAddress(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		if first := strings.TrimSpace(strings.Split(forwarded, ",")[0]); first != "" {
			return first
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func (s *Server) healingEntries(w http.ResponseWriter, r *http.Request) {
	options, queryOptions, ok := parseListOptions(w, r)
	if !ok {
		return
	}
	entryType, ok := optionalEnumQuery(w, r, "type", "breath", "praise", "treehole", "sound")
	if !ok {
		return
	}
	data, err := s.store.HealingEntries(r.Context(), currentUserID(r), entryType, queryOptions)
	writeListResult(w, data, options, err)
}

func (s *Server) healingEntry(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.HealingEntry(r.Context(), currentUserID(r), pathID(r))
	writeResult(w, data, err)
}

func (s *Server) createHealingEntry(w http.ResponseWriter, r *http.Request) {
	var payload domain.HealingEntry
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validateHealingEntry(w, payload) {
		return
	}
	data, err := s.store.CreateHealingEntry(r.Context(), currentUserID(r), payload)
	writeResult(w, data, err)
}

func (s *Server) deleteHealingEntry(w http.ResponseWriter, r *http.Request) {
	err := s.store.DeleteHealingEntry(r.Context(), currentUserID(r), pathID(r))
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

func (s *Server) invalidateDashboard(r *http.Request, userID domain.ID) {
	if s.dashboardCache != nil {
		s.dashboardCache.InvalidateUser(r.Context(), userID)
	}
}

func (s *Server) invalidateDashboardDay(r *http.Request, userID domain.ID, day time.Time) {
	if s.dashboardCache != nil && !day.IsZero() {
		s.dashboardCache.Invalidate(r.Context(), userID, day)
	}
}

func parseDayQuery(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	value := strings.TrimSpace(r.URL.Query().Get("day"))
	if value == "" {
		return time.Now(), true
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, validationError(w, "day must use YYYY-MM-DD format")
	}
	return parsed, true
}

func parseWeekdayQuery(w http.ResponseWriter, r *http.Request) (int, bool) {
	value := strings.TrimSpace(r.URL.Query().Get("weekday"))
	if value == "" {
		return int(time.Now().Weekday()), true
	}
	weekday, err := strconv.Atoi(value)
	if err != nil || weekday < 0 || weekday > 6 {
		return 0, validationError(w, "weekday must be an integer between 0 and 6")
	}
	return weekday, true
}

func parseListOptions(w http.ResponseWriter, r *http.Request) (repository.ListOptions, repository.ListOptions, bool) {
	query := r.URL.Query()
	limit, ok := parseOptionalIntQuery(w, query, "limit", repository.DefaultListLimit)
	if !ok {
		return repository.ListOptions{}, repository.ListOptions{}, false
	}
	offset, ok := parseOptionalIntQuery(w, query, "offset", 0)
	if !ok {
		return repository.ListOptions{}, repository.ListOptions{}, false
	}
	if limit <= 0 || limit > repository.MaxListLimit {
		return repository.ListOptions{}, repository.ListOptions{}, validationError(w, fmt.Sprintf("limit must be between 1 and %d", repository.MaxListLimit))
	}
	if offset < 0 {
		return repository.ListOptions{}, repository.ListOptions{}, validationError(w, "offset must be greater than or equal to 0")
	}
	search := strings.TrimSpace(query.Get("q"))
	if len([]rune(search)) > maxSearchQueryRunes {
		return repository.ListOptions{}, repository.ListOptions{}, validationError(w, fmt.Sprintf("q must be at most %d characters", maxSearchQueryRunes))
	}
	options := repository.ListOptions{
		Query:  search,
		Limit:  limit,
		Offset: offset,
	}
	queryOptions := options
	queryOptions.Limit = options.Limit + 1
	return options, queryOptions, true
}

func parseOptionalIntQuery(w http.ResponseWriter, query url.Values, name string, fallback int) (int, bool) {
	value := strings.TrimSpace(query.Get(name))
	if value == "" {
		return fallback, true
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, validationError(w, name+" must be an integer")
	}
	return parsed, true
}

func optionalEnumQuery(w http.ResponseWriter, r *http.Request, name string, allowed ...string) (string, bool) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return "", true
	}
	if !oneOf(value, allowed...) {
		return "", validationError(w, fmt.Sprintf("%s must be one of %s", name, strings.Join(allowed, ", ")))
	}
	return value, true
}

func isPreviewImport(r *http.Request) bool {
	value := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("preview")))
	return value == "1" || value == "true" || value == "yes"
}

func courseImportKey(course domain.Course) string {
	return strings.ToLower(strings.Join([]string{
		strings.TrimSpace(course.Title),
		strings.TrimSpace(course.ClassName),
		strings.TrimSpace(course.StartTime),
		strings.TrimSpace(course.EndTime),
		strconv.Itoa(course.Weekday),
	}, "|"))
}

func parentImportKey(parent domain.ParentProfile) string {
	contact := strings.TrimSpace(parent.Contact)
	if contact != "" {
		return strings.ToLower(strings.Join([]string{strings.TrimSpace(parent.StudentName), strings.TrimSpace(parent.ClassName), contact}, "|"))
	}
	return strings.ToLower(strings.Join([]string{strings.TrimSpace(parent.StudentName), strings.TrimSpace(parent.ClassName), strings.TrimSpace(parent.ParentName)}, "|"))
}

func pathID(r *http.Request) domain.ID {
	return domain.ID(chi.URLParam(r, "id"))
}

func writeResult(w http.ResponseWriter, data any, err error) {
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repository.ErrNotFound) || strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func writeListResult[T any](w http.ResponseWriter, items []T, options repository.ListOptions, err error) {
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	count := len(items)
	hasMore := count > options.Limit
	if hasMore {
		items = items[:options.Limit]
		count = len(items)
	}
	nextOffset := 0
	if hasMore {
		nextOffset = options.Offset + count
	}
	writeJSON(w, http.StatusOK, ListResponse[T]{
		Items: items,
		PageInfo: PageInfo{
			Limit:      options.Limit,
			Offset:     options.Offset,
			Count:      count,
			NextOffset: nextOffset,
			HasMore:    hasMore,
		},
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json", "detail": err.Error()})
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json", "detail": "request body must contain a single JSON object"})
		return false
	}
	return true
}

func validateWechatLoginInput(w http.ResponseWriter, code string, nickName string, avatarURL string, requireCode bool) bool {
	if requireCode && !requireText(w, map[string]string{"code": code}) {
		return false
	}
	return requireMaxFields(w, map[string]fieldLimit{
		"code":      {Value: code, Max: 512},
		"nickName":  {Value: nickName, Max: 80},
		"avatarUrl": {Value: avatarURL, Max: 500},
	})
}

func validateUserProfile(w http.ResponseWriter, value domain.UserProfile) bool {
	fields := map[string]string{"name": value.Name}
	if !requireText(w, fields) {
		return false
	}
	if !requireMaxFields(w, map[string]fieldLimit{
		"name":    {Value: value.Name, Max: 40},
		"school":  {Value: value.School, Max: 80},
		"stage":   {Value: value.Stage, Max: 40},
		"subject": {Value: value.Subject, Max: 40},
	}) {
		return false
	}
	if value.ProStatus != "" && !oneOf(value.ProStatus, "free", "trial", "pro", "expired") {
		return validationError(w, "proStatus must be one of free, trial, pro, expired")
	}
	if value.ReminderPolicy != "" && !oneOf(value.ReminderPolicy, "normal", "low_interrupt") {
		return validationError(w, "reminderPolicy must be normal or low_interrupt")
	}
	return true
}

func validateFavorite(w http.ResponseWriter, value domain.Favorite) bool {
	if !requireText(w, map[string]string{"type": value.Type, "title": value.Title, "content": value.Content}) {
		return false
	}
	if !oneOf(value.Type, "communication_template", "ai_praise", "class_feedback") {
		return validationError(w, "type must be one of communication_template, ai_praise, class_feedback")
	}
	return requireMaxFields(w, map[string]fieldLimit{
		"title":   {Value: value.Title, Max: 80},
		"content": {Value: value.Content, Max: 2000},
	})
}

func validateCourse(w http.ResponseWriter, value domain.Course) bool {
	if !requireText(w, map[string]string{"title": value.Title, "className": value.ClassName, "startTime": value.StartTime, "endTime": value.EndTime}) {
		return false
	}
	if value.Weekday < 0 || value.Weekday > 6 {
		return validationError(w, "weekday must be between 0 and 6")
	}
	if !validClock(value.StartTime) || !validClock(value.EndTime) {
		return validationError(w, "startTime and endTime must use HH:MM format")
	}
	if !clockBefore(value.StartTime, value.EndTime) {
		return validationError(w, "endTime must be after startTime")
	}
	return requireMaxFields(w, map[string]fieldLimit{
		"title":     {Value: value.Title, Max: 80},
		"className": {Value: value.ClassName, Max: 80},
		"location":  {Value: value.Location, Max: 120},
		"note":      {Value: value.Note, Max: 1000},
	})
}

func validateReminder(w http.ResponseWriter, value domain.Reminder, allowStatus bool) bool {
	if !requireText(w, map[string]string{"title": value.Title}) {
		return false
	}
	if value.RemindAt.IsZero() {
		return validationError(w, "remindAt is required")
	}
	if value.Status != "" && (!allowStatus || !oneOf(value.Status, "pending", "done", "snoozed", "deleted")) {
		return validationError(w, "status must be one of pending, done, snoozed, deleted")
	}
	return requireMaxFields(w, map[string]fieldLimit{
		"title":    {Value: value.Title, Max: 120},
		"category": {Value: value.Category, Max: 40},
		"note":     {Value: value.Note, Max: 1000},
	})
}

func validateParent(w http.ResponseWriter, value domain.ParentProfile) bool {
	if !requireText(w, map[string]string{
		"studentName":  value.StudentName,
		"className":    value.ClassName,
		"parentName":   value.ParentName,
		"relationship": value.Relationship,
	}) {
		return false
	}
	if value.RiskLevel != "" && !oneOf(value.RiskLevel, "low", "medium", "high") {
		return validationError(w, "riskLevel must be low, medium, or high")
	}
	return requireMaxFields(w, map[string]fieldLimit{
		"studentName":        {Value: value.StudentName, Max: 60},
		"className":          {Value: value.ClassName, Max: 80},
		"parentName":         {Value: value.ParentName, Max: 80},
		"relationship":       {Value: value.Relationship, Max: 40},
		"contact":            {Value: value.Contact, Max: 80},
		"communicationStyle": {Value: value.CommunicationStyle, Max: 80},
		"importantNotes":     {Value: value.ImportantNotes, Max: 2000},
		"nextAction":         {Value: value.NextAction, Max: 1000},
	})
}

func validateCommunicationRecord(w http.ResponseWriter, value domain.CommunicationRecord) bool {
	if !requireText(w, map[string]string{"student": value.Student, "channel": value.Channel, "reason": value.Reason, "summary": value.Summary}) {
		return false
	}
	if value.RiskLevel != "" && !oneOf(value.RiskLevel, "low", "medium", "high") {
		return validationError(w, "riskLevel must be low, medium, or high")
	}
	if value.FollowUpStatus != "" && !oneOf(value.FollowUpStatus, "pending", "done") {
		return validationError(w, "followUpStatus must be pending or done")
	}
	return requireMaxFields(w, map[string]fieldLimit{
		"student": {Value: value.Student, Max: 60},
		"channel": {Value: value.Channel, Max: 40},
		"reason":  {Value: value.Reason, Max: 200},
		"summary": {Value: value.Summary, Max: 2000},
		"result":  {Value: value.Result, Max: 2000},
	})
}

func validateParentDraftRequest(w http.ResponseWriter, value service.ParentDraftRequest) bool {
	if !requireText(w, map[string]string{"issue": value.Issue}) {
		return false
	}
	return requireMaxFields(w, map[string]fieldLimit{
		"issue":       {Value: value.Issue, Max: 1200},
		"parentStyle": {Value: value.ParentStyle, Max: 80},
		"tone":        {Value: value.Tone, Max: 40},
		"studentName": {Value: value.StudentName, Max: 60},
	})
}

func validatePraiseRequest(w http.ResponseWriter, value service.PraiseRequest) bool {
	if !requireText(w, map[string]string{"content": value.Content}) {
		return false
	}
	return requireMaxFields(w, map[string]fieldLimit{
		"persona": {Value: value.Persona, Max: 80},
		"content": {Value: value.Content, Max: 1000},
		"mood":    {Value: value.Mood, Max: 40},
	})
}

func validateAIAction(w http.ResponseWriter, value domain.AIAction) bool {
	if !requireText(w, map[string]string{"action": value.Action}) {
		return false
	}
	if value.GenerationID == "" {
		return validationError(w, "generationId is required")
	}
	if !oneOf(value.Action, "copy", "save_record", "save_healing", "review_confirmed", "review_skipped") {
		return validationError(w, "action must be one of copy, save_record, save_healing, review_confirmed, review_skipped")
	}
	if !requireMaxFields(w, map[string]fieldLimit{
		"draftId": {Value: value.DraftID, Max: 120},
		"note":    {Value: value.Note, Max: 500},
	}) {
		return false
	}
	if len(value.Metadata) > 20 {
		return validationError(w, "metadata must contain at most 20 keys")
	}
	if len(value.Metadata) > 0 {
		data, err := json.Marshal(value.Metadata)
		if err != nil {
			return validationError(w, "metadata must be valid json")
		}
		if len(data) > maxAIActionMetadataBytes {
			return validationError(w, "metadata must be at most 4096 bytes")
		}
		if !validateAIActionMetadataValue(w, value.Metadata, 0) {
			return false
		}
	}
	return true
}

func validateAIActionMetadataValue(w http.ResponseWriter, value any, depth int) bool {
	if depth > maxAIActionMetadataDepth {
		return validationError(w, "metadata nesting is too deep")
	}
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) > 20 {
			return validationError(w, "metadata object must contain at most 20 keys")
		}
		for key, item := range typed {
			if len([]rune(key)) > 80 {
				return validationError(w, "metadata keys must be at most 80 characters")
			}
			if !validateAIActionMetadataValue(w, item, depth+1) {
				return false
			}
		}
	case []any:
		if len(typed) > maxAIActionMetadataArrayItems {
			return validationError(w, "metadata arrays must contain at most 20 items")
		}
		for _, item := range typed {
			if !validateAIActionMetadataValue(w, item, depth+1) {
				return false
			}
		}
	case string:
		if len([]rune(typed)) > maxAIActionMetadataStringRunes {
			return validationError(w, "metadata strings must be at most 500 characters")
		}
	}
	return true
}

func usesAIOutput(action string) bool {
	return action == "copy" || action == "save_record" || action == "save_healing"
}

func generationRequiresReview(generation domain.AIGeneration, draftID string) bool {
	if isReviewSafety(generation.SafetyLabel) {
		return true
	}
	for _, draft := range generationOutputDrafts(generation.Output) {
		if draftID != "" && draft.ID != "" && string(draft.ID) != draftID {
			continue
		}
		if draft.ReviewRequired || draft.Fallback || isReviewSafety(draft.Safety) || isReviewSafety(draft.SafetyLevel) {
			return true
		}
	}
	return false
}

func hasConfirmedAIReview(actions []domain.AIAction, draftID string) bool {
	if strings.TrimSpace(draftID) == "" {
		return false
	}
	for _, action := range actions {
		if action.Action != "review_confirmed" {
			continue
		}
		if action.DraftID == draftID {
			return true
		}
	}
	return false
}

func actionDraftExists(generation domain.AIGeneration, draftID string) bool {
	draftID = strings.TrimSpace(draftID)
	drafts := generationOutputDrafts(generation.Output)
	if len(drafts) == 0 {
		return false
	}
	if draftID == "" {
		return false
	}
	hasIdentifiedDraft := false
	for _, draft := range drafts {
		id := strings.TrimSpace(string(draft.ID))
		if id == "" {
			continue
		}
		hasIdentifiedDraft = true
		if id == draftID {
			return true
		}
	}
	if !hasIdentifiedDraft && len(drafts) == 1 {
		return true
	}
	return false
}

func isReviewSafety(value string) bool {
	return oneOf(value, "teacher_review_required", "crisis_support_required", "student_safety_review_required", "medical_review_required", "review", "high")
}

func generationOutputDrafts(output map[string]any) []domain.AIDraft {
	if len(output) == 0 {
		return nil
	}
	drafts := make([]domain.AIDraft, 0)
	switch values := output["drafts"].(type) {
	case []domain.AIDraft:
		drafts = append(drafts, values...)
	case []any:
		for _, value := range values {
			if draft, ok := decodeAIDraft(value); ok {
				drafts = append(drafts, draft)
			}
		}
	}
	if value, ok := output["draft"]; ok {
		if draft, ok := decodeAIDraft(value); ok {
			drafts = append(drafts, draft)
		}
	}
	if _, ok := output["content"]; ok {
		if draft, ok := decodeAIDraft(output); ok {
			drafts = append(drafts, draft)
		}
	}
	return drafts
}

func decodeAIDraft(value any) (domain.AIDraft, bool) {
	data, err := json.Marshal(value)
	if err != nil {
		return domain.AIDraft{}, false
	}
	var draft domain.AIDraft
	if err := json.Unmarshal(data, &draft); err != nil {
		return domain.AIDraft{}, false
	}
	return draft, draft.ID != "" || draft.Content != "" || draft.Safety != "" || draft.SafetyLevel != ""
}

func validateHealingEntry(w http.ResponseWriter, value domain.HealingEntry) bool {
	if !requireText(w, map[string]string{"type": value.Type}) {
		return false
	}
	if !oneOf(value.Type, "breath", "praise", "treehole", "sound") {
		return validationError(w, "type must be one of breath, praise, treehole, sound")
	}
	return requireMaxFields(w, map[string]fieldLimit{
		"mood":    {Value: value.Mood, Max: 40},
		"content": {Value: value.Content, Max: 2000},
		"aiReply": {Value: value.AIReply, Max: 2000},
	})
}

type fieldLimit struct {
	Value string
	Max   int
}

func requireText(w http.ResponseWriter, fields map[string]string) bool {
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return validationError(w, field+" is required")
		}
	}
	return true
}

func requireMax(w http.ResponseWriter, field string, value string, max int) bool {
	if len([]rune(value)) > max {
		return validationError(w, fmt.Sprintf("%s must be at most %d characters", field, max))
	}
	return true
}

func requireMaxFields(w http.ResponseWriter, fields map[string]fieldLimit) bool {
	for field, limit := range fields {
		if !requireMax(w, field, limit.Value, limit.Max) {
			return false
		}
	}
	return true
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func validClock(value string) bool {
	_, err := time.Parse("15:04", value)
	return err == nil
}

func clockBefore(start string, end string) bool {
	startTime, err := time.Parse("15:04", start)
	if err != nil {
		return false
	}
	endTime, err := time.Parse("15:04", end)
	if err != nil {
		return false
	}
	return startTime.Before(endTime)
}

func sanitizeOpenIDSeed(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			builder.WriteRune(char)
		}
	}
	if builder.Len() == 0 {
		return "dev"
	}
	if builder.Len() > 80 {
		return builder.String()[:80]
	}
	return builder.String()
}

func validationError(w http.ResponseWriter, message string) bool {
	writeJSON(w, http.StatusBadRequest, map[string]any{"error": "validation failed", "detail": message})
	return false
}

func readUpload(w http.ResponseWriter, r *http.Request) ([]byte, string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid upload", "detail": err.Error()})
		return nil, "", false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing file field"})
		return nil, "", false
	}
	defer file.Close()
	filename := strings.TrimSpace(header.Filename)
	if filename == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing filename"})
		return nil, "", false
	}
	if !oneOf(strings.ToLower(filepath.Ext(filename)), ".csv", ".xlsx") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unsupported file type", "detail": "only .csv or .xlsx files are supported"})
		return nil, "", false
	}
	if header.Size == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "empty upload"})
		return nil, "", false
	}
	data, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "read upload failed", "detail": err.Error()})
		return nil, "", false
	}
	if len(data) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "empty upload"})
		return nil, "", false
	}
	return data, filename, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeCSVDownload(w http.ResponseWriter, filename string, data []byte) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func cors(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedOrigins))
	allowAnyOrigin := len(allowedOrigins) == 0
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAnyOrigin = true
			continue
		}
		allowed[origin] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			switch {
			case allowAnyOrigin:
				w.Header().Set("Access-Control-Allow-Origin", "*")
			case origin != "" && allowed[origin]:
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
