package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) UserProfile(ctx context.Context, userID domain.ID) (domain.UserProfile, error) {
	var profile domain.UserProfile
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, name, COALESCE(school, ''), COALESCE(stage, ''), COALESCE(subject, ''), is_head_teacher, pro_status, reminder_policy, created_at
		FROM users
		WHERE id = $1`, string(userID)).
		Scan(&profile.ID, &profile.Name, &profile.School, &profile.Stage, &profile.Subject, &profile.IsHeadTeacher, &profile.ProStatus, &profile.ReminderPolicy, &profile.CreatedAt)
	if isNoRows(err) {
		return domain.UserProfile{}, notFound("user profile", userID)
	}
	return profile, err
}

func (s *PostgresStore) UpdateUserProfile(ctx context.Context, userID domain.ID, profile domain.UserProfile) (domain.UserProfile, error) {
	current, err := s.UserProfile(ctx, userID)
	if err != nil {
		return domain.UserProfile{}, err
	}
	if profile.Name == "" {
		profile.Name = current.Name
	}
	if profile.School == "" {
		profile.School = current.School
	}
	if profile.Stage == "" {
		profile.Stage = current.Stage
	}
	if profile.Subject == "" {
		profile.Subject = current.Subject
	}
	if profile.ProStatus == "" {
		profile.ProStatus = current.ProStatus
	}
	if profile.ReminderPolicy == "" {
		profile.ReminderPolicy = current.ReminderPolicy
	}
	err = s.pool.QueryRow(ctx, `
		UPDATE users
		SET name = $2, school = $3, stage = $4, subject = $5, is_head_teacher = $6, pro_status = $7, reminder_policy = $8, updated_at = now()
		WHERE id = $1
		RETURNING id::text, name, COALESCE(school, ''), COALESCE(stage, ''), COALESCE(subject, ''), is_head_teacher, pro_status, reminder_policy, created_at`,
		string(userID), profile.Name, profile.School, profile.Stage, profile.Subject, profile.IsHeadTeacher, profile.ProStatus, profile.ReminderPolicy).
		Scan(&profile.ID, &profile.Name, &profile.School, &profile.Stage, &profile.Subject, &profile.IsHeadTeacher, &profile.ProStatus, &profile.ReminderPolicy, &profile.CreatedAt)
	if isNoRows(err) {
		return domain.UserProfile{}, notFound("user profile", userID)
	}
	return profile, err
}

func (s *PostgresStore) Dashboard(ctx context.Context, userID domain.ID, day time.Time) (domain.DashboardSummary, error) {
	weekday := int(day.Weekday())
	courses, err := s.CoursesByDay(ctx, userID, weekday)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	reminders, err := s.RemindersByDay(ctx, userID, day)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	parents, err := s.Parents(ctx, userID)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
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

func (s *PostgresStore) CoursesByDay(ctx context.Context, userID domain.ID, weekday int) ([]domain.Course, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, title, class_name, COALESCE(location, ''), weekday, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'), COALESCE(note, ''), created_at
		FROM courses
		WHERE user_id = $1 AND weekday = $2
		ORDER BY start_time ASC`, string(userID), weekday)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Course, 0)
	for rows.Next() {
		var item domain.Course
		if err := rows.Scan(&item.ID, &item.Title, &item.ClassName, &item.Location, &item.Weekday, &item.StartTime, &item.EndTime, &item.Note, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) Course(ctx context.Context, userID domain.ID, id domain.ID) (domain.Course, error) {
	var item domain.Course
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, title, class_name, COALESCE(location, ''), weekday, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'), COALESCE(note, ''), created_at
		FROM courses
		WHERE user_id = $1 AND id = $2`, string(userID), string(id)).
		Scan(&item.ID, &item.Title, &item.ClassName, &item.Location, &item.Weekday, &item.StartTime, &item.EndTime, &item.Note, &item.CreatedAt)
	if isNoRows(err) {
		return domain.Course{}, notFound("course", id)
	}
	return item, err
}

func (s *PostgresStore) CreateCourse(ctx context.Context, userID domain.ID, course domain.Course) (domain.Course, error) {
	if course.Weekday < 0 || course.Weekday > 6 {
		course.Weekday = int(time.Now().Weekday())
	}
	if course.StartTime == "" {
		course.StartTime = "08:00"
	}
	if course.EndTime == "" {
		course.EndTime = "08:45"
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO courses (user_id, title, class_name, location, weekday, start_time, end_time, note)
		VALUES ($1, $2, $3, $4, $5, $6::time, $7::time, $8)
		RETURNING id::text, created_at`, string(userID), course.Title, course.ClassName, course.Location, course.Weekday, course.StartTime, course.EndTime, course.Note).
		Scan(&course.ID, &course.CreatedAt)
	return course, err
}

