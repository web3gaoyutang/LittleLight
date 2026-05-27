package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	UserProfile(ctx context.Context, userID domain.ID) (domain.UserProfile, error)
	UpdateUserProfile(ctx context.Context, userID domain.ID, profile domain.UserProfile) (domain.UserProfile, error)
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
	Parents(ctx context.Context, userID domain.ID) ([]domain.ParentProfile, error)
	Parent(ctx context.Context, userID domain.ID, id domain.ID) (domain.ParentProfile, error)
	CreateParent(ctx context.Context, userID domain.ID, parent domain.ParentProfile) (domain.ParentProfile, error)
	UpdateParent(ctx context.Context, userID domain.ID, id domain.ID, parent domain.ParentProfile) (domain.ParentProfile, error)
	DeleteParent(ctx context.Context, userID domain.ID, id domain.ID) error
	CommunicationRecords(ctx context.Context, userID domain.ID, parentID *domain.ID) ([]domain.CommunicationRecord, error)
	CommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID) (domain.CommunicationRecord, error)
	CreateCommunicationRecord(ctx context.Context, userID domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error)
	UpdateCommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error)
	DeleteCommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID) error
	HealingEntries(ctx context.Context, userID domain.ID, entryType string) ([]domain.HealingEntry, error)
	HealingEntry(ctx context.Context, userID domain.ID, id domain.ID) (domain.HealingEntry, error)
	CreateHealingEntry(ctx context.Context, userID domain.ID, entry domain.HealingEntry) (domain.HealingEntry, error)
	DeleteHealingEntry(ctx context.Context, userID domain.ID, id domain.ID) error
	AIGenerations(ctx context.Context, userID domain.ID, scenario string) ([]domain.AIGeneration, error)
	AIGeneration(ctx context.Context, userID domain.ID, id domain.ID) (domain.AIGeneration, error)
	CreateAIGeneration(ctx context.Context, userID domain.ID, generation domain.AIGeneration) (domain.AIGeneration, error)
	Favorites(ctx context.Context, userID domain.ID, favoriteType string) ([]domain.Favorite, error)
	CreateFavorite(ctx context.Context, userID domain.ID, favorite domain.Favorite) (domain.Favorite, error)
	DeleteFavorite(ctx context.Context, userID domain.ID, id domain.ID) error
}

type MemoryStore struct {
	mu        sync.RWMutex
	courses   []domain.Course
	reminders []domain.Reminder
	parents   []domain.ParentProfile
	records   []domain.CommunicationRecord
	healing   []domain.HealingEntry
	aiLogs    []domain.AIGeneration
	favorites []domain.Favorite
	profile   domain.UserProfile
	seq       int64
}

