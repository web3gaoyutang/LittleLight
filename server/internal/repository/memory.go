package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

type Store interface {
	Dashboard(ctx context.Context, userID domain.ID, day time.Time) (domain.DashboardSummary, error)
	CoursesByDay(ctx context.Context, userID domain.ID, weekday int) ([]domain.Course, error)
	RemindersByDay(ctx context.Context, userID domain.ID, day time.Time) ([]domain.Reminder, error)
	CreateReminder(ctx context.Context, userID domain.ID, reminder domain.Reminder) (domain.Reminder, error)
	CompleteReminder(ctx context.Context, userID domain.ID, id domain.ID) error
	Parents(ctx context.Context, userID domain.ID) ([]domain.ParentProfile, error)
	CreateParent(ctx context.Context, userID domain.ID, parent domain.ParentProfile) (domain.ParentProfile, error)
	CommunicationRecords(ctx context.Context, userID domain.ID, parentID *domain.ID) ([]domain.CommunicationRecord, error)
	CreateCommunicationRecord(ctx context.Context, userID domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error)
	CreateHealingEntry(ctx context.Context, userID domain.ID, entry domain.HealingEntry) (domain.HealingEntry, error)
}

type MemoryStore struct {
	mu        sync.RWMutex
	courses   []domain.Course
	reminders []domain.Reminder
	parents   []domain.ParentProfile
	records   []domain.CommunicationRecord
	healing   []domain.HealingEntry
	seq       int64
}

func NewMemoryStore() *MemoryStore {
	now := time.Now()
	parent1 := domain.ParentProfile{ID: "parent_lin", StudentName: "林晓晓", ClassName: "高二(5)班", ParentName: "林晓晓妈妈", Relationship: "母亲", CommunicationStyle: "比较敏感", RiskLevel: "medium", ImportantNotes: "睡眠与到校状态需要持续观察。", NextAction: "先确认睡眠，再同步课堂参与中的积极信号。", CreatedAt: now}
	parent2 := domain.ParentProfile{ID: "parent_chen", StudentName: "陈子默", ClassName: "高二(3)班", ParentName: "陈子默爸爸", Relationship: "父亲", CommunicationStyle: "关注成绩", RiskLevel: "low", ImportantNotes: "关注测试反馈和订正节奏。", NextAction: "周五前同步订正计划。", CreatedAt: now}
	return &MemoryStore{
		seq: 100,
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
	}
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

func (s *MemoryStore) CompleteReminder(ctx context.Context, userID domain.ID, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for index := range s.reminders {
		if s.reminders[index].ID == id {
			s.reminders[index].Status = "done"
			s.reminders[index].DoneAt = &now
			return nil
		}
	}
	return fmt.Errorf("reminder not found: %s", id)
}

func (s *MemoryStore) Parents(ctx context.Context, userID domain.ID) ([]domain.ParentProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := append([]domain.ParentProfile(nil), s.parents...)
	return items, nil
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

func (s *MemoryStore) CreateCommunicationRecord(ctx context.Context, userID domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	record.ID = domain.ID(fmt.Sprintf("record_%d", s.seq))
	record.CreatedAt = time.Now()
	s.records = append([]domain.CommunicationRecord{record}, s.records...)
	return record, nil
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

