package repository

import (
	"context"
	"testing"
	"time"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

const testUserID = domain.ID("00000000-0000-0000-0000-000000000001")

func TestMemoryStoreDashboardAndReminderCompletion(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	summary, err := store.Dashboard(ctx, testUserID, time.Now())
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if summary.CoursesCount == 0 {
		t.Fatalf("expected seeded courses")
	}
	if summary.RemindersCount == 0 {
		t.Fatalf("expected seeded reminders")
	}
	if summary.NextCourse == nil {
		t.Fatalf("expected next course")
	}

	reminder, err := store.CreateReminder(ctx, testUserID, domain.Reminder{Title: "测试提醒", Category: "个人事项", RemindAt: time.Now().Add(time.Hour), Note: "测试"})
	if err != nil {
		t.Fatalf("create reminder: %v", err)
	}
	if reminder.ID == "" || reminder.Status != "pending" {
		t.Fatalf("unexpected reminder: %+v", reminder)
	}
	if err := store.CompleteReminder(ctx, testUserID, reminder.ID); err != nil {
		t.Fatalf("complete reminder: %v", err)
	}

	updated, err := store.UpdateReminder(ctx, testUserID, reminder.ID, domain.Reminder{Title: "更新后的提醒", Category: "个人事项", RemindAt: time.Now().Add(2 * time.Hour), Status: "pending", Note: "已编辑"})
	if err != nil {
		t.Fatalf("update reminder: %v", err)
	}
	if updated.Title != "更新后的提醒" {
		t.Fatalf("unexpected updated reminder: %+v", updated)
	}
	snoozed, err := store.SnoozeReminder(ctx, testUserID, reminder.ID, time.Now().Add(3*time.Hour))
	if err != nil {
		t.Fatalf("snooze reminder: %v", err)
	}
	if snoozed.Status != "snoozed" {
		t.Fatalf("expected snoozed reminder: %+v", snoozed)
	}
	if err := store.DeleteReminder(ctx, testUserID, reminder.ID); err != nil {
		t.Fatalf("delete reminder: %v", err)
	}
	if _, err := store.Reminder(ctx, testUserID, reminder.ID); err == nil {
		t.Fatalf("expected deleted reminder to be hidden")
	}
}

func TestMemoryStoreParentsAndRecords(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	parents, err := store.Parents(ctx, testUserID)
	if err != nil {
		t.Fatalf("parents: %v", err)
	}
	if len(parents) < 2 {
		t.Fatalf("expected seeded parents")
	}
	record, err := store.CreateCommunicationRecord(ctx, testUserID, domain.CommunicationRecord{ParentID: parents[0].ID, Student: parents[0].StudentName, Channel: "微信", Reason: "测试沟通", Summary: "同步情况", RiskLevel: "low"})
	if err != nil {
		t.Fatalf("create record: %v", err)
	}
	if record.ID == "" {
		t.Fatalf("expected record id")
	}
	updatedParent, err := store.UpdateParent(ctx, testUserID, parents[0].ID, domain.ParentProfile{
		StudentName:        parents[0].StudentName,
		ClassName:          parents[0].ClassName,
		ParentName:         parents[0].ParentName,
		Relationship:       parents[0].Relationship,
		Contact:            "13800000000",
		CommunicationStyle: "需要先共情",
		RiskLevel:          "high",
		ImportantNotes:     "更新后的重点信息",
		NextAction:         "明天电话跟进",
	})
	if err != nil {
		t.Fatalf("update parent: %v", err)
	}
	if updatedParent.RiskLevel != "high" || updatedParent.Contact == "" {
		t.Fatalf("unexpected updated parent: %+v", updatedParent)
	}
	updatedRecord, err := store.UpdateCommunicationRecord(ctx, testUserID, record.ID, domain.CommunicationRecord{
		ParentID:  parents[0].ID,
		Student:   parents[0].StudentName,
		Channel:   "电话",
		Reason:    "补充沟通",
		Summary:   "已补充沟通背景",
		Result:    "约定明天再确认",
		RiskLevel: "medium",
	})
	if err != nil {
		t.Fatalf("update record: %v", err)
	}
	if updatedRecord.Channel != "电话" {
		t.Fatalf("unexpected updated record: %+v", updatedRecord)
	}
	if err := store.DeleteCommunicationRecord(ctx, testUserID, record.ID); err != nil {
		t.Fatalf("delete record: %v", err)
	}
	if _, err := store.CommunicationRecord(ctx, testUserID, record.ID); err == nil {
		t.Fatalf("expected deleted record to be gone")
	}
}

func TestMemoryStoreCoursesCRUD(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	course, err := store.CreateCourse(ctx, testUserID, domain.Course{Title: "班会", ClassName: "高二(1)班", Location: "301", Weekday: 2, StartTime: "08:00", EndTime: "08:45", Note: "主题班会"})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	if course.ID == "" {
		t.Fatalf("expected course id")
	}
	updated, err := store.UpdateCourse(ctx, testUserID, course.ID, domain.Course{Title: "主题班会", ClassName: "高二(1)班", Location: "301", Weekday: 2, StartTime: "08:10", EndTime: "08:55", Note: "更新内容"})
	if err != nil {
		t.Fatalf("update course: %v", err)
	}
	if updated.Title != "主题班会" {
		t.Fatalf("unexpected updated course: %+v", updated)
	}
	if err := store.DeleteCourse(ctx, testUserID, course.ID); err != nil {
		t.Fatalf("delete course: %v", err)
	}
	if _, err := store.Course(ctx, testUserID, course.ID); err == nil {
		t.Fatalf("expected deleted course to be gone")
	}
}

func TestMemoryStoreProfileAndFavorites(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	profile, err := store.UserProfile(ctx, testUserID)
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	profile.ReminderPolicy = "normal"
	updated, err := store.UpdateUserProfile(ctx, testUserID, profile)
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.ReminderPolicy != "normal" {
		t.Fatalf("unexpected profile: %+v", updated)
	}

	favorite, err := store.CreateFavorite(ctx, testUserID, domain.Favorite{Type: "communication_template", Title: "测试收藏", Content: "测试内容"})
	if err != nil {
		t.Fatalf("create favorite: %v", err)
	}
	items, err := store.Favorites(ctx, testUserID, "communication_template")
	if err != nil {
		t.Fatalf("favorites: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected favorites")
	}
	if err := store.DeleteFavorite(ctx, testUserID, favorite.ID); err != nil {
		t.Fatalf("delete favorite: %v", err)
	}
}