func NewMemoryStore() *MemoryStore {
	now := time.Now()
	parent1 := domain.ParentProfile{ID: "parent_lin", StudentName: "林晓晓", ClassName: "高二(5)班", ParentName: "林晓晓妈妈", Relationship: "母亲", CommunicationStyle: "比较敏感", RiskLevel: "medium", ImportantNotes: "睡眠与到校状态需要持续观察。", NextAction: "先确认睡眠，再同步课堂参与中的积极信号。", CreatedAt: now}
	parent2 := domain.ParentProfile{ID: "parent_chen", StudentName: "陈子默", ClassName: "高二(3)班", ParentName: "陈子默爸爸", Relationship: "父亲", CommunicationStyle: "关注成绩", RiskLevel: "low", ImportantNotes: "关注测试反馈和订正节奏。", NextAction: "周五前同步订正计划。", CreatedAt: now}
	return &MemoryStore{
		seq: 100,
		profile: domain.UserProfile{
			ID:             "00000000-0000-0000-0000-000000000001",
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
			{ID: "record_chen", ParentID: parent2.ID, Student: "陈子默", Channel: "微信", Reason: "测试反馈", Summary: "已说明测试问题与订正方向。", Result: "家长认可 3 天订正计划。", RiskLevel: "low", FollowUpAt: now.Add(72 * time.Hour), CreatedAt: now.Add(-24 * time.Hour)},
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
}

func (s *MemoryStore) UserProfile(ctx context.Context, userID domain.ID) (domain.UserProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.profile, nil
}

func (s *MemoryStore) UpdateUserProfile(ctx context.Context, userID domain.ID, profile domain.UserProfile) (domain.UserProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	profile.ID = s.profile.ID
	profile.CreatedAt = s.profile.CreatedAt
	if profile.Name == "" {
		profile.Name = s.profile.Name
	}
	if profile.School == "" {
		profile.School = s.profile.School
	}
	if profile.Stage == "" {
		profile.Stage = s.profile.Stage
	}
	if profile.Subject == "" {
		profile.Subject = s.profile.Subject
	}
	if profile.ProStatus == "" {
		profile.ProStatus = s.profile.ProStatus
	}
	if profile.ReminderPolicy == "" {
		profile.ReminderPolicy = s.profile.ReminderPolicy
	}
	s.profile = profile
	return s.profile, nil
}

func (s *MemoryStore) Dashboard(ctx context.Context, userID domain.ID, day time.Time) (domain.DashboardSummary, error) {
	weekday := int(day.Weekday())
	courses, _ := s.CoursesByDay(ctx, userID, weekday)
	reminders, _ := s.RemindersByDay(ctx, userID, day)
	parents, _ := s.Parents(ctx, userID)
	var next *domain.Course
	if len(courses) > 0 {
		next = &courses[0]
	}
	rhythm := domain.RhythmState{Code: "steady", Title: "温柔但高效", Description: "今天事项可控，先处理下一节课，再推进家长跟进。"}
	if len(courses)+len(reminders) >= 5 {
		rhythm = domain.RhythmState{Code: "busy", Title: "节奏偏满", Description: "课程和待办较集中，建议按课前、课后、沟通三段处理。"}
	}
	return domain.DashboardSummary{TodayLabel: day.Format("2006-01-02"), CoursesCount: len(courses), RemindersCount: len(reminders), FollowUpsCount: len(parents), NextCourse: next, Reminders: reminders, FocusParents: parents, Rhythm: rhythm}, nil
}

func (s *MemoryStore) CoursesByDay(ctx context.Context, userID domain.ID, weekday int) ([]domain.Course, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Course, 0)
	for _, item := range s.courses {
		if item.Weekday == weekday {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *MemoryStore) Course(ctx context.Context, userID domain.ID, id domain.ID) (domain.Course, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.courses {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.Course{}, notFound("course", id)
}

func (s *MemoryStore) CreateCourse(ctx context.Context, userID domain.ID, course domain.Course) (domain.Course, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	course.ID = domain.ID(fmt.Sprintf("course_%d", s.seq))
	course.CreatedAt = time.Now()
	if course.Weekday < 0 || course.Weekday > 6 {
		course.Weekday = int(time.Now().Weekday())
	}
	s.courses = append(s.courses, course)
	return course, nil
}

func (s *MemoryStore) UpdateCourse(ctx context.Context, userID domain.ID, id domain.ID, course domain.Course) (domain.Course, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.courses {
		if s.courses[index].ID == id {
			course.ID = id
			course.CreatedAt = s.courses[index].CreatedAt
			s.courses[index] = course
			return course, nil
		}
	}
	return domain.Course{}, notFound("course", id)
}

func (s *MemoryStore) DeleteCourse(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.courses {
		if s.courses[index].ID == id {
			s.courses = append(s.courses[:index], s.courses[index+1:]...)
			return nil
		}
	}
	return notFound("course", id)
}

func (s *MemoryStore) RemindersByDay(ctx context.Context, userID domain.ID, day time.Time) ([]domain.Reminder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Reminder, 0)
	for _, item := range s.reminders {
		if sameDay(item.RemindAt, day) && item.Status != "deleted" {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *MemoryStore) Reminder(ctx context.Context, userID domain.ID, id domain.ID) (domain.Reminder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.reminders {
		if item.ID == id && item.Status != "deleted" {
			return item, nil
		}
	}
	return domain.Reminder{}, notFound("reminder", id)
}

func (s *MemoryStore) CreateReminder(ctx context.Context, userID domain.ID, reminder domain.Reminder) (domain.Reminder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	reminder.ID = domain.ID(fmt.Sprintf("reminder_%d", s.seq))
	reminder.CreatedAt = time.Now()
	if reminder.Status == "" {
		reminder.Status = "pending"
	}
	s.reminders = append([]domain.Reminder{reminder}, s.reminders...)
	return reminder, nil
}

func (s *MemoryStore) UpdateReminder(ctx context.Context, userID domain.ID, id domain.ID, reminder domain.Reminder) (domain.Reminder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.reminders {
		if s.reminders[index].ID == id && s.reminders[index].Status != "deleted" {
			reminder.ID = id
			reminder.CreatedAt = s.reminders[index].CreatedAt
			if reminder.Status == "" {
				reminder.Status = s.reminders[index].Status
			}
			s.reminders[index] = reminder
			return reminder, nil
		}
	}
	return domain.Reminder{}, notFound("reminder", id)
}

func (s *MemoryStore) CompleteReminder(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for index := range s.reminders {
		if s.reminders[index].ID == id && s.reminders[index].Status != "deleted" {
			s.reminders[index].Status = "done"
			s.reminders[index].DoneAt = &now
			return nil
		}
	}
	return notFound("reminder", id)
}

func (s *MemoryStore) DeleteReminder(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.reminders {
		if s.reminders[index].ID == id && s.reminders[index].Status != "deleted" {
			s.reminders[index].Status = "deleted"
			return nil
		}
	}
	return notFound("reminder", id)
}

func (s *MemoryStore) SnoozeReminder(ctx context.Context, userID domain.ID, id domain.ID, until time.Time) (domain.Reminder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.reminders {
		if s.reminders[index].ID == id && s.reminders[index].Status != "deleted" {
			s.reminders[index].Status = "snoozed"
			s.reminders[index].RemindAt = until
			s.reminders[index].DoneAt = nil
			return s.reminders[index], nil
		}
	}
	return domain.Reminder{}, notFound("reminder", id)
}

func (s *MemoryStore) Parents(ctx context.Context, userID domain.ID) ([]domain.ParentProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := append([]domain.ParentProfile(nil), s.parents...)
	return items, nil
}

func (s *MemoryStore) Parent(ctx context.Context, userID domain.ID, id domain.ID) (domain.ParentProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.parents {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.ParentProfile{}, notFound("parent", id)
}

func (s *MemoryStore) CreateParent(ctx context.Context, userID domain.ID, parent domain.ParentProfile) (domain.ParentProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	parent.ID = domain.ID(fmt.Sprintf("parent_%d", s.seq))
	parent.CreatedAt = time.Now()
	s.parents = append([]domain.ParentProfile{parent}, s.parents...)
	return parent, nil
}

func (s *MemoryStore) UpdateParent(ctx context.Context, userID domain.ID, id domain.ID, parent domain.ParentProfile) (domain.ParentProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.parents {
		if s.parents[index].ID == id {
			parent.ID = id
			parent.CreatedAt = s.parents[index].CreatedAt
			if parent.RiskLevel == "" {
				parent.RiskLevel = "low"
			}
			s.parents[index] = parent
			return parent, nil
		}
	}
	return domain.ParentProfile{}, notFound("parent", id)
}

func (s *MemoryStore) DeleteParent(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.parents {
		if s.parents[index].ID == id {
			s.parents = append(s.parents[:index], s.parents[index+1:]...)
			return nil
		}
	}
	return notFound("parent", id)
}

func (s *MemoryStore) CommunicationRecords(ctx context.Context, userID domain.ID, parentID *domain.ID) ([]domain.CommunicationRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.CommunicationRecord, 0)
	for _, item := range s.records {
		if parentID == nil || item.ParentID == *parentID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *MemoryStore) CommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID) (domain.CommunicationRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.records {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.CommunicationRecord{}, notFound("communication record", id)
}

func (s *MemoryStore) CreateCommunicationRecord(ctx context.Context, userID domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	record.ID = domain.ID(fmt.Sprintf("record_%d", s.seq))
	record.CreatedAt = time.Now()
	s.records = append([]domain.CommunicationRecord{record}, s.records...)
	return record, nil
}

func (s *MemoryStore) UpdateCommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.records {
		if s.records[index].ID == id {
			record.ID = id
			record.CreatedAt = s.records[index].CreatedAt
			if record.RiskLevel == "" {
				record.RiskLevel = "low"
			}
			s.records[index] = record
			return record, nil
		}
	}
	return domain.CommunicationRecord{}, notFound("communication record", id)
}

func (s *MemoryStore) DeleteCommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.records {
		if s.records[index].ID == id {
			s.records = append(s.records[:index], s.records[index+1:]...)
			return nil
		}
	}
	return notFound("communication record", id)
}

func (s *MemoryStore) HealingEntries(ctx context.Context, userID domain.ID, entryType string) ([]domain.HealingEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.HealingEntry, 0)
	for _, item := range s.healing {
		if entryType == "" || item.Type == entryType {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *MemoryStore) HealingEntry(ctx context.Context, userID domain.ID, id domain.ID) (domain.HealingEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.healing {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.HealingEntry{}, notFound("healing entry", id)
}

func (s *MemoryStore) CreateHealingEntry(ctx context.Context, userID domain.ID, entry domain.HealingEntry) (domain.HealingEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	entry.ID = domain.ID(fmt.Sprintf("healing_%d", s.seq))
	entry.CreatedAt = time.Now()
	s.healing = append([]domain.HealingEntry{entry}, s.healing...)
	return entry, nil
}

func (s *MemoryStore) DeleteHealingEntry(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.healing {
		if s.healing[index].ID == id {
			s.healing = append(s.healing[:index], s.healing[index+1:]...)
			return nil
		}
	}
	return notFound("healing entry", id)
}

func (s *MemoryStore) AIGenerations(ctx context.Context, userID domain.ID, scenario string) ([]domain.AIGeneration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.AIGeneration, 0)
	for _, item := range s.aiLogs {
		if scenario == "" || item.Scenario == scenario {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *MemoryStore) AIGeneration(ctx context.Context, userID domain.ID, id domain.ID) (domain.AIGeneration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.aiLogs {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.AIGeneration{}, notFound("ai generation", id)
}

func (s *MemoryStore) CreateAIGeneration(ctx context.Context, userID domain.ID, generation domain.AIGeneration) (domain.AIGeneration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	s.aiLogs = append([]domain.AIGeneration{generation}, s.aiLogs...)
	return generation, nil
}

func (s *MemoryStore) Favorites(ctx context.Context, userID domain.ID, favoriteType string) ([]domain.Favorite, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Favorite, 0)
	for _, item := range s.favorites {
		if favoriteType == "" || item.Type == favoriteType {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *MemoryStore) CreateFavorite(ctx context.Context, userID domain.ID, favorite domain.Favorite) (domain.Favorite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	favorite.ID = domain.ID(fmt.Sprintf("favorite_%d", s.seq))
	favorite.CreatedAt = time.Now()
	s.favorites = append([]domain.Favorite{favorite}, s.favorites...)
	return favorite, nil
}

func (s *MemoryStore) DeleteFavorite(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.favorites {
		if s.favorites[index].ID == id {
			s.favorites = append(s.favorites[:index], s.favorites[index+1:]...)
			return nil
		}
	}
	return notFound("favorite", id)
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