func (s *PostgresStore) UpdateCourse(ctx context.Context, userID domain.ID, id domain.ID, course domain.Course) (domain.Course, error) {
	if course.StartTime == "" {
		course.StartTime = "08:00"
	}
	if course.EndTime == "" {
		course.EndTime = "08:45"
	}
	err := s.pool.QueryRow(ctx, `
		UPDATE courses
		SET title = $3, class_name = $4, location = $5, weekday = $6, start_time = $7::time, end_time = $8::time, note = $9, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, title, class_name, COALESCE(location, ''), weekday, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'), COALESCE(note, ''), created_at`,
		string(userID), string(id), course.Title, course.ClassName, course.Location, course.Weekday, course.StartTime, course.EndTime, course.Note).
		Scan(&course.ID, &course.Title, &course.ClassName, &course.Location, &course.Weekday, &course.StartTime, &course.EndTime, &course.Note, &course.CreatedAt)
	if isNoRows(err) {
		return domain.Course{}, notFound("course", id)
	}
	return course, err
}

func (s *PostgresStore) DeleteCourse(ctx context.Context, userID domain.ID, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM courses WHERE user_id = $1 AND id = $2`, string(userID), string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("course", id)
	}
	return nil
}

func (s *PostgresStore) RemindersByDay(ctx context.Context, userID domain.ID, day time.Time) ([]domain.Reminder, error) {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := start.Add(24 * time.Hour)
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, title, category, remind_at, status, COALESCE(note, ''), parent_id::text, course_id::text, created_at, done_at
		FROM reminders
		WHERE user_id = $1 AND remind_at >= $2 AND remind_at < $3 AND status <> 'deleted'
		ORDER BY remind_at ASC`, string(userID), start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Reminder, 0)
	for rows.Next() {
		var item domain.Reminder
		var parentID, courseID sql.NullString
		var doneAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Title, &item.Category, &item.RemindAt, &item.Status, &item.Note, &parentID, &courseID, &item.CreatedAt, &doneAt); err != nil {
			return nil, err
		}
		item.ParentID = nullableDomainID(parentID)
		item.CourseID = nullableDomainID(courseID)
		if doneAt.Valid {
			item.DoneAt = &doneAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) Reminder(ctx context.Context, userID domain.ID, id domain.ID) (domain.Reminder, error) {
	var item domain.Reminder
	var parentID, courseID sql.NullString
	var doneAt sql.NullTime
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, title, category, remind_at, status, COALESCE(note, ''), parent_id::text, course_id::text, created_at, done_at
		FROM reminders
		WHERE user_id = $1 AND id = $2 AND status <> 'deleted'`, string(userID), string(id)).
		Scan(&item.ID, &item.Title, &item.Category, &item.RemindAt, &item.Status, &item.Note, &parentID, &courseID, &item.CreatedAt, &doneAt)
	if isNoRows(err) {
		return domain.Reminder{}, notFound("reminder", id)
	}
	item.ParentID = nullableDomainID(parentID)
	item.CourseID = nullableDomainID(courseID)
	if doneAt.Valid {
		item.DoneAt = &doneAt.Time
	}
	return item, err
}

func (s *PostgresStore) CreateReminder(ctx context.Context, userID domain.ID, reminder domain.Reminder) (domain.Reminder, error) {
	if reminder.Status == "" {
		reminder.Status = "pending"
	}
	if reminder.Category == "" {
		reminder.Category = "personal"
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO reminders (user_id, parent_id, course_id, title, category, remind_at, status, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, created_at`, string(userID), fromDomainID(reminder.ParentID), fromDomainID(reminder.CourseID), reminder.Title, reminder.Category, reminder.RemindAt, reminder.Status, reminder.Note).
		Scan(&reminder.ID, &reminder.CreatedAt)
	return reminder, err
}

