package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrMissingUserID = errors.New("user id is required")
)

const (
	defaultMemoryUserID = domain.ID("00000000-0000-0000-0000-000000000001")
	defaultMockOpenID   = "mock-openid-littlelight-teacher"
)

type Store interface {
	FindOrCreateUserByWechatOpenID(ctx context.Context, openID string, nickName string, avatarURL string) (domain.UserProfile, error)
	CreateAuthSession(ctx context.Context, session domain.AuthSession) (domain.AuthSession, error)
	AuthSessionByTokenHash(ctx context.Context, tokenHash string) (domain.AuthSession, error)
	RevokeAuthSession(ctx context.Context, tokenHash string, userID domain.ID) error
	UserProfile(ctx context.Context, userID domain.ID) (domain.UserProfile, error)
	UpdateUserProfile(ctx context.Context, userID domain.ID, profile domain.UserProfile) (domain.UserProfile, error)
	DeleteUser(ctx context.Context, userID domain.ID) error
	ExportUserData(ctx context.Context, userID domain.ID) (domain.AccountExport, error)
	Dashboard(ctx context.Context, userID domain.ID, day time.Time) (domain.DashboardSummary, error)
	CoursesByDay(ctx context.Context, userID domain.ID, weekday int) ([]domain.Course, error)
	Course(ctx context.Context, userID domain.ID, id domain.ID) (domain.Course, error)
	CreateCourse(ctx context.Context, userID domain.ID, course domain.Course) (domain.Course, error)
	UpdateCourse(ctx context.Context, userID domain.ID, id domain.ID, course domain.Course) (domain.Course, error)
	DeleteCourse(ctx context.Context, userID domain.ID, id domain.ID) error
	RemindersByDay(ctx context.Context, userID domain.ID, day time.Time) ([]domain.Reminder, error)
	Reminder(ctx context.Context, userID domain.ID, id domain.ID) (domain.Reminder, error)
	CreateReminder(ctx context.Context, userID domain.ID, reminder domain.Reminder) (domain.Reminder, error)
	UpdateReminder(ctx context.Context, userID domain.ID, id domain.ID, reminder domain.Reminder) (domain.Reminder, error)
	CompleteReminder(ctx context.Context, userID domain.ID, id domain.ID) error
	DeleteReminder(ctx context.Context, userID domain.ID, id domain.ID) error
	SnoozeReminder(ctx context.Context, userID domain.ID, id domain.ID, until time.Time) (domain.Reminder, error)
	Parents(ctx context.Context, userID domain.ID, options ListOptions) ([]domain.ParentProfile, error)
	Parent(ctx context.Context, userID domain.ID, id domain.ID) (domain.ParentProfile, error)
	CreateParent(ctx context.Context, userID domain.ID, parent domain.ParentProfile) (domain.ParentProfile, error)
	UpdateParent(ctx context.Context, userID domain.ID, id domain.ID, parent domain.ParentProfile) (domain.ParentProfile, error)
	DeleteParent(ctx context.Context, userID domain.ID, id domain.ID) error
	CommunicationRecords(ctx context.Context, userID domain.ID, parentID *domain.ID, options ListOptions) ([]domain.CommunicationRecord, error)
	CommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID) (domain.CommunicationRecord, error)
	CreateCommunicationRecord(ctx context.Context, userID domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error)
	UpdateCommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error)
	CompleteCommunicationFollowUp(ctx context.Context, userID domain.ID, id domain.ID) (domain.CommunicationRecord, error)
	DeleteCommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID) error
	HealingEntries(ctx context.Context, userID domain.ID, entryType string, options ListOptions) ([]domain.HealingEntry, error)
	HealingEntry(ctx context.Context, userID domain.ID, id domain.ID) (domain.HealingEntry, error)
	CreateHealingEntry(ctx context.Context, userID domain.ID, entry domain.HealingEntry) (domain.HealingEntry, error)
	DeleteHealingEntry(ctx context.Context, userID domain.ID, id domain.ID) error
	AIGenerations(ctx context.Context, userID domain.ID, scenario string, options ListOptions) ([]domain.AIGeneration, error)
	AIGeneration(ctx context.Context, userID domain.ID, id domain.ID) (domain.AIGeneration, error)
	CreateAIGeneration(ctx context.Context, userID domain.ID, generation domain.AIGeneration) (domain.AIGeneration, error)
	UpdateAIGenerationOutput(ctx context.Context, userID domain.ID, id domain.ID, output map[string]any) error
	DeleteAIGeneration(ctx context.Context, userID domain.ID, id domain.ID) error
	CreateAIAction(ctx context.Context, userID domain.ID, action domain.AIAction) (domain.AIAction, error)
	AIActions(ctx context.Context, userID domain.ID, generationID domain.ID) ([]domain.AIAction, error)
	Favorites(ctx context.Context, userID domain.ID, favoriteType string, options ListOptions) ([]domain.Favorite, error)
	CreateFavorite(ctx context.Context, userID domain.ID, favorite domain.Favorite) (domain.Favorite, error)
	DeleteFavorite(ctx context.Context, userID domain.ID, id domain.ID) error
}

