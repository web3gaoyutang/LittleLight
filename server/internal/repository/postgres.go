package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
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

func (s *PostgresStore) CompleteReminder(ctx context.Context, userID domain.ID, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `UPDATE reminders SET status = 'done', done_at = now(), updated_at = now() WHERE user_id = $1 AND id = $2`, string(userID), string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("reminder not found: %s", id)
	}
	return nil
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

func (s *PostgresStore) CreateCommunicationRecord(ctx context.Context, userID domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error) {
	if record.RiskLevel == "" {
		record.RiskLevel = "low"
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO communication_records (user_id, parent_id, student, channel, reason, summary, result, risk_level, follow_up_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id::text, created_at`, string(userID), string(record.ParentID), record.Student, record.Channel, record.Reason, record.Summary, record.Result, record.RiskLevel, nullableTime(record.FollowUpAt)).
		Scan(&record.ID, &record.CreatedAt)
	return record, err
}

func (s *PostgresStore) CreateHealingEntry(ctx context.Context, userID domain.ID, entry domain.HealingEntry) (domain.HealingEntry, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO healing_entries (user_id, type, mood, content, ai_reply)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, created_at`, string(userID), entry.Type, entry.Mood, entry.Content, entry.AIReply).
		Scan(&entry.ID, &entry.CreatedAt)
	return entry, err
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

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

var _ Store = (*PostgresStore)(nil)