func (s *PostgresStore) UpdateReminder(ctx context.Context, userID domain.ID, id domain.ID, reminder domain.Reminder) (domain.Reminder, error) {
	if reminder.Status == "" {
		reminder.Status = "pending"
	}
	if reminder.Category == "" {
		reminder.Category = "personal"
	}
	var parentID, courseID sql.NullString
	var doneAt sql.NullTime
	err := s.pool.QueryRow(ctx, `
		UPDATE reminders
		SET parent_id = $3, course_id = $4, title = $5, category = $6, remind_at = $7, status = $8, note = $9, done_at = $10, updated_at = now()
		WHERE user_id = $1 AND id = $2 AND status <> 'deleted'
		RETURNING id::text, title, category, remind_at, status, COALESCE(note, ''), parent_id::text, course_id::text, created_at, done_at`,
		string(userID), string(id), fromDomainID(reminder.ParentID), fromDomainID(reminder.CourseID), reminder.Title, reminder.Category, reminder.RemindAt, reminder.Status, reminder.Note, fromTimePtr(reminder.DoneAt)).
		Scan(&reminder.ID, &reminder.Title, &reminder.Category, &reminder.RemindAt, &reminder.Status, &reminder.Note, &parentID, &courseID, &reminder.CreatedAt, &doneAt)
	if isNoRows(err) {
		return domain.Reminder{}, notFound("reminder", id)
	}
	reminder.ParentID = nullableDomainID(parentID)
	reminder.CourseID = nullableDomainID(courseID)
	if doneAt.Valid {
		reminder.DoneAt = &doneAt.Time
	} else {
		reminder.DoneAt = nil
	}
	return reminder, err
}

func (s *PostgresStore) CompleteReminder(ctx context.Context, userID domain.ID, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `UPDATE reminders SET status = 'done', done_at = now(), updated_at = now() WHERE user_id = $1 AND id = $2 AND status <> 'deleted'`, string(userID), string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("reminder", id)
	}
	return nil
}

func (s *PostgresStore) DeleteReminder(ctx context.Context, userID domain.ID, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `UPDATE reminders SET status = 'deleted', updated_at = now() WHERE user_id = $1 AND id = $2 AND status <> 'deleted'`, string(userID), string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("reminder", id)
	}
	return nil
}

func (s *PostgresStore) SnoozeReminder(ctx context.Context, userID domain.ID, id domain.ID, until time.Time) (domain.Reminder, error) {
	var item domain.Reminder
	var parentID, courseID sql.NullString
	var doneAt sql.NullTime
	err := s.pool.QueryRow(ctx, `
		UPDATE reminders
		SET status = 'snoozed', remind_at = $3, done_at = NULL, updated_at = now()
		WHERE user_id = $1 AND id = $2 AND status <> 'deleted'
		RETURNING id::text, title, category, remind_at, status, COALESCE(note, ''), parent_id::text, course_id::text, created_at, done_at`,
		string(userID), string(id), until).
		Scan(&item.ID, &item.Title, &item.Category, &item.RemindAt, &item.Status, &item.Note, &parentID, &courseID, &item.CreatedAt, &doneAt)
	if isNoRows(err) {
		return domain.Reminder{}, notFound("reminder", id)
	}
	item.ParentID = nullableDomainID(parentID)
	item.CourseID = nullableDomainID(courseID)
	if doneAt.Valid {
		item.DoneAt = &doneAt.Time
	}
	return item, err
}