type ListOptions struct {
	Query  string
	Limit  int
	Offset int
}

const (
	DefaultListLimit  = 50
	MaxListLimit      = 100
	MaxListProbeLimit = MaxListLimit + 1
)

type MemoryStore struct {
	mu          sync.RWMutex
	users       map[domain.ID]*memoryUserData
	wechatUsers map[string]domain.ID
	seq         int64
}

type memoryUserData struct {
	profile   domain.UserProfile
	courses   []domain.Course
	reminders []domain.Reminder
	parents   []domain.ParentProfile
	records   []domain.CommunicationRecord
	healing   []domain.HealingEntry
	aiLogs    []domain.AIGeneration
	aiActions []domain.AIAction
	favorites []domain.Favorite
	sessions  []domain.AuthSession
}

func NewMemoryStore() *MemoryStore {
	now := time.Now()
	parent1 := domain.ParentProfile{ID: "parent_lin", StudentName: "林晓晓", ClassName: "高二(5)班", ParentName: "林晓晓妈妈", Relationship: "母亲", CommunicationStyle: "比较敏感", RiskLevel: "medium", ImportantNotes: "睡眠与到校状态需要持续观察。", NextAction: "先确认睡眠，再同步课堂参与中的积极信号。", CreatedAt: now}
	parent2 := domain.ParentProfile{ID: "parent_chen", StudentName: "陈子默", ClassName: "高二(3)班", ParentName: "陈子默爸爸", Relationship: "父亲", CommunicationStyle: "关注成绩", RiskLevel: "low", ImportantNotes: "关注测试反馈和订正节奏。", NextAction: "周五前同步订正计划。", CreatedAt: now}
	demo := &memoryUserData{
		profile: domain.UserProfile{
			ID:             defaultMemoryUserID,
			Name:           "林小微",
			School:         "微光实验小学",
			Stage:          "小学",
			Subject:        "语文",
			IsHeadTeacher:  true,
			ProStatus:      "trial",
			ReminderPolicy: "low_interrupt",
			CreatedAt:      now,
		},
		courses: []domain.Course{
			{ID: "course_psychology", Title: "心理健康", ClassName: "高二(3)班", Location: "教学楼 B 座 402 室", Weekday: int(now.Weekday()), StartTime: "09:30", EndTime: "10:15", Note: "情绪识别与压力调节", CreatedAt: now},
			{ID: "course_talk", Title: "个别谈话", ClassName: "林晓晓", Location: "咨询室", Weekday: int(now.Weekday()), StartTime: "13:40", EndTime: "14:00", Note: "睡眠与到校状态", CreatedAt: now},
		},
		reminders: []domain.Reminder{
			{ID: "reminder_lin", Title: "给林晓晓妈妈发睡眠观察", Category: "回访", RemindAt: atToday("11:45"), Status: "pending", Note: "语气温和，先肯定再建议", ParentID: &parent1.ID, CreatedAt: now},
			{ID: "reminder_chen", Title: "回访陈子默爸爸", Category: "回访", RemindAt: atToday("17:20"), Status: "pending", Note: "同步测试反馈和订正计划", ParentID: &parent2.ID, CreatedAt: now},
		},
		parents: []domain.ParentProfile{parent1, parent2},
		records: []domain.CommunicationRecord{
			{ID: "record_chen", ParentID: &parent2.ID, Student: "陈子默", Channel: "微信", Reason: "测试反馈", Summary: "已说明测试问题与订正方向。", Result: "家长认可 3 天订正计划。", RiskLevel: "low", FollowUpAt: now.Add(72 * time.Hour), FollowUpStatus: "pending", CreatedAt: now.Add(-24 * time.Hour)},
		},
		healing: []domain.HealingEntry{
			{ID: "healing_seed_1", Type: "praise", Mood: "warm", Content: "今天处理了课程和家长反馈。", AIReply: "你已经稳稳接住了很多复杂信息，先给自己一点恢复空间。", CreatedAt: now.Add(-2 * time.Hour)},
			{ID: "healing_seed_2", Type: "breath", Mood: "calm", Content: "完成 1 分钟呼吸练习", AIReply: "已完成一次短恢复。", CreatedAt: now.Add(-4 * time.Hour)},
		},
		aiLogs: []domain.AIGeneration{
			{ID: "ai_generation_seed_1", Scenario: "praise", Input: map[string]any{"persona": "温柔前辈", "content": "今天处理了课程和家长反馈。"}, Output: map[string]any{"content": "你已经稳稳接住了很多复杂信息，先给自己一点恢复空间。"}, SafetyLabel: "self_care", TokenUsage: 0, CreatedAt: now.Add(-2 * time.Hour)},
		},
		favorites: []domain.Favorite{
			{ID: "favorite_reply_1", Type: "communication_template", Title: "先肯定再建议", Content: "先同步孩子已经做到的部分，再给出一个可执行的小建议。", CreatedAt: now},
			{ID: "favorite_praise_1", Type: "ai_praise", Title: "忙碌后恢复", Content: "你今天处理了很多细碎但重要的事，先允许自己慢下来。", CreatedAt: now},
		},
	}
	return &MemoryStore{
		seq:         100,
		users:       map[domain.ID]*memoryUserData{defaultMemoryUserID: demo},
		wechatUsers: map[string]domain.ID{defaultMockOpenID: defaultMemoryUserID},
	}
}

