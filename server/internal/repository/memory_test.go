package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

const testUserID = domain.ID("00000000-0000-0000-0000-000000000001")

func TestMemoryStoreRejectsEmptyUserID(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.UserProfile(context.Background(), "")
	if !errors.Is(err, ErrMissingUserID) {
		t.Fatalf("expected ErrMissingUserID, got %v", err)
	}
}

func TestMemoryStoreAuthSessionsCanBeRevoked(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	session, err := store.CreateAuthSession(ctx, domain.AuthSession{
		UserID:    testUserID,
		TokenHash: "token-hash-1",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("create auth session: %v", err)
	}
	if session.ID == "" || session.CreatedAt.IsZero() {
		t.Fatalf("unexpected auth session: %+v", session)
	}

	found, err := store.AuthSessionByTokenHash(ctx, "token-hash-1")
	if err != nil {
		t.Fatalf("auth session lookup: %v", err)
	}
	if found.UserID != testUserID {
		t.Fatalf("unexpected auth session owner: %+v", found)
	}
	if err := store.RevokeAuthSession(ctx, "token-hash-1", testUserID); err != nil {
		t.Fatalf("revoke auth session: %v", err)
	}
	if _, err := store.AuthSessionByTokenHash(ctx, "token-hash-1"); err == nil {
		t.Fatalf("expected revoked auth session to be unavailable")
	}
}

func TestMemoryStoreDeleteUserRemovesOwnedDataAndWechatMapping(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	profile, err := store.FindOrCreateUserByWechatOpenID(ctx, "openid-delete-me", "待删除老师", "")
	if err != nil {
		t.Fatalf("create user by openid: %v", err)
	}
	if _, err := store.CreateCourse(ctx, profile.ID, domain.Course{Title: "待删除课程", ClassName: "一班", Weekday: 1, StartTime: "09:00", EndTime: "09:45"}); err != nil {
		t.Fatalf("create owned course: %v", err)
	}
	if _, err := store.CreateAuthSession(ctx, domain.AuthSession{UserID: profile.ID, TokenHash: "delete-user-token", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("create owned auth session: %v", err)
	}

	if err := store.DeleteUser(ctx, profile.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	if _, err := store.UserProfile(ctx, profile.ID); err == nil {
		t.Fatalf("expected deleted user profile to be gone")
	}
	if _, err := store.AuthSessionByTokenHash(ctx, "delete-user-token"); err == nil {
		t.Fatalf("expected deleted user session to be gone")
	}

	recreated, err := store.FindOrCreateUserByWechatOpenID(ctx, "openid-delete-me", "重新登录老师", "")
	if err != nil {
		t.Fatalf("recreate user by openid: %v", err)
	}
	if recreated.ID == profile.ID {
		t.Fatalf("expected deleted openid mapping to be recreated with a new user id")
	}
	courses, err := store.CoursesByDay(ctx, recreated.ID, 1)
	if err != nil {
		t.Fatalf("list recreated user courses: %v", err)
	}
	if len(courses) != 0 {
		t.Fatalf("expected recreated account to start without deleted data, got %+v", courses)
	}
}

func TestMemoryStoreExportUserDataIncludesOwnedDataAndAIActions(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	profile, err := store.FindOrCreateUserByWechatOpenID(ctx, "openid-export-me", "导出老师", "")
	if err != nil {
		t.Fatalf("create user by openid: %v", err)
	}
	course, err := store.CreateCourse(ctx, profile.ID, domain.Course{Title: "导出课程", ClassName: "一班", Weekday: 1, StartTime: "09:00", EndTime: "09:45"})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	parent, err := store.CreateParent(ctx, profile.ID, domain.ParentProfile{
		StudentName:  "导出学生",
		ClassName:    "一班",
		ParentName:   "导出家长",
		Relationship: "母亲",
		RiskLevel:    "medium",
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	reminder, err := store.CreateReminder(ctx, profile.ID, domain.Reminder{
		Title:    "导出提醒",
		Category: "回访",
		RemindAt: time.Now().Add(time.Hour),
		ParentID: &parent.ID,
		CourseID: &course.ID,
	})
	if err != nil {
		t.Fatalf("create reminder: %v", err)
	}
	if err := store.DeleteReminder(ctx, profile.ID, reminder.ID); err != nil {
		t.Fatalf("soft delete reminder: %v", err)
	}
	if _, err := store.CreateCommunicationRecord(ctx, profile.ID, domain.CommunicationRecord{
		ParentID:  &parent.ID,
		Student:   parent.StudentName,
		Channel:   "微信",
		Reason:    "导出沟通",
		Summary:   "导出摘要",
		RiskLevel: "medium",
	}); err != nil {
		t.Fatalf("create record: %v", err)
	}
	if _, err := store.CreateHealingEntry(ctx, profile.ID, domain.HealingEntry{Type: "praise", Mood: "warm", Content: "导出疗愈", AIReply: "导出回复"}); err != nil {
		t.Fatalf("create healing: %v", err)
	}
	generation, err := store.CreateAIGeneration(ctx, profile.ID, domain.AIGeneration{
		Scenario:    "praise",
		Input:       map[string]any{"content": "导出 AI 输入"},
		Output:      map[string]any{"draft": map[string]any{"content": "导出 AI 输出"}},
		SafetyLabel: "self_care",
	})
	if err != nil {
		t.Fatalf("create ai generation: %v", err)
	}
	if _, err := store.CreateAIAction(ctx, profile.ID, domain.AIAction{
		GenerationID: generation.ID,
		Action:       "review_confirmed",
		DraftID:      "draft-1",
		Note:         "导出复核",
	}); err != nil {
		t.Fatalf("create ai action: %v", err)
	}
	if _, err := store.CreateFavorite(ctx, profile.ID, domain.Favorite{Type: "ai_praise", Title: "导出收藏", Content: "导出收藏内容"}); err != nil {
		t.Fatalf("create favorite: %v", err)
	}

	data, err := store.ExportUserData(ctx, profile.ID)
	if err != nil {
		t.Fatalf("export user data: %v", err)
	}
	if data.Profile.ID != profile.ID || data.Profile.Name != "导出老师" || data.ExportedAt.IsZero() {
		t.Fatalf("unexpected export profile metadata: %+v", data)
	}
	if len(data.Courses) != 1 || data.Courses[0].Title != "导出课程" {
		t.Fatalf("expected exported course, got %+v", data.Courses)
	}
	if len(data.Reminders) != 1 || data.Reminders[0].Status != "deleted" {
		t.Fatalf("expected exported soft-deleted reminder, got %+v", data.Reminders)
	}
	if len(data.Parents) != 1 || len(data.CommunicationRecords) != 1 || len(data.HealingEntries) != 1 || len(data.Favorites) != 1 {
		t.Fatalf("expected exported owned data, got %+v", data)
	}
	if len(data.AIGenerations) != 1 || len(data.AIGenerations[0].Actions) != 1 || data.AIGenerations[0].Actions[0].Action != "review_confirmed" {
		t.Fatalf("expected exported ai generation actions, got %+v", data.AIGenerations)
	}

	if _, err := store.ExportUserData(ctx, "missing-user"); err == nil {
		t.Fatalf("expected missing user export to fail")
	}
}

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
	if summary.FollowUpsCount != 0 || len(summary.FocusParents) != 0 {
		t.Fatalf("expected no due seeded follow-ups today, got %+v", summary)
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
	summary, err = store.Dashboard(ctx, testUserID, time.Now())
	if err != nil {
		t.Fatalf("dashboard after reminder completion: %v", err)
	}
	foundCompleted := false
	for _, item := range summary.Reminders {
		if item.ID == reminder.ID {
			foundCompleted = true
			if item.Status != "done" {
				t.Fatalf("expected completed reminder to remain visible as done, got %+v", item)
			}
		}
	}
	if !foundCompleted {
		t.Fatalf("expected completed reminder to remain in dashboard list, got %+v", summary.Reminders)
	}
	if summary.RemindersCount != 2 {
		t.Fatalf("expected completed reminder to be excluded from pending count, got %+v", summary)
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

func TestMemoryStoreDashboardCountsDueFollowUpsOnly(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	parents, err := store.Parents(ctx, testUserID, ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("parents: %v", err)
	}
	if len(parents) < 2 {
		t.Fatalf("expected seeded parents")
	}
	targetDay := time.Now().AddDate(0, 0, 1)

	dueRecord, err := store.CreateCommunicationRecord(ctx, testUserID, domain.CommunicationRecord{
		ParentID:   &parents[0].ID,
		Student:    parents[0].StudentName,
		Channel:    "微信",
		Reason:     "到期跟进",
		Summary:    "需要今天跟进",
		RiskLevel:  "medium",
		FollowUpAt: targetDay.Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("create due record: %v", err)
	}
	_, err = store.CreateCommunicationRecord(ctx, testUserID, domain.CommunicationRecord{
		ParentID:   &parents[1].ID,
		Student:    parents[1].StudentName,
		Channel:    "微信",
		Reason:     "未来跟进",
		Summary:    "未来再看",
		RiskLevel:  "low",
		FollowUpAt: targetDay.AddDate(0, 0, 1),
	})
	if err != nil {
		t.Fatalf("create future record: %v", err)
	}

	summary, err := store.Dashboard(ctx, testUserID, targetDay)
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if summary.FollowUpsCount != 1 {
		t.Fatalf("expected only the explicit due record, got %+v", summary)
	}
	if len(summary.FocusParents) != 1 || summary.FocusParents[0].ID != parents[0].ID {
		t.Fatalf("expected only the due parent, got %+v", summary.FocusParents)
	}

	completed, err := store.CompleteCommunicationFollowUp(ctx, testUserID, dueRecord.ID)
	if err != nil {
		t.Fatalf("complete communication follow-up: %v", err)
	}
	if completed.FollowUpStatus != "done" || completed.FollowedUpAt == nil {
		t.Fatalf("expected completed follow-up metadata, got %+v", completed)
	}
	summary, err = store.Dashboard(ctx, testUserID, targetDay)
	if err != nil {
		t.Fatalf("dashboard after follow-up completion: %v", err)
	}
	if summary.FollowUpsCount != 0 || len(summary.FocusParents) != 0 {
		t.Fatalf("expected completed follow-up to disappear from dashboard, got %+v", summary)
	}
}

func TestNextCourseForDaySkipsFinishedCourses(t *testing.T) {
	today := time.Now()
	currentMinute := today.Format("15:04")
	courses := []domain.Course{
		{ID: "finished", Title: "刚结束的课", StartTime: "00:00", EndTime: currentMinute},
		{ID: "late", Title: "晚课", StartTime: "23:30", EndTime: "23:59"},
	}

	next := nextCourseForDay(courses, today)
	if next == nil || next.ID != "late" {
		t.Fatalf("expected next unfinished course, got %+v", next)
	}

	pastOnly := []domain.Course{{ID: "past", Title: "已结束", StartTime: "00:00", EndTime: "00:01"}}
	if next := nextCourseForDay(pastOnly, today); next != nil {
		t.Fatalf("expected no next course after all courses finished, got %+v", next)
	}

	tomorrow := today.AddDate(0, 0, 1)
	futureDay := []domain.Course{{ID: "early", Title: "明天早课", StartTime: "00:00", EndTime: "00:01"}}
	next = nextCourseForDay(futureDay, tomorrow)
	if next == nil || next.ID != "early" {
		t.Fatalf("expected first course for future day, got %+v", next)
	}
}

func TestMemoryStoreParentsAndRecords(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	parents, err := store.Parents(ctx, testUserID, ListOptions{})
	if err != nil {
		t.Fatalf("parents: %v", err)
	}
	if len(parents) < 2 {
		t.Fatalf("expected seeded parents")
	}
	record, err := store.CreateCommunicationRecord(ctx, testUserID, domain.CommunicationRecord{ParentID: &parents[0].ID, Student: parents[0].StudentName, Channel: "微信", Reason: "测试沟通", Summary: "同步情况", RiskLevel: "low"})
	if err != nil {
		t.Fatalf("create record: %v", err)
	}
	if record.ID == "" {
		t.Fatalf("expected record id")
	}
	if record.FollowUpStatus != "pending" {
		t.Fatalf("expected new record to default to pending follow-up, got %+v", record)
	}
	completed, err := store.CompleteCommunicationFollowUp(ctx, testUserID, record.ID)
	if err != nil {
		t.Fatalf("complete record follow-up: %v", err)
	}
	if completed.FollowUpStatus != "done" || completed.FollowedUpAt == nil {
		t.Fatalf("unexpected completed record: %+v", completed)
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
		ParentID:  &parents[0].ID,
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

func TestMemoryStoreDeleteParentKeepsHistoricalCommunicationRecords(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	parent, err := store.CreateParent(ctx, testUserID, domain.ParentProfile{
		StudentName:  "历史学生",
		ClassName:    "一班",
		ParentName:   "历史家长",
		Relationship: "母亲",
		RiskLevel:    "low",
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	record, err := store.CreateCommunicationRecord(ctx, testUserID, domain.CommunicationRecord{
		ParentID:  &parent.ID,
		Student:   parent.StudentName,
		Channel:   "微信",
		Reason:    "历史沟通",
		Summary:   "删除家长后也应保留",
		RiskLevel: "low",
	})
	if err != nil {
		t.Fatalf("create record: %v", err)
	}

	if err := store.DeleteParent(ctx, testUserID, parent.ID); err != nil {
		t.Fatalf("delete parent: %v", err)
	}
	found, err := store.CommunicationRecord(ctx, testUserID, record.ID)
	if err != nil {
		t.Fatalf("communication record should remain after parent delete: %v", err)
	}
	if found.ParentID != nil {
		t.Fatalf("expected record parent to be cleared after parent delete, got %+v", found.ParentID)
	}
	allRecords, err := store.CommunicationRecords(ctx, testUserID, nil, ListOptions{Limit: 100})
	if err != nil {
		t.Fatalf("list all records: %v", err)
	}
	if !containsCommunicationRecord(allRecords, record.ID) {
		t.Fatalf("expected unlinked historical record in all records, got %+v", allRecords)
	}
	filteredRecords, err := store.CommunicationRecords(ctx, testUserID, &parent.ID, ListOptions{Limit: 100})
	if err != nil {
		t.Fatalf("list records by deleted parent: %v", err)
	}
	if containsCommunicationRecord(filteredRecords, record.ID) {
		t.Fatalf("expected deleted parent filter to exclude unlinked record, got %+v", filteredRecords)
	}
}

func TestMemoryStoreListOptionsFilterAndPaginate(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	filteredParents, err := store.Parents(ctx, testUserID, ListOptions{Query: "陈子默", Limit: 10})
	if err != nil {
		t.Fatalf("filtered parents: %v", err)
	}
	if len(filteredParents) != 1 || filteredParents[0].StudentName != "陈子默" {
		t.Fatalf("unexpected filtered parents: %+v", filteredParents)
	}

	pagedParents, err := store.Parents(ctx, testUserID, ListOptions{Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("paged parents: %v", err)
	}
	if len(pagedParents) != 1 {
		t.Fatalf("unexpected paged parents: %+v", pagedParents)
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
	items, err := store.Favorites(ctx, testUserID, "communication_template", ListOptions{Query: "测试", Limit: 10})
	if err != nil {
		t.Fatalf("favorites: %v", err)
	}
	if len(items) != 1 || items[0].Title != "测试收藏" {
		t.Fatalf("expected searched favorite, got %+v", items)
	}
	paged, err := store.Favorites(ctx, testUserID, "", ListOptions{Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("paged favorites: %v", err)
	}
	if len(paged) != 1 {
		t.Fatalf("expected paged favorites, got %+v", paged)
	}
	if err := store.DeleteFavorite(ctx, testUserID, favorite.ID); err != nil {
		t.Fatalf("delete favorite: %v", err)
	}
}

func TestMemoryStoreHealingEntries(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	entry, err := store.CreateHealingEntry(ctx, testUserID, domain.HealingEntry{
		Type:    "praise",
		Mood:    "warm",
		Content: "今天完成了很多工作",
		AIReply: "你已经稳稳推进了很多复杂事项。",
	})
	if err != nil {
		t.Fatalf("create healing entry: %v", err)
	}
	if entry.ID == "" || entry.CreatedAt.IsZero() {
		t.Fatalf("unexpected healing entry: %+v", entry)
	}

	items, err := store.HealingEntries(ctx, testUserID, "praise", ListOptions{})
	if err != nil {
		t.Fatalf("healing entries: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected healing entries")
	}

	found, err := store.HealingEntry(ctx, testUserID, entry.ID)
	if err != nil {
		t.Fatalf("healing entry detail: %v", err)
	}
	if found.AIReply != entry.AIReply {
		t.Fatalf("unexpected healing entry detail: %+v", found)
	}

	if err := store.DeleteHealingEntry(ctx, testUserID, entry.ID); err != nil {
		t.Fatalf("delete healing entry: %v", err)
	}
	if _, err := store.HealingEntry(ctx, testUserID, entry.ID); err == nil {
		t.Fatalf("expected deleted healing entry to be gone")
	}
}

func TestMemoryStoreAIGenerations(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	generation, err := store.CreateAIGeneration(ctx, testUserID, domain.AIGeneration{
		Scenario:    "parent_drafts",
		Input:       map[string]any{"issue": "孩子连续三天没交作业"},
		Output:      map[string]any{"drafts": []any{map[string]any{"content": "先同步观察，再给出建议。"}}},
		SafetyLabel: "teacher_review_required",
	})
	if err != nil {
		t.Fatalf("create ai generation: %v", err)
	}
	if generation.ID == "" || generation.CreatedAt.IsZero() {
		t.Fatalf("unexpected ai generation: %+v", generation)
	}

	items, err := store.AIGenerations(ctx, testUserID, "parent_drafts", ListOptions{})
	if err != nil {
		t.Fatalf("ai generations: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected ai generations")
	}

	found, err := store.AIGeneration(ctx, testUserID, generation.ID)
	if err != nil {
		t.Fatalf("ai generation detail: %v", err)
	}
	if found.Scenario != "parent_drafts" || found.SafetyLabel != "teacher_review_required" {
		t.Fatalf("unexpected ai generation detail: %+v", found)
	}
	if err := store.UpdateAIGenerationOutput(ctx, testUserID, generation.ID, map[string]any{"drafts": []any{map[string]any{"generationId": string(generation.ID), "content": "已回填 generation id"}}}); err != nil {
		t.Fatalf("update ai generation output: %v", err)
	}
	found, err = store.AIGeneration(ctx, testUserID, generation.ID)
	if err != nil {
		t.Fatalf("ai generation detail after output update: %v", err)
	}
	drafts, ok := found.Output["drafts"].([]any)
	if !ok || len(drafts) != 1 {
		t.Fatalf("expected updated output drafts, got %+v", found.Output)
	}
	draft, ok := drafts[0].(map[string]any)
	if !ok || draft["generationId"] != string(generation.ID) {
		t.Fatalf("expected updated output generation id, got %+v", drafts[0])
	}

	action, err := store.CreateAIAction(ctx, testUserID, domain.AIAction{
		GenerationID: generation.ID,
		Action:       "copy",
		DraftID:      "draft-1",
		Note:         "复制前已复核",
		Metadata:     map[string]any{"surface": "test"},
	})
	if err != nil {
		t.Fatalf("create ai action: %v", err)
	}
	if action.ID == "" || action.CreatedAt.IsZero() || action.Metadata["surface"] != "test" {
		t.Fatalf("unexpected ai action: %+v", action)
	}

	found, err = store.AIGeneration(ctx, testUserID, generation.ID)
	if err != nil {
		t.Fatalf("ai generation detail with actions: %v", err)
	}
	if len(found.Actions) != 1 || found.Actions[0].Action != "copy" {
		t.Fatalf("expected ai action on generation detail, got %+v", found)
	}

	if _, err := store.CreateAIAction(ctx, testUserID, domain.AIAction{GenerationID: "missing", Action: "copy"}); err == nil {
		t.Fatalf("expected missing generation action to fail")
	}

	if err := store.DeleteAIGeneration(ctx, testUserID, generation.ID); err != nil {
		t.Fatalf("delete ai generation: %v", err)
	}
	if _, err := store.AIGeneration(ctx, testUserID, generation.ID); err == nil {
		t.Fatalf("expected deleted ai generation to be gone")
	}
	if _, err := store.CreateAIAction(ctx, testUserID, domain.AIAction{GenerationID: generation.ID, Action: "copy"}); err == nil {
		t.Fatalf("expected actions for deleted generation to fail")
	}
}

func containsCommunicationRecord(records []domain.CommunicationRecord, id domain.ID) bool {
	for _, record := range records {
		if record.ID == id {
			return true
		}
	}
	return false
}