func (s *PostgresStore) Parents(ctx context.Context, userID domain.ID) ([]domain.ParentProfile, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, student_name, class_name, parent_name, relationship, COALESCE(contact, ''), COALESCE(communication_style, ''), risk_level, COALESCE(important_notes, ''), COALESCE(next_action, ''), created_at
		FROM parent_profiles
		WHERE user_id = $1
		ORDER BY CASE risk_level WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END, updated_at DESC`, string(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.ParentProfile, 0)
	for rows.Next() {
		var item domain.ParentProfile
		if err := rows.Scan(&item.ID, &item.StudentName, &item.ClassName, &item.ParentName, &item.Relationship, &item.Contact, &item.CommunicationStyle, &item.RiskLevel, &item.ImportantNotes, &item.NextAction, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) Parent(ctx context.Context, userID domain.ID, id domain.ID) (domain.ParentProfile, error) {
	var item domain.ParentProfile
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, student_name, class_name, parent_name, relationship, COALESCE(contact, ''), COALESCE(communication_style, ''), risk_level, COALESCE(important_notes, ''), COALESCE(next_action, ''), created_at
		FROM parent_profiles
		WHERE user_id = $1 AND id = $2`, string(userID), string(id)).
		Scan(&item.ID, &item.StudentName, &item.ClassName, &item.ParentName, &item.Relationship, &item.Contact, &item.CommunicationStyle, &item.RiskLevel, &item.ImportantNotes, &item.NextAction, &item.CreatedAt)
	if isNoRows(err) {
		return domain.ParentProfile{}, notFound("parent", id)
	}
	return item, err
}

func (s *PostgresStore) CreateParent(ctx context.Context, userID domain.ID, parent domain.ParentProfile) (domain.ParentProfile, error) {
	if parent.RiskLevel == "" {
		parent.RiskLevel = "low"
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO parent_profiles (user_id, student_name, class_name, parent_name, relationship, contact, communication_style, risk_level, important_notes, next_action)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id::text, created_at`, string(userID), parent.StudentName, parent.ClassName, parent.ParentName, parent.Relationship, parent.Contact, parent.CommunicationStyle, parent.RiskLevel, parent.ImportantNotes, parent.NextAction).
		Scan(&parent.ID, &parent.CreatedAt)
	return parent, err
}

func (s *PostgresStore) UpdateParent(ctx context.Context, userID domain.ID, id domain.ID, parent domain.ParentProfile) (domain.ParentProfile, error) {
	if parent.RiskLevel == "" {
		parent.RiskLevel = "low"
	}
	err := s.pool.QueryRow(ctx, `
		UPDATE parent_profiles
		SET student_name = $3, class_name = $4, parent_name = $5, relationship = $6, contact = $7, communication_style = $8, risk_level = $9, important_notes = $10, next_action = $11, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, student_name, class_name, parent_name, relationship, COALESCE(contact, ''), COALESCE(communication_style, ''), risk_level, COALESCE(important_notes, ''), COALESCE(next_action, ''), created_at`,
		string(userID), string(id), parent.StudentName, parent.ClassName, parent.ParentName, parent.Relationship, parent.Contact, parent.CommunicationStyle, parent.RiskLevel, parent.ImportantNotes, parent.NextAction).
		Scan(&parent.ID, &parent.StudentName, &parent.ClassName, &parent.ParentName, &parent.Relationship, &parent.Contact, &parent.CommunicationStyle, &parent.RiskLevel, &parent.ImportantNotes, &parent.NextAction, &parent.CreatedAt)
	if isNoRows(err) {
		return domain.ParentProfile{}, notFound("parent", id)
	}
	return parent, err
}

func (s *PostgresStore) DeleteParent(ctx context.Context, userID domain.ID, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM parent_profiles WHERE user_id = $1 AND id = $2`, string(userID), string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("parent", id)
	}
	return nil
}

