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
}
