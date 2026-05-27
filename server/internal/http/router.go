package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/web3gaoyutang/littlelight/server/internal/domain"
	"github.com/web3gaoyutang/littlelight/server/internal/platform/cache"
	"github.com/web3gaoyutang/littlelight/server/internal/repository"
	"github.com/web3gaoyutang/littlelight/server/internal/service"
)

type Server struct {
	store          repository.Store
	ai             *service.AIService
	dashboardCache *cache.DashboardCache
}

func NewServer(store repository.Store, ai *service.AIService, dashboardCache *cache.DashboardCache) *Server {
	return &Server{store: store, ai: ai, dashboardCache: dashboardCache}
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors)
	r.Use(withUser)

	r.Get("/healthz", s.health)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/me", s.userProfile)
		r.Put("/me", s.updateUserProfile)
		r.Get("/me/favorites", s.favorites)
		r.Post("/me/favorites", s.createFavorite)
		r.Delete("/me/favorites/{id}", s.deleteFavorite)
		r.Get("/dashboard", s.dashboard)
		r.Get("/courses", s.courses)
		r.Post("/courses", s.createCourse)
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
		r.Get("/parents/{id}", s.parent)
		r.Put("/parents/{id}", s.updateParent)
		r.Delete("/parents/{id}", s.deleteParent)
		r.Get("/communication-records", s.communicationRecords)
		r.Post("/communication-records", s.createCommunicationRecord)
		r.Get("/communication-records/{id}", s.communicationRecord)
		r.Put("/communication-records/{id}", s.updateCommunicationRecord)
		r.Delete("/communication-records/{id}", s.deleteCommunicationRecord)
		r.Post("/ai/parent-drafts", s.parentDrafts)
		r.Post("/ai/praise", s.praise)
		r.Get("/healing/entries", s.healingEntries)
		r.Post("/healing/entries", s.createHealingEntry)
		r.Get("/healing/entries/{id}", s.healingEntry)
		r.Delete("/healing/entries/{id}", s.deleteHealingEntry)
	})
	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().Format(time.RFC3339)})
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
	data, err := s.store.UpdateUserProfile(r.Context(), currentUserID(r), payload)
	writeResult(w, data, err)
}

func (s *Server) favorites(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.Favorites(r.Context(), currentUserID(r), r.URL.Query().Get("type"))
	writeResult(w, data, err)
}

func (s *Server) createFavorite(w http.ResponseWriter, r *http.Request) {
	var payload domain.Favorite
	if !decodeJSON(w, r, &payload) {
		return
	}
	data, err := s.store.CreateFavorite(r.Context(), currentUserID(r), payload)
	writeResult(w, data, err)
}

func (s *Server) deleteFavorite(w http.ResponseWriter, r *http.Request) {
	err := s.store.DeleteFavorite(r.Context(), currentUserID(r), pathID(r))
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	day := parseDay(r)
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
	weekday, _ := strconv.Atoi(r.URL.Query().Get("weekday"))
	if weekday < 0 || weekday > 6 {
		weekday = int(time.Now().Weekday())
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

func (s *Server) reminders(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.RemindersByDay(r.Context(), currentUserID(r), parseDay(r))
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
	if payload.RemindAt.IsZero() {
		payload.RemindAt = time.Now().Add(30 * time.Minute)
	}
	userID := currentUserID(r)
	data, err := s.store.CreateReminder(r.Context(), userID, payload)
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, data, err)
}

func (s *Server) updateReminder(w http.ResponseWriter, r *http.Request) {
	var payload domain.Reminder
	if !decodeJSON(w, r, &payload) {
		return
	}
	if payload.RemindAt.IsZero() {
		payload.RemindAt = time.Now().Add(30 * time.Minute)
	}
	userID := currentUserID(r)
	data, err := s.store.UpdateReminder(r.Context(), userID, pathID(r), payload)
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, data, err)
}

func (s *Server) completeReminder(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	err := s.store.CompleteReminder(r.Context(), userID, pathID(r))
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

func (s *Server) deleteReminder(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	err := s.store.DeleteReminder(r.Context(), userID, pathID(r))
	if err == nil {
		s.invalidateDashboard(r, userID)
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
		payload.Until = time.Now().Add(30 * time.Minute)
	}
	userID := currentUserID(r)
	data, err := s.store.SnoozeReminder(r.Context(), userID, pathID(r), payload.Until)
	if err == nil {
		s.invalidateDashboard(r, userID)
	}
	writeResult(w, data, err)
}

func (s *Server) parents(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.Parents(r.Context(), currentUserID(r))
	writeResult(w, data, err)
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

func (s *Server) communicationRecords(w http.ResponseWriter, r *http.Request) {
	var parentID *domain.ID
	if raw := r.URL.Query().Get("parentId"); raw != "" {
		id := domain.ID(raw)
		parentID = &id
	}
	data, err := s.store.CommunicationRecords(r.Context(), currentUserID(r), parentID)
	writeResult(w, data, err)
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
	data, err := s.store.CreateCommunicationRecord(r.Context(), currentUserID(r), payload)
	writeResult(w, data, err)
}

func (s *Server) updateCommunicationRecord(w http.ResponseWriter, r *http.Request) {
	var payload domain.CommunicationRecord
	if !decodeJSON(w, r, &payload) {
		return
	}
	data, err := s.store.UpdateCommunicationRecord(r.Context(), currentUserID(r), pathID(r), payload)
	writeResult(w, data, err)
}

func (s *Server) deleteCommunicationRecord(w http.ResponseWriter, r *http.Request) {
	err := s.store.DeleteCommunicationRecord(r.Context(), currentUserID(r), pathID(r))
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

func (s *Server) parentDrafts(w http.ResponseWriter, r *http.Request) {
	var payload service.ParentDraftRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	data, err := s.ai.GenerateParentDrafts(r.Context(), payload)
	writeResult(w, data, err)
}

func (s *Server) praise(w http.ResponseWriter, r *http.Request) {
	var payload service.PraiseRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	data, err := s.ai.GeneratePraise(r.Context(), payload)
	writeResult(w, data, err)
}

func (s *Server) healingEntries(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.HealingEntries(r.Context(), currentUserID(r), r.URL.Query().Get("type"))
	writeResult(w, data, err)
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
	data, err := s.store.CreateHealingEntry(r.Context(), currentUserID(r), payload)
	writeResult(w, data, err)
}

func (s *Server) deleteHealingEntry(w http.ResponseWriter, r *http.Request) {
	err := s.store.DeleteHealingEntry(r.Context(), currentUserID(r), pathID(r))
	writeResult(w, map[string]bool{"ok": err == nil}, err)
}

func (s *Server) invalidateDashboard(r *http.Request, userID domain.ID) {
	if s.dashboardCache != nil {
		s.dashboardCache.Invalidate(r.Context(), userID, parseDay(r))
	}
}

func parseDay(r *http.Request) time.Time {
	if value := r.URL.Query().Get("day"); value != "" {
		if parsed, err := time.Parse("2006-01-02", value); err == nil {
			return parsed
		}
	}
	return time.Now()
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

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json", "detail": err.Error()})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