func (s *PostgresStore) CommunicationRecords(ctx context.Context, userID domain.ID, parentID *domain.ID) ([]domain.CommunicationRecord, error) {
	query := `
		SELECT id::text, parent_id::text, student, channel, reason, summary, COALESCE(result, ''), risk_level, follow_up_at, created_at
		FROM communication_records
		WHERE user_id = $1`
	args := []any{string(userID)}
	if parentID != nil {
		query += " AND parent_id = $2"
		args = append(args, string(*parentID))
	}
	query += " ORDER BY created_at DESC LIMIT 50"
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.CommunicationRecord, 0)
	for rows.Next() {
		var item domain.CommunicationRecord
		var parent sql.NullString
		var followUp sql.NullTime
		if err := rows.Scan(&item.ID, &parent, &item.Student, &item.Channel, &item.Reason, &item.Summary, &item.Result, &item.RiskLevel, &followUp, &item.CreatedAt); err != nil {
			return nil, err
		}
		if parent.Valid {
			item.ParentID = domain.ID(parent.String)
		}
		if followUp.Valid {
			item.FollowUpAt = followUp.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID) (domain.CommunicationRecord, error) {
	var item domain.CommunicationRecord
	var parent sql.NullString
	var followUp sql.NullTime
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, parent_id::text, student, channel, reason, summary, COALESCE(result, ''), risk_level, follow_up_at, created_at
		FROM communication_records
		WHERE user_id = $1 AND id = $2`, string(userID), string(id)).
		Scan(&item.ID, &parent, &item.Student, &item.Channel, &item.Reason, &item.Summary, &item.Result, &item.RiskLevel, &followUp, &item.CreatedAt)
	if isNoRows(err) {
		return domain.CommunicationRecord{}, notFound("communication record", id)
	}
	if parent.Valid {
		item.ParentID = domain.ID(parent.String)
	}
	if followUp.Valid {
		item.FollowUpAt = followUp.Time
	}
	return item, err
}

func (s *PostgresStore) CreateCommunicationRecord(ctx context.Context, userID domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error) {
	if record.RiskLevel == "" {
		record.RiskLevel = "low"
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO communication_records (user_id, parent_id, student, channel, reason, summary, result, risk_level, follow_up_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id::text, created_at`, string(userID), nullableRecordParentID(record.ParentID), record.Student, record.Channel, record.Reason, record.Summary, record.Result, record.RiskLevel, nullableTime(record.FollowUpAt)).
		Scan(&record.ID, &record.CreatedAt)
	return record, err
}

func (s *PostgresStore) UpdateCommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error) {
	if record.RiskLevel == "" {
		record.RiskLevel = "low"
	}
	var parent sql.NullString
	var followUp sql.NullTime
	err := s.pool.QueryRow(ctx, `
		UPDATE communication_records
		SET parent_id = $3, student = $4, channel = $5, reason = $6, summary = $7, result = $8, risk_level = $9, follow_up_at = $10, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, parent_id::text, student, channel, reason, summary, COALESCE(result, ''), risk_level, follow_up_at, created_at`,
		string(userID), string(id), nullableRecordParentID(record.ParentID), record.Student, record.Channel, record.Reason, record.Summary, record.Result, record.RiskLevel, nullableTime(record.FollowUpAt)).
		Scan(&record.ID, &parent, &record.Student, &record.Channel, &record.Reason, &record.Summary, &record.Result, &record.RiskLevel, &followUp, &record.CreatedAt)
	if isNoRows(err) {
		return domain.CommunicationRecord{}, notFound("communication record", id)
	}
	if parent.Valid {
		record.ParentID = domain.ID(parent.String)
	}
	if followUp.Valid {
		record.FollowUpAt = followUp.Time
	}
	return record, err
}