func (s *MemoryStore) FindOrCreateUserByWechatOpenID(ctx context.Context, openID string, nickName string, avatarURL string) (domain.UserProfile, error) {
	openID = strings.TrimSpace(openID)
	if openID == "" {
		return domain.UserProfile{}, errors.New("wechat openid is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	userID, ok := s.wechatUsers[openID]
	if !ok {
		s.seq++
		userID = domain.ID(fmt.Sprintf("memory_user_%d", s.seq))
		s.wechatUsers[openID] = userID
		s.users[userID] = &memoryUserData{profile: defaultUserProfile(userID, nickName, time.Now())}
	}
	data, err := s.existingUserDataLocked(userID)
	if err != nil {
		return domain.UserProfile{}, err
	}
	if name := strings.TrimSpace(nickName); name != "" && name != data.profile.Name {
		data.profile.Name = name
	}
	return data.profile, nil
}

func (s *MemoryStore) CreateAuthSession(ctx context.Context, session domain.AuthSession) (domain.AuthSession, error) {
	session.TokenHash = strings.TrimSpace(session.TokenHash)
	if session.UserID == "" {
		return domain.AuthSession{}, ErrMissingUserID
	}
	if session.TokenHash == "" {
		return domain.AuthSession{}, errors.New("session token hash is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(session.UserID)
	s.seq++
	session.ID = domain.ID(fmt.Sprintf("auth_session_%d", s.seq))
	session.CreatedAt = time.Now()
	data.sessions = append([]domain.AuthSession{session}, data.sessions...)
	return session, nil
}

func (s *MemoryStore) AuthSessionByTokenHash(ctx context.Context, tokenHash string) (domain.AuthSession, error) {
	tokenHash = strings.TrimSpace(tokenHash)
	if tokenHash == "" {
		return domain.AuthSession{}, notFound("auth session", "")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, data := range s.users {
		for _, session := range data.sessions {
			if session.TokenHash == tokenHash {
				if session.RevokedAt != nil || !session.ExpiresAt.After(time.Now()) {
					return domain.AuthSession{}, notFound("auth session", domain.ID(session.ID))
				}
				if _, ok := s.users[session.UserID]; !ok {
					return domain.AuthSession{}, notFound("auth session", domain.ID(session.ID))
				}
				return session, nil
			}
		}
	}
	return domain.AuthSession{}, notFound("auth session", "")
}

func (s *MemoryStore) RevokeAuthSession(ctx context.Context, tokenHash string, userID domain.ID) error {
	tokenHash = strings.TrimSpace(tokenHash)
	if userID == "" {
		return ErrMissingUserID
	}
	if tokenHash == "" {
		return notFound("auth session", "")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	now := time.Now()
	for index := range data.sessions {
		if data.sessions[index].TokenHash == tokenHash {
			data.sessions[index].RevokedAt = &now
			return nil
		}
	}
	return notFound("auth session", "")
}

func (s *MemoryStore) UserProfile(ctx context.Context, userID domain.ID) (domain.UserProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.existingUserDataLocked(userID)
	if err != nil {
		return domain.UserProfile{}, err
	}
	return data.profile, nil
}

func (s *MemoryStore) UpdateUserProfile(ctx context.Context, userID domain.ID, profile domain.UserProfile) (domain.UserProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.existingUserDataLocked(userID)
	if err != nil {
		return domain.UserProfile{}, err
	}
	profile.ID = data.profile.ID
	profile.CreatedAt = data.profile.CreatedAt
	if profile.Name == "" {
		profile.Name = data.profile.Name
	}
	if profile.School == "" {
		profile.School = data.profile.School
	}
	if profile.Stage == "" {
		profile.Stage = data.profile.Stage
	}
	if profile.Subject == "" {
		profile.Subject = data.profile.Subject
	}
	if profile.ProStatus == "" {
		profile.ProStatus = data.profile.ProStatus
	}
	if profile.ReminderPolicy == "" {
		profile.ReminderPolicy = data.profile.ReminderPolicy
	}
	data.profile = profile
	return data.profile, nil
}

func (s *MemoryStore) DeleteUser(ctx context.Context, userID domain.ID) error {
	if userID == "" {
		return ErrMissingUserID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return notFound("user profile", userID)
	}
	for openID, mappedUserID := range s.wechatUsers {
		if mappedUserID == userID {
			delete(s.wechatUsers, openID)
		}
	}
	delete(s.users, userID)
	return nil
}

func (s *MemoryStore) ExportUserData(ctx context.Context, userID domain.ID) (domain.AccountExport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.existingUserDataLocked(userID)
	if err != nil {
		return domain.AccountExport{}, err
	}
	aiLogs := append([]domain.AIGeneration(nil), data.aiLogs...)
	for index := range aiLogs {
		aiLogs[index].Actions = actionsForGeneration(data.aiActions, aiLogs[index].ID)
	}
	return domain.AccountExport{
		Profile:              data.profile,
		Courses:              append([]domain.Course(nil), data.courses...),
		Reminders:            append([]domain.Reminder(nil), data.reminders...),
		Parents:              append([]domain.ParentProfile(nil), data.parents...),
		CommunicationRecords: append([]domain.CommunicationRecord(nil), data.records...),
		HealingEntries:       append([]domain.HealingEntry(nil), data.healing...),
		AIGenerations:        aiLogs,
		Favorites:            append([]domain.Favorite(nil), data.favorites...),
		ExportedAt:           time.Now(),
	}, nil
}

func (s *MemoryStore) Dashboard(ctx context.Context, userID domain.ID, day time.Time) (domain.DashboardSummary, error) {
	weekday := int(day.Weekday())
	courses, err := s.CoursesByDay(ctx, userID, weekday)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	reminders, err := s.RemindersByDay(ctx, userID, day)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	pendingReminders := pendingReminderCount(reminders)
	followUpParents, followUpsCount := s.dueFollowUps(ctx, userID, day)
	next := nextCourseForDay(courses, day)
	rhythm := domain.RhythmState{Code: "steady", Title: "温柔但高效", Description: "今天事项可控，先处理下一节课，再推进家长跟进。"}
	if len(courses)+pendingReminders >= 5 {
		rhythm = domain.RhythmState{Code: "busy", Title: "节奏偏满", Description: "课程和待办较集中，建议按课前、课后、沟通三段处理。"}
	}
	return domain.DashboardSummary{TodayLabel: day.Format("2006-01-02"), CoursesCount: len(courses), RemindersCount: pendingReminders, FollowUpsCount: followUpsCount, NextCourse: next, Reminders: reminders, FocusParents: followUpParents, Rhythm: rhythm}, nil
}

func (s *MemoryStore) dueFollowUps(ctx context.Context, userID domain.ID, day time.Time) ([]domain.ParentProfile, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	end := endOfDay(day)
	parentByID := make(map[domain.ID]domain.ParentProfile, len(data.parents))
	for _, parent := range data.parents {
		parentByID[parent.ID] = parent
	}
	seen := map[domain.ID]bool{}
	focusParents := make([]domain.ParentProfile, 0)
	followUpsCount := 0
	for _, item := range data.records {
		if item.FollowUpStatus == "done" || item.FollowUpAt.IsZero() || item.FollowUpAt.After(end) {
			continue
		}
		followUpsCount++
		if item.ParentID == nil || *item.ParentID == "" || seen[*item.ParentID] {
			continue
		}
		parent, ok := parentByID[*item.ParentID]
		if !ok {
			continue
		}
		seen[*item.ParentID] = true
		focusParents = append(focusParents, parent)
	}
	return focusParents, followUpsCount
}

func (s *MemoryStore) CoursesByDay(ctx context.Context, userID domain.ID, weekday int) ([]domain.Course, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	items := make([]domain.Course, 0)
	for _, item := range data.courses {
		if item.Weekday == weekday {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *MemoryStore) Course(ctx context.Context, userID domain.ID, id domain.ID) (domain.Course, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for _, item := range data.courses {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.Course{}, notFound("course", id)
}

func (s *MemoryStore) CreateCourse(ctx context.Context, userID domain.ID, course domain.Course) (domain.Course, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	s.seq++
	course.ID = domain.ID(fmt.Sprintf("course_%d", s.seq))
	course.CreatedAt = time.Now()
	if course.Weekday < 0 || course.Weekday > 6 {
		course.Weekday = int(time.Now().Weekday())
	}
	data.courses = append(data.courses, course)
	return course, nil
}

func (s *MemoryStore) UpdateCourse(ctx context.Context, userID domain.ID, id domain.ID, course domain.Course) (domain.Course, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for index := range data.courses {
		if data.courses[index].ID == id {
			course.ID = id
			course.CreatedAt = data.courses[index].CreatedAt
			data.courses[index] = course
			return course, nil
		}
	}
	return domain.Course{}, notFound("course", id)
}

func (s *MemoryStore) DeleteCourse(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for index := range data.courses {
		if data.courses[index].ID == id {
			data.courses = append(data.courses[:index], data.courses[index+1:]...)
			return nil
		}
	}
	return notFound("course", id)
}

func (s *MemoryStore) RemindersByDay(ctx context.Context, userID domain.ID, day time.Time) ([]domain.Reminder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	items := make([]domain.Reminder, 0)
	for _, item := range data.reminders {
		if sameDay(item.RemindAt, day) && item.Status != "deleted" {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *MemoryStore) Reminder(ctx context.Context, userID domain.ID, id domain.ID) (domain.Reminder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for _, item := range data.reminders {
		if item.ID == id && item.Status != "deleted" {
			return item, nil
		}
	}
	return domain.Reminder{}, notFound("reminder", id)
}

func (s *MemoryStore) CreateReminder(ctx context.Context, userID domain.ID, reminder domain.Reminder) (domain.Reminder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	s.seq++
	reminder.ID = domain.ID(fmt.Sprintf("reminder_%d", s.seq))
	reminder.CreatedAt = time.Now()
	if reminder.Status == "" {
		reminder.Status = "pending"
	}
	data.reminders = append([]domain.Reminder{reminder}, data.reminders...)
	return reminder, nil
}

func (s *MemoryStore) UpdateReminder(ctx context.Context, userID domain.ID, id domain.ID, reminder domain.Reminder) (domain.Reminder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for index := range data.reminders {
		if data.reminders[index].ID == id && data.reminders[index].Status != "deleted" {
			reminder.ID = id
			reminder.CreatedAt = data.reminders[index].CreatedAt
			if reminder.Status == "" {
				reminder.Status = data.reminders[index].Status
			}
			data.reminders[index] = reminder
			return reminder, nil
		}
	}
	return domain.Reminder{}, notFound("reminder", id)
}

func (s *MemoryStore) CompleteReminder(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	now := time.Now()
	for index := range data.reminders {
		if data.reminders[index].ID == id && data.reminders[index].Status != "deleted" {
			data.reminders[index].Status = "done"
			data.reminders[index].DoneAt = &now
			return nil
		}
	}
	return notFound("reminder", id)
}

func (s *MemoryStore) DeleteReminder(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for index := range data.reminders {
		if data.reminders[index].ID == id && data.reminders[index].Status != "deleted" {
			data.reminders[index].Status = "deleted"
			return nil
		}
	}
	return notFound("reminder", id)
}

func (s *MemoryStore) SnoozeReminder(ctx context.Context, userID domain.ID, id domain.ID, until time.Time) (domain.Reminder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for index := range data.reminders {
		if data.reminders[index].ID == id && data.reminders[index].Status != "deleted" {
			data.reminders[index].Status = "snoozed"
			data.reminders[index].RemindAt = until
			data.reminders[index].DoneAt = nil
			return data.reminders[index], nil
		}
	}
	return domain.Reminder{}, notFound("reminder", id)
}

func (s *MemoryStore) Parents(ctx context.Context, userID domain.ID, options ListOptions) ([]domain.ParentProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	items := make([]domain.ParentProfile, 0, len(data.parents))
	query := normalizeQuery(options.Query)
	for _, item := range data.parents {
		if query == "" || containsAll(query, item.StudentName, item.ClassName, item.ParentName, item.Relationship, item.Contact, item.CommunicationStyle, item.RiskLevel, item.ImportantNotes, item.NextAction) {
			items = append(items, item)
		}
	}
	items = paginate(items, options)
	return items, nil
}

func (s *MemoryStore) Parent(ctx context.Context, userID domain.ID, id domain.ID) (domain.ParentProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for _, item := range data.parents {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.ParentProfile{}, notFound("parent", id)
}

func (s *MemoryStore) CreateParent(ctx context.Context, userID domain.ID, parent domain.ParentProfile) (domain.ParentProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	s.seq++
	parent.ID = domain.ID(fmt.Sprintf("parent_%d", s.seq))
	parent.CreatedAt = time.Now()
	data.parents = append([]domain.ParentProfile{parent}, data.parents...)
	return parent, nil
}

func (s *MemoryStore) UpdateParent(ctx context.Context, userID domain.ID, id domain.ID, parent domain.ParentProfile) (domain.ParentProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for index := range data.parents {
		if data.parents[index].ID == id {
			parent.ID = id
			parent.CreatedAt = data.parents[index].CreatedAt
			if parent.RiskLevel == "" {
				parent.RiskLevel = "low"
			}
			data.parents[index] = parent
			return parent, nil
		}
	}
	return domain.ParentProfile{}, notFound("parent", id)
}

func (s *MemoryStore) DeleteParent(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for index := range data.parents {
		if data.parents[index].ID == id {
			data.parents = append(data.parents[:index], data.parents[index+1:]...)
			for index := range data.reminders {
				if data.reminders[index].ParentID != nil && *data.reminders[index].ParentID == id {
					data.reminders[index].ParentID = nil
				}
			}
			for index := range data.records {
				if data.records[index].ParentID != nil && *data.records[index].ParentID == id {
					data.records[index].ParentID = nil
				}
			}
			return nil
		}
	}
	return notFound("parent", id)
}

func (s *MemoryStore) CommunicationRecords(ctx context.Context, userID domain.ID, parentID *domain.ID, options ListOptions) ([]domain.CommunicationRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	items := make([]domain.CommunicationRecord, 0)
	query := normalizeQuery(options.Query)
	for _, item := range data.records {
		if (parentID == nil || (item.ParentID != nil && *item.ParentID == *parentID)) && (query == "" || containsAll(query, item.Student, item.Channel, item.Reason, item.Summary, item.Result, item.RiskLevel)) {
			items = append(items, item)
		}
	}
	items = paginate(items, options)
	return items, nil
}

func (s *MemoryStore) CommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID) (domain.CommunicationRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for _, item := range data.records {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.CommunicationRecord{}, notFound("communication record", id)
}

func (s *MemoryStore) CreateCommunicationRecord(ctx context.Context, userID domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	s.seq++
	record.ID = domain.ID(fmt.Sprintf("record_%d", s.seq))
	record.CreatedAt = time.Now()
	if record.RiskLevel == "" {
		record.RiskLevel = "low"
	}
	if record.FollowUpStatus == "" {
		record.FollowUpStatus = "pending"
	}
	if record.FollowUpStatus == "done" && record.FollowedUpAt == nil {
		now := time.Now()
		record.FollowedUpAt = &now
	}
	data.records = append([]domain.CommunicationRecord{record}, data.records...)
	return record, nil
}

func (s *MemoryStore) UpdateCommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for index := range data.records {
		if data.records[index].ID == id {
			record.ID = id
			record.CreatedAt = data.records[index].CreatedAt
			if record.RiskLevel == "" {
				record.RiskLevel = "low"
			}
			if record.FollowUpStatus == "" {
				record.FollowUpStatus = data.records[index].FollowUpStatus
			}
			if record.FollowUpStatus == "" {
				record.FollowUpStatus = "pending"
			}
			if record.FollowUpStatus == "done" && record.FollowedUpAt == nil {
				record.FollowedUpAt = data.records[index].FollowedUpAt
			}
			if record.FollowUpStatus == "done" && record.FollowedUpAt == nil {
				now := time.Now()
				record.FollowedUpAt = &now
			}
			if record.FollowUpStatus != "done" {
				record.FollowedUpAt = nil
			}
			data.records[index] = record
			return record, nil
		}
	}
	return domain.CommunicationRecord{}, notFound("communication record", id)
}

func (s *MemoryStore) CompleteCommunicationFollowUp(ctx context.Context, userID domain.ID, id domain.ID) (domain.CommunicationRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	now := time.Now()
	for index := range data.records {
		if data.records[index].ID == id {
			data.records[index].FollowUpStatus = "done"
			data.records[index].FollowedUpAt = &now
			return data.records[index], nil
		}
	}
	return domain.CommunicationRecord{}, notFound("communication record", id)
}

func (s *MemoryStore) DeleteCommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for index := range data.records {
		if data.records[index].ID == id {
			data.records = append(data.records[:index], data.records[index+1:]...)
			return nil
		}
	}
	return notFound("communication record", id)
}

func (s *MemoryStore) HealingEntries(ctx context.Context, userID domain.ID, entryType string, options ListOptions) ([]domain.HealingEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	items := make([]domain.HealingEntry, 0)
	query := normalizeQuery(options.Query)
	for _, item := range data.healing {
		if (entryType == "" || item.Type == entryType) && (query == "" || containsAll(query, item.Type, item.Mood, item.Content, item.AIReply)) {
			items = append(items, item)
		}
	}
	items = paginate(items, options)
	return items, nil
}

func (s *MemoryStore) HealingEntry(ctx context.Context, userID domain.ID, id domain.ID) (domain.HealingEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for _, item := range data.healing {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.HealingEntry{}, notFound("healing entry", id)
}

func (s *MemoryStore) CreateHealingEntry(ctx context.Context, userID domain.ID, entry domain.HealingEntry) (domain.HealingEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	s.seq++
	entry.ID = domain.ID(fmt.Sprintf("healing_%d", s.seq))
	entry.CreatedAt = time.Now()
	data.healing = append([]domain.HealingEntry{entry}, data.healing...)
	return entry, nil
}

func (s *MemoryStore) DeleteHealingEntry(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for index := range data.healing {
		if data.healing[index].ID == id {
			data.healing = append(data.healing[:index], data.healing[index+1:]...)
			return nil
		}
	}
	return notFound("healing entry", id)
}

func (s *MemoryStore) AIGenerations(ctx context.Context, userID domain.ID, scenario string, options ListOptions) ([]domain.AIGeneration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	items := make([]domain.AIGeneration, 0)
	query := normalizeQuery(options.Query)
	for _, item := range data.aiLogs {
		if (scenario == "" || item.Scenario == scenario) && (query == "" || containsAll(query, item.Scenario, item.SafetyLabel, fmt.Sprint(item.Input), fmt.Sprint(item.Output))) {
			item.Actions = actionsForGeneration(data.aiActions, item.ID)
			items = append(items, item)
		}
	}
	items = paginate(items, options)
	return items, nil
}

func (s *MemoryStore) AIGeneration(ctx context.Context, userID domain.ID, id domain.ID) (domain.AIGeneration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for _, item := range data.aiLogs {
		if item.ID == id {
			item.Actions = actionsForGeneration(data.aiActions, item.ID)
			return item, nil
		}
	}
	return domain.AIGeneration{}, notFound("ai generation", id)
}

func (s *MemoryStore) CreateAIGeneration(ctx context.Context, userID domain.ID, generation domain.AIGeneration) (domain.AIGeneration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	s.seq++
	generation.ID = domain.ID(fmt.Sprintf("ai_generation_%d", s.seq))
	generation.CreatedAt = time.Now()
	if generation.Input == nil {
		generation.Input = map[string]any{}
	}
	if generation.Output == nil {
		generation.Output = map[string]any{}
	}
	if generation.SafetyLabel == "" {
		generation.SafetyLabel = "teacher_review_required"
	}
	data.aiLogs = append([]domain.AIGeneration{generation}, data.aiLogs...)
	return generation, nil
}

func (s *MemoryStore) UpdateAIGenerationOutput(ctx context.Context, userID domain.ID, id domain.ID, output map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	if output == nil {
		output = map[string]any{}
	}
	for index := range data.aiLogs {
		if data.aiLogs[index].ID == id {
			data.aiLogs[index].Output = output
			return nil
		}
	}
	return notFound("ai generation", id)
}

func (s *MemoryStore) DeleteAIGeneration(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for index := range data.aiLogs {
		if data.aiLogs[index].ID == id {
			data.aiLogs = append(data.aiLogs[:index], data.aiLogs[index+1:]...)
			filteredActions := data.aiActions[:0]
			for _, action := range data.aiActions {
				if action.GenerationID != id {
					filteredActions = append(filteredActions, action)
				}
			}
			data.aiActions = filteredActions
			return nil
		}
	}
	return notFound("ai generation", id)
}

func (s *MemoryStore) CreateAIAction(ctx context.Context, userID domain.ID, action domain.AIAction) (domain.AIAction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	found := false
	for _, generation := range data.aiLogs {
		if generation.ID == action.GenerationID {
			found = true
			break
		}
	}
	if !found {
		return domain.AIAction{}, notFound("ai generation", action.GenerationID)
	}
	s.seq++
	action.ID = domain.ID(fmt.Sprintf("ai_action_%d", s.seq))
	action.CreatedAt = time.Now()
	if action.Metadata == nil {
		action.Metadata = map[string]any{}
	}
	data.aiActions = append(data.aiActions, action)
	return action, nil
}

func (s *MemoryStore) AIActions(ctx context.Context, userID domain.ID, generationID domain.ID) ([]domain.AIAction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	found := false
	for _, generation := range data.aiLogs {
		if generation.ID == generationID {
			found = true
			break
		}
	}
	if !found {
		return nil, notFound("ai generation", generationID)
	}
	return actionsForGeneration(data.aiActions, generationID), nil
}

func actionsForGeneration(actions []domain.AIAction, generationID domain.ID) []domain.AIAction {
	items := make([]domain.AIAction, 0)
	for _, action := range actions {
		if action.GenerationID == generationID {
			items = append(items, action)
		}
	}
	return items
}

func (s *MemoryStore) Favorites(ctx context.Context, userID domain.ID, favoriteType string, options ListOptions) ([]domain.Favorite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	items := make([]domain.Favorite, 0)
	query := normalizeQuery(options.Query)
	for _, item := range data.favorites {
		if (favoriteType == "" || item.Type == favoriteType) && (query == "" || containsAll(query, item.Type, item.Title, item.Content)) {
			items = append(items, item)
		}
	}
	items = paginate(items, options)
	return items, nil
}

func (s *MemoryStore) CreateFavorite(ctx context.Context, userID domain.ID, favorite domain.Favorite) (domain.Favorite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	s.seq++
	favorite.ID = domain.ID(fmt.Sprintf("favorite_%d", s.seq))
	favorite.CreatedAt = time.Now()
	data.favorites = append([]domain.Favorite{favorite}, data.favorites...)
	return favorite, nil
}

func (s *MemoryStore) DeleteFavorite(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.userDataLocked(userID)
	for index := range data.favorites {
		if data.favorites[index].ID == id {
			data.favorites = append(data.favorites[:index], data.favorites[index+1:]...)
			return nil
		}
	}
	return notFound("favorite", id)
}

func (s *MemoryStore) userDataLocked(userID domain.ID) *memoryUserData {
	if userID == "" {
		panic(ErrMissingUserID)
	}
	if s.users == nil {
		s.users = make(map[domain.ID]*memoryUserData)
	}
	if data, ok := s.users[userID]; ok {
		return data
	}
	data := &memoryUserData{profile: defaultUserProfile(userID, "", time.Now())}
	s.users[userID] = data
	return data
}

func (s *MemoryStore) existingUserDataLocked(userID domain.ID) (*memoryUserData, error) {
	if userID == "" {
		return nil, ErrMissingUserID
	}
	if data, ok := s.users[userID]; ok {
		return data, nil
	}
	return nil, notFound("user profile", userID)
}

func defaultUserProfile(userID domain.ID, nickName string, createdAt time.Time) domain.UserProfile {
	name := strings.TrimSpace(nickName)
	if name == "" {
		name = "微光老师"
	}
	return domain.UserProfile{
		ID:             userID,
		Name:           name,
		ProStatus:      "free",
		ReminderPolicy: "low_interrupt",
		CreatedAt:      createdAt,
	}
}

func nextCourseForDay(courses []domain.Course, day time.Time) *domain.Course {
	if len(courses) == 0 {
		return nil
	}
	nowMinutes := 0
	if sameCalendarDay(day, time.Now()) {
		nowMinutes = time.Now().Hour()*60 + time.Now().Minute()
	}
	nextIndex := -1
	nextStart := 24 * 60
	for index := range courses {
		if clockMinutes(courses[index].EndTime) <= nowMinutes {
			continue
		}
		startMinutes := clockMinutes(courses[index].StartTime)
		if nextIndex == -1 || startMinutes < nextStart {
			nextIndex = index
			nextStart = startMinutes
		}
	}
	if nextIndex == -1 {
		return nil
	}
	return &courses[nextIndex]
}

func sameCalendarDay(left time.Time, right time.Time) bool {
	leftYear, leftMonth, leftDay := left.Date()
	rightYear, rightMonth, rightDay := right.Date()
	return leftYear == rightYear && leftMonth == rightMonth && leftDay == rightDay
}

func clockMinutes(value string) int {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed.Hour()*60 + parsed.Minute()
}

func normalizeQuery(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func containsAll(query string, values ...string) bool {
	if query == "" {
		return true
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func paginate[T any](items []T, options ListOptions) []T {
	limit := options.Limit
	if limit <= 0 || limit > MaxListProbeLimit {
		limit = DefaultListLimit
	}
	offset := options.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []T{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func atToday(value string) time.Time {
	now := time.Now()
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return now
	}
	return time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func notFound(kind string, id domain.ID) error {
	return fmt.Errorf("%w: %s %s", ErrNotFound, kind, id)
}

var _ Store = (*MemoryStore)(nil)