func (s *PostgresStore) DeleteCommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM communication_records WHERE user_id = $1 AND id = $2`, string(userID), string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("communication record", id)
	}
	return nil
}

func (s *PostgresStore) HealingEntries(ctx context.Context, userID domain.ID, entryType string) ([]domain.HealingEntry, error) {
	query := `
		SELECT id::text, type, COALESCE(mood, ''), COALESCE(content, ''), COALESCE(ai_reply, ''), created_at
		FROM healing_entries
		WHERE user_id = $1`
	args := []any{string(userID)}
	if entryType != "" {
		query += " AND type = $2"
		args = append(args, entryType)
	}
	query += " ORDER BY created_at DESC LIMIT 50"
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.HealingEntry, 0)
	for rows.Next() {
		var item domain.HealingEntry
		if err := rows.Scan(&item.ID, &item.Type, &item.Mood, &item.Content, &item.AIReply, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) HealingEntry(ctx context.Context, userID domain.ID, id domain.ID) (domain.HealingEntry, error) {
	var item domain.HealingEntry
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, type, COALESCE(mood, ''), COALESCE(content, ''), COALESCE(ai_reply, ''), created_at
		FROM healing_entries
		WHERE user_id = $1 AND id = $2`, string(userID), string(id)).
		Scan(&item.ID, &item.Type, &item.Mood, &item.Content, &item.AIReply, &item.CreatedAt)
	if isNoRows(err) {
		return domain.HealingEntry{}, notFound("healing entry", id)
	}
	return item, err
}

func (s *PostgresStore) CreateHealingEntry(ctx context.Context, userID domain.ID, entry domain.HealingEntry) (domain.HealingEntry, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO healing_entries (user_id, type, mood, content, ai_reply)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, created_at`, string(userID), entry.Type, entry.Mood, entry.Content, entry.AIReply).
		Scan(&entry.ID, &entry.CreatedAt)
	return entry, err
}

func (s *PostgresStore) DeleteHealingEntry(ctx context.Context, userID domain.ID, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM healing_entries WHERE user_id = $1 AND id = $2`, string(userID), string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("healing entry", id)
	}
	return nil
}

func (s *PostgresStore) Favorites(ctx context.Context, userID domain.ID, favoriteType string) ([]domain.Favorite, error) {
	query := `
		SELECT id::text, type, title, content, source_id::text, created_at
		FROM favorites
		WHERE user_id = $1`
	args := []any{string(userID)}
	if favoriteType != "" {
		query += " AND type = $2"
		args = append(args, favoriteType)
	}
	query += " ORDER BY created_at DESC"
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Favorite, 0)
	for rows.Next() {
		var item domain.Favorite
		var sourceID sql.NullString
		if err := rows.Scan(&item.ID, &item.Type, &item.Title, &item.Content, &sourceID, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.SourceID = nullableDomainID(sourceID)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateFavorite(ctx context.Context, userID domain.ID, favorite domain.Favorite) (domain.Favorite, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO favorites (user_id, type, title, content, source_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, created_at`, string(userID), favorite.Type, favorite.Title, favorite.Content, fromDomainID(favorite.SourceID)).
		Scan(&favorite.ID, &favorite.CreatedAt)
	return favorite, err
}

func (s *PostgresStore) DeleteFavorite(ctx context.Context, userID domain.ID, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM favorites WHERE user_id = $1 AND id = $2`, string(userID), string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("favorite", id)
	}
	return nil
}

func nullableDomainID(value sql.NullString) *domain.ID {
	if !value.Valid || value.String == "" {
		return nil
	}
	id := domain.ID(value.String)
	return &id
}

func fromDomainID(value *domain.ID) any {
	if value == nil || *value == "" {
		return nil
	}
	return string(*value)
}

func fromTimePtr(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return *value
}

func nullableRecordParentID(value domain.ID) any {
	if value == "" {
		return nil
	}
	return string(value)
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows)
}

var _ Store = (*PostgresStore)(nil)

