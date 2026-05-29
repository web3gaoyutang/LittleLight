package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

func (s *PostgresStore) FindOrCreateUserByWechatOpenID(ctx context.Context, openID string, nickName string, avatarURL string) (domain.UserProfile, error) {
	openID = strings.TrimSpace(openID)
	if openID == "" {
		return domain.UserProfile{}, errors.New("wechat openid is required")
	}
	name := strings.TrimSpace(nickName)
	if name == "" {
		name = "微光老师"
	}
	var profile domain.UserProfile
	err := s.pool.QueryRow(ctx, `
		WITH upserted AS (
			INSERT INTO users (wechat_open_id, name, avatar_url)
			VALUES ($1, $2, $3)
			ON CONFLICT (wechat_open_id) WHERE wechat_open_id IS NOT NULL DO UPDATE
			SET name = CASE WHEN $2 <> '' THEN $2 ELSE users.name END,
			    avatar_url = CASE WHEN $3 <> '' THEN $3 ELSE users.avatar_url END,
			    updated_at = now()
			RETURNING id, name, school, stage, subject, is_head_teacher, pro_status, reminder_policy, created_at
		)
		SELECT id::text, name, COALESCE(school, ''), COALESCE(stage, ''), COALESCE(subject, ''), is_head_teacher, pro_status, reminder_policy, created_at
		FROM upserted`, openID, name, strings.TrimSpace(avatarURL)).
		Scan(&profile.ID, &profile.Name, &profile.School, &profile.Stage, &profile.Subject, &profile.IsHeadTeacher, &profile.ProStatus, &profile.ReminderPolicy, &profile.CreatedAt)
	return profile, err
}

func (s *PostgresStore) CreateAuthSession(ctx context.Context, session domain.AuthSession) (domain.AuthSession, error) {
	session.TokenHash = strings.TrimSpace(session.TokenHash)
	if session.UserID == "" {
		return domain.AuthSession{}, ErrMissingUserID
	}
	if session.TokenHash == "" {
		return domain.AuthSession{}, errors.New("session token hash is required")
	}
	var revokedAt sql.NullTime
	err := s.pool.QueryRow(ctx, `
		INSERT INTO auth_sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id::text, user_id::text, token_hash, expires_at, revoked_at, created_at`,
		string(session.UserID), session.TokenHash, session.ExpiresAt).
		Scan(&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &revokedAt, &session.CreatedAt)
	if revokedAt.Valid {
		session.RevokedAt = &revokedAt.Time
	}
	return session, err
}

func (s *PostgresStore) AuthSessionByTokenHash(ctx context.Context, tokenHash string) (domain.AuthSession, error) {
	tokenHash = strings.TrimSpace(tokenHash)
	if tokenHash == "" {
		return domain.AuthSession{}, notFound("auth session", "")
	}
	var session domain.AuthSession
	var revokedAt sql.NullTime
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, user_id::text, token_hash, expires_at, revoked_at, created_at
		FROM auth_sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()`, tokenHash).
		Scan(&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &revokedAt, &session.CreatedAt)
	if isNoRows(err) {
		return domain.AuthSession{}, notFound("auth session", "")
	}
	if revokedAt.Valid {
		session.RevokedAt = &revokedAt.Time
	}
	return session, err
}

func (s *PostgresStore) RevokeAuthSession(ctx context.Context, tokenHash string, userID domain.ID) error {
	tokenHash = strings.TrimSpace(tokenHash)
	if userID == "" {
		return ErrMissingUserID
	}
	if tokenHash == "" {
		return notFound("auth session", "")
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = COALESCE(revoked_at, now())
		WHERE token_hash = $1 AND user_id = $2`, tokenHash, string(userID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("auth session", "")
	}
	return nil
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

func (s *PostgresStore) DeleteUser(ctx context.Context, userID domain.ID) error {
	if userID == "" {
		return ErrMissingUserID
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, string(userID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("user profile", userID)
	}
	return nil
}

func (s *PostgresStore) ExportUserData(ctx context.Context, userID domain.ID) (domain.AccountExport, error) {
	profile, err := s.UserProfile(ctx, userID)
	if err != nil {
		return domain.AccountExport{}, err
	}
	courses, err := s.allCourses(ctx, userID)
	if err != nil {
		return domain.AccountExport{}, err
	}
	reminders, err := s.allReminders(ctx, userID)
	if err != nil {
		return domain.AccountExport{}, err
	}
	parents, err := s.allParents(ctx, userID)
	if err != nil {
		return domain.AccountExport{}, err
	}
	records, err := s.allCommunicationRecords(ctx, userID)
	if err != nil {
		return domain.AccountExport{}, err
	}
	healing, err := s.allHealingEntries(ctx, userID)
	if err != nil {
		return domain.AccountExport{}, err
	}
	aiLogs, err := s.allAIGenerations(ctx, userID)
	if err != nil {
		return domain.AccountExport{}, err
	}
	for index := range aiLogs {
		actions, err := s.AIActions(ctx, userID, aiLogs[index].ID)
		if err != nil {
			return domain.AccountExport{}, err
		}
		aiLogs[index].Actions = actions
	}
	favorites, err := s.allFavorites(ctx, userID)
	if err != nil {
		return domain.AccountExport{}, err
	}
	return domain.AccountExport{
		Profile:              profile,
		Courses:              courses,
		Reminders:            reminders,
		Parents:              parents,
		CommunicationRecords: records,
		HealingEntries:       healing,
		AIGenerations:        aiLogs,
		Favorites:            favorites,
		ExportedAt:           time.Now(),
	}, nil
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
	pendingReminders := pendingReminderCount(reminders)
	dueRecords, err := s.dueCommunicationRecords(ctx, userID, day)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	followUpParentIDs, followUpsCount := dueFollowUpParentIDs(dueRecords)
	followUpParents, err := s.parentsByIDs(ctx, userID, followUpParentIDs)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	next := nextCourseForDay(courses, day)
	rhythm := domain.RhythmState{Code: "steady", Title: "温柔但高效", Description: "今天事项可控，先处理下一节课，再推进家长跟进。"}
	if len(courses)+pendingReminders >= 5 {
		rhythm = domain.RhythmState{Code: "busy", Title: "节奏偏满", Description: "课程和待办较集中，建议按课前、课后、沟通三段处理。"}
	}
	return domain.DashboardSummary{TodayLabel: day.Format("2006-01-02"), CoursesCount: len(courses), RemindersCount: pendingReminders, FollowUpsCount: followUpsCount, NextCourse: next, Reminders: reminders, FocusParents: followUpParents, Rhythm: rhythm}, nil
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

func (s *PostgresStore) allCourses(ctx context.Context, userID domain.ID) ([]domain.Course, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, title, class_name, COALESCE(location, ''), weekday, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'), COALESCE(note, ''), created_at
		FROM courses
		WHERE user_id = $1
		ORDER BY weekday ASC, start_time ASC, created_at ASC`, string(userID))
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

func (s *PostgresStore) allReminders(ctx context.Context, userID domain.ID) ([]domain.Reminder, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, title, category, remind_at, status, COALESCE(note, ''), parent_id::text, course_id::text, created_at, done_at
		FROM reminders
		WHERE user_id = $1
		ORDER BY remind_at DESC, created_at DESC`, string(userID))
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

func (s *PostgresStore) Parents(ctx context.Context, userID domain.ID, options ListOptions) ([]domain.ParentProfile, error) {
	options = normalizeListOptions(options)
	query := `
		SELECT id::text, student_name, class_name, parent_name, relationship, COALESCE(contact, ''), COALESCE(communication_style, ''), risk_level, COALESCE(important_notes, ''), COALESCE(next_action, ''), created_at
		FROM parent_profiles
		WHERE user_id = $1`
	args := []any{string(userID)}
	if options.Query != "" {
		query += ` AND (student_name ILIKE $2 OR class_name ILIKE $2 OR parent_name ILIKE $2 OR relationship ILIKE $2 OR COALESCE(contact, '') ILIKE $2 OR COALESCE(communication_style, '') ILIKE $2 OR COALESCE(important_notes, '') ILIKE $2 OR COALESCE(next_action, '') ILIKE $2)`
		args = append(args, "%"+options.Query+"%")
	}
	query += fmt.Sprintf(" ORDER BY CASE risk_level WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END, updated_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, options.Limit, options.Offset)
	rows, err := s.pool.Query(ctx, query, args...)
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

func (s *PostgresStore) allParents(ctx context.Context, userID domain.ID) ([]domain.ParentProfile, error) {
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

func (s *PostgresStore) CommunicationRecords(ctx context.Context, userID domain.ID, parentID *domain.ID, options ListOptions) ([]domain.CommunicationRecord, error) {
	options = normalizeListOptions(options)
	query := `
		SELECT id::text, parent_id::text, student, channel, reason, summary, COALESCE(result, ''), risk_level, follow_up_at, follow_up_status, followed_up_at, created_at
		FROM communication_records
		WHERE user_id = $1`
	args := []any{string(userID)}
	if parentID != nil {
		query += fmt.Sprintf(" AND parent_id = $%d", len(args)+1)
		args = append(args, string(*parentID))
	}
	if options.Query != "" {
		query += fmt.Sprintf(" AND (student ILIKE $%d OR channel ILIKE $%d OR reason ILIKE $%d OR summary ILIKE $%d OR COALESCE(result, '') ILIKE $%d OR risk_level ILIKE $%d)", len(args)+1, len(args)+1, len(args)+1, len(args)+1, len(args)+1, len(args)+1)
		args = append(args, "%"+options.Query+"%")
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, options.Limit, options.Offset)
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.CommunicationRecord, 0)
	for rows.Next() {
		var item domain.CommunicationRecord
		var parent sql.NullString
		var followUp, followedUp sql.NullTime
		if err := rows.Scan(&item.ID, &parent, &item.Student, &item.Channel, &item.Reason, &item.Summary, &item.Result, &item.RiskLevel, &followUp, &item.FollowUpStatus, &followedUp, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.ParentID = nullableDomainID(parent)
		if followUp.Valid {
			item.FollowUpAt = followUp.Time
		}
		if followedUp.Valid {
			item.FollowedUpAt = &followedUp.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) allCommunicationRecords(ctx context.Context, userID domain.ID) ([]domain.CommunicationRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, parent_id::text, student, channel, reason, summary, COALESCE(result, ''), risk_level, follow_up_at, follow_up_status, followed_up_at, created_at
		FROM communication_records
		WHERE user_id = $1
		ORDER BY created_at DESC`, string(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.CommunicationRecord, 0)
	for rows.Next() {
		var item domain.CommunicationRecord
		var parent sql.NullString
		var followUp, followedUp sql.NullTime
		if err := rows.Scan(&item.ID, &parent, &item.Student, &item.Channel, &item.Reason, &item.Summary, &item.Result, &item.RiskLevel, &followUp, &item.FollowUpStatus, &followedUp, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.ParentID = nullableDomainID(parent)
		if followUp.Valid {
			item.FollowUpAt = followUp.Time
		}
		if followedUp.Valid {
			item.FollowedUpAt = &followedUp.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) dueCommunicationRecords(ctx context.Context, userID domain.ID, day time.Time) ([]domain.CommunicationRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, parent_id::text, student, channel, reason, summary, COALESCE(result, ''), risk_level, follow_up_at, follow_up_status, followed_up_at, created_at
		FROM communication_records
		WHERE user_id = $1 AND follow_up_status <> 'done' AND follow_up_at IS NOT NULL AND follow_up_at <= $2
		ORDER BY follow_up_at ASC, created_at DESC`, string(userID), endOfDay(day))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.CommunicationRecord, 0)
	for rows.Next() {
		var item domain.CommunicationRecord
		var parent sql.NullString
		var followUp, followedUp sql.NullTime
		if err := rows.Scan(&item.ID, &parent, &item.Student, &item.Channel, &item.Reason, &item.Summary, &item.Result, &item.RiskLevel, &followUp, &item.FollowUpStatus, &followedUp, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.ParentID = nullableDomainID(parent)
		if followUp.Valid {
			item.FollowUpAt = followUp.Time
		}
		if followedUp.Valid {
			item.FollowedUpAt = &followedUp.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) parentsByIDs(ctx context.Context, userID domain.ID, ids []domain.ID) ([]domain.ParentProfile, error) {
	if len(ids) == 0 {
		return []domain.ParentProfile{}, nil
	}
	args := []any{string(userID)}
	placeholders := make([]string, 0, len(ids))
	for _, id := range ids {
		args = append(args, string(id))
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
	}
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id::text, student_name, class_name, parent_name, relationship, COALESCE(contact, ''), COALESCE(communication_style, ''), risk_level, COALESCE(important_notes, ''), COALESCE(next_action, ''), created_at
		FROM parent_profiles
		WHERE user_id = $1 AND id IN (%s)`, strings.Join(placeholders, ",")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	parentByID := make(map[domain.ID]domain.ParentProfile, len(ids))
	for rows.Next() {
		var item domain.ParentProfile
		if err := rows.Scan(&item.ID, &item.StudentName, &item.ClassName, &item.ParentName, &item.Relationship, &item.Contact, &item.CommunicationStyle, &item.RiskLevel, &item.ImportantNotes, &item.NextAction, &item.CreatedAt); err != nil {
			return nil, err
		}
		parentByID[item.ID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	parents := make([]domain.ParentProfile, 0, len(ids))
	for _, id := range ids {
		if parent, ok := parentByID[id]; ok {
			parents = append(parents, parent)
		}
	}
	return parents, nil
}

func (s *PostgresStore) CommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID) (domain.CommunicationRecord, error) {
	var item domain.CommunicationRecord
	var parent sql.NullString
	var followUp, followedUp sql.NullTime
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, parent_id::text, student, channel, reason, summary, COALESCE(result, ''), risk_level, follow_up_at, follow_up_status, followed_up_at, created_at
		FROM communication_records
		WHERE user_id = $1 AND id = $2`, string(userID), string(id)).
		Scan(&item.ID, &parent, &item.Student, &item.Channel, &item.Reason, &item.Summary, &item.Result, &item.RiskLevel, &followUp, &item.FollowUpStatus, &followedUp, &item.CreatedAt)
	if isNoRows(err) {
		return domain.CommunicationRecord{}, notFound("communication record", id)
	}
	item.ParentID = nullableDomainID(parent)
	if followUp.Valid {
		item.FollowUpAt = followUp.Time
	}
	if followedUp.Valid {
		item.FollowedUpAt = &followedUp.Time
	}
	return item, err
}

func (s *PostgresStore) CreateCommunicationRecord(ctx context.Context, userID domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error) {
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
	if record.FollowUpStatus != "done" {
		record.FollowedUpAt = nil
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO communication_records (user_id, parent_id, student, channel, reason, summary, result, risk_level, follow_up_at, follow_up_status, followed_up_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id::text, created_at`, string(userID), fromDomainID(record.ParentID), record.Student, record.Channel, record.Reason, record.Summary, record.Result, record.RiskLevel, nullableTime(record.FollowUpAt), record.FollowUpStatus, fromTimePtr(record.FollowedUpAt)).
		Scan(&record.ID, &record.CreatedAt)
	return record, err
}

func (s *PostgresStore) UpdateCommunicationRecord(ctx context.Context, userID domain.ID, id domain.ID, record domain.CommunicationRecord) (domain.CommunicationRecord, error) {
	if record.RiskLevel == "" {
		record.RiskLevel = "low"
	}
	if record.FollowUpStatus == "" {
		current, err := s.CommunicationRecord(ctx, userID, id)
		if err != nil {
			return domain.CommunicationRecord{}, err
		}
		record.FollowUpStatus = current.FollowUpStatus
		record.FollowedUpAt = current.FollowedUpAt
	}
	if record.FollowUpStatus == "done" && record.FollowedUpAt == nil {
		now := time.Now()
		record.FollowedUpAt = &now
	}
	if record.FollowUpStatus != "done" {
		record.FollowedUpAt = nil
	}
	var parent sql.NullString
	var followUp, followedUp sql.NullTime
	err := s.pool.QueryRow(ctx, `
		UPDATE communication_records
		SET parent_id = $3, student = $4, channel = $5, reason = $6, summary = $7, result = $8, risk_level = $9, follow_up_at = $10, follow_up_status = $11, followed_up_at = $12, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, parent_id::text, student, channel, reason, summary, COALESCE(result, ''), risk_level, follow_up_at, follow_up_status, followed_up_at, created_at`,
		string(userID), string(id), fromDomainID(record.ParentID), record.Student, record.Channel, record.Reason, record.Summary, record.Result, record.RiskLevel, nullableTime(record.FollowUpAt), record.FollowUpStatus, fromTimePtr(record.FollowedUpAt)).
		Scan(&record.ID, &parent, &record.Student, &record.Channel, &record.Reason, &record.Summary, &record.Result, &record.RiskLevel, &followUp, &record.FollowUpStatus, &followedUp, &record.CreatedAt)
	if isNoRows(err) {
		return domain.CommunicationRecord{}, notFound("communication record", id)
	}
	record.ParentID = nullableDomainID(parent)
	if followUp.Valid {
		record.FollowUpAt = followUp.Time
	}
	if followedUp.Valid {
		record.FollowedUpAt = &followedUp.Time
	} else {
		record.FollowedUpAt = nil
	}
	return record, err
}

func (s *PostgresStore) CompleteCommunicationFollowUp(ctx context.Context, userID domain.ID, id domain.ID) (domain.CommunicationRecord, error) {
	var record domain.CommunicationRecord
	var parent sql.NullString
	var followUp, followedUp sql.NullTime
	err := s.pool.QueryRow(ctx, `
		UPDATE communication_records
		SET follow_up_status = 'done', followed_up_at = now(), updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, parent_id::text, student, channel, reason, summary, COALESCE(result, ''), risk_level, follow_up_at, follow_up_status, followed_up_at, created_at`,
		string(userID), string(id)).
		Scan(&record.ID, &parent, &record.Student, &record.Channel, &record.Reason, &record.Summary, &record.Result, &record.RiskLevel, &followUp, &record.FollowUpStatus, &followedUp, &record.CreatedAt)
	if isNoRows(err) {
		return domain.CommunicationRecord{}, notFound("communication record", id)
	}
	record.ParentID = nullableDomainID(parent)
	if followUp.Valid {
		record.FollowUpAt = followUp.Time
	}
	if followedUp.Valid {
		record.FollowedUpAt = &followedUp.Time
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

func (s *PostgresStore) HealingEntries(ctx context.Context, userID domain.ID, entryType string, options ListOptions) ([]domain.HealingEntry, error) {
	options = normalizeListOptions(options)
	query := `
		SELECT id::text, type, COALESCE(mood, ''), COALESCE(content, ''), COALESCE(ai_reply, ''), created_at
		FROM healing_entries
		WHERE user_id = $1`
	args := []any{string(userID)}
	if entryType != "" {
		query += fmt.Sprintf(" AND type = $%d", len(args)+1)
		args = append(args, entryType)
	}
	if options.Query != "" {
		query += fmt.Sprintf(" AND (type ILIKE $%d OR COALESCE(mood, '') ILIKE $%d OR COALESCE(content, '') ILIKE $%d OR COALESCE(ai_reply, '') ILIKE $%d)", len(args)+1, len(args)+1, len(args)+1, len(args)+1)
		args = append(args, "%"+options.Query+"%")
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, options.Limit, options.Offset)
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

func (s *PostgresStore) allHealingEntries(ctx context.Context, userID domain.ID) ([]domain.HealingEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, type, COALESCE(mood, ''), COALESCE(content, ''), COALESCE(ai_reply, ''), created_at
		FROM healing_entries
		WHERE user_id = $1
		ORDER BY created_at DESC`, string(userID))
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

func (s *PostgresStore) AIGenerations(ctx context.Context, userID domain.ID, scenario string, options ListOptions) ([]domain.AIGeneration, error) {
	options = normalizeListOptions(options)
	query := `
		SELECT id::text, scenario, input, output, safety_label, token_usage, created_at
		FROM ai_generations
		WHERE user_id = $1`
	args := []any{string(userID)}
	if scenario != "" {
		query += fmt.Sprintf(" AND scenario = $%d", len(args)+1)
		args = append(args, scenario)
	}
	if options.Query != "" {
		query += fmt.Sprintf(" AND (scenario ILIKE $%d OR safety_label ILIKE $%d OR input::text ILIKE $%d OR output::text ILIKE $%d)", len(args)+1, len(args)+1, len(args)+1, len(args)+1)
		args = append(args, "%"+options.Query+"%")
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, options.Limit, options.Offset)
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.AIGeneration, 0)
	for rows.Next() {
		item, err := scanAIGeneration(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) allAIGenerations(ctx context.Context, userID domain.ID) ([]domain.AIGeneration, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, scenario, input, output, safety_label, token_usage, created_at
		FROM ai_generations
		WHERE user_id = $1
		ORDER BY created_at DESC`, string(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.AIGeneration, 0)
	for rows.Next() {
		item, err := scanAIGeneration(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) AIGeneration(ctx context.Context, userID domain.ID, id domain.ID) (domain.AIGeneration, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id::text, scenario, input, output, safety_label, token_usage, created_at
		FROM ai_generations
		WHERE user_id = $1 AND id = $2`, string(userID), string(id))
	item, err := scanAIGeneration(row)
	if isNoRows(err) {
		return domain.AIGeneration{}, notFound("ai generation", id)
	}
	if err != nil {
		return domain.AIGeneration{}, err
	}
	actions, err := s.AIActions(ctx, userID, id)
	if err != nil {
		return domain.AIGeneration{}, err
	}
	item.Actions = actions
	return item, err
}

func (s *PostgresStore) CreateAIGeneration(ctx context.Context, userID domain.ID, generation domain.AIGeneration) (domain.AIGeneration, error) {
	if generation.Input == nil {
		generation.Input = map[string]any{}
	}
	if generation.Output == nil {
		generation.Output = map[string]any{}
	}
	if generation.SafetyLabel == "" {
		generation.SafetyLabel = "teacher_review_required"
	}
	input, err := json.Marshal(generation.Input)
	if err != nil {
		return domain.AIGeneration{}, err
	}
	output, err := json.Marshal(generation.Output)
	if err != nil {
		return domain.AIGeneration{}, err
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO ai_generations (user_id, scenario, input, output, safety_label, token_usage)
		VALUES ($1, $2, $3::jsonb, $4::jsonb, $5, $6)
		RETURNING id::text, created_at`, string(userID), generation.Scenario, string(input), string(output), generation.SafetyLabel, generation.TokenUsage).
		Scan(&generation.ID, &generation.CreatedAt)
	return generation, err
}

func (s *PostgresStore) UpdateAIGenerationOutput(ctx context.Context, userID domain.ID, id domain.ID, output map[string]any) error {
	if output == nil {
		output = map[string]any{}
	}
	data, err := json.Marshal(output)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE ai_generations
		SET output = $3::jsonb
		WHERE user_id = $1 AND id = $2`, string(userID), string(id), string(data))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("ai generation", id)
	}
	return nil
}

func (s *PostgresStore) DeleteAIGeneration(ctx context.Context, userID domain.ID, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM ai_generations WHERE user_id = $1 AND id = $2`, string(userID), string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("ai generation", id)
	}
	return nil
}

func (s *PostgresStore) CreateAIAction(ctx context.Context, userID domain.ID, action domain.AIAction) (domain.AIAction, error) {
	if _, err := s.AIGeneration(ctx, userID, action.GenerationID); err != nil {
		return domain.AIAction{}, err
	}
	if action.Metadata == nil {
		action.Metadata = map[string]any{}
	}
	metadata, err := json.Marshal(action.Metadata)
	if err != nil {
		return domain.AIAction{}, err
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO ai_generation_actions (user_id, generation_id, action, draft_id, note, metadata)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
		RETURNING id::text, generation_id::text, action, COALESCE(draft_id, ''), COALESCE(note, ''), metadata, created_at`,
		string(userID), string(action.GenerationID), action.Action, nullableString(action.DraftID), nullableString(action.Note), string(metadata)).
		Scan(&action.ID, &action.GenerationID, &action.Action, &action.DraftID, &action.Note, &metadata, &action.CreatedAt)
	if err != nil {
		return domain.AIAction{}, err
	}
	action.Metadata = map[string]any{}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &action.Metadata); err != nil {
			return domain.AIAction{}, err
		}
	}
	return action, nil
}

func (s *PostgresStore) AIActions(ctx context.Context, userID domain.ID, generationID domain.ID) ([]domain.AIAction, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, generation_id::text, action, COALESCE(draft_id, ''), COALESCE(note, ''), metadata, created_at
		FROM ai_generation_actions
		WHERE user_id = $1 AND generation_id = $2
		ORDER BY created_at ASC`, string(userID), string(generationID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.AIAction, 0)
	for rows.Next() {
		var item domain.AIAction
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.GenerationID, &item.Action, &item.DraftID, &item.Note, &metadata, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Metadata = map[string]any{}
		if len(metadata) > 0 {
			if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
				return nil, err
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) Favorites(ctx context.Context, userID domain.ID, favoriteType string, options ListOptions) ([]domain.Favorite, error) {
	options = normalizeListOptions(options)
	query := `
		SELECT id::text, type, title, content, source_id::text, created_at
		FROM favorites
		WHERE user_id = $1`
	args := []any{string(userID)}
	if favoriteType != "" {
		query += fmt.Sprintf(" AND type = $%d", len(args)+1)
		args = append(args, favoriteType)
	}
	if options.Query != "" {
		query += fmt.Sprintf(" AND (type ILIKE $%d OR title ILIKE $%d OR content ILIKE $%d)", len(args)+1, len(args)+1, len(args)+1)
		args = append(args, "%"+options.Query+"%")
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, options.Limit, options.Offset)
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

func (s *PostgresStore) allFavorites(ctx context.Context, userID domain.ID) ([]domain.Favorite, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, type, title, content, source_id::text, created_at
		FROM favorites
		WHERE user_id = $1
		ORDER BY created_at DESC`, string(userID))
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

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

func normalizeListOptions(options ListOptions) ListOptions {
	options.Query = strings.TrimSpace(options.Query)
	if options.Limit <= 0 || options.Limit > MaxListProbeLimit {
		options.Limit = DefaultListLimit
	}
	if options.Offset < 0 {
		options.Offset = 0
	}
	return options
}

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows)
}

type aiGenerationScanner interface {
	Scan(dest ...any) error
}

func scanAIGeneration(row aiGenerationScanner) (domain.AIGeneration, error) {
	var item domain.AIGeneration
	var inputRaw, outputRaw []byte
	if err := row.Scan(&item.ID, &item.Scenario, &inputRaw, &outputRaw, &item.SafetyLabel, &item.TokenUsage, &item.CreatedAt); err != nil {
		return domain.AIGeneration{}, err
	}
	item.Input = map[string]any{}
	if len(inputRaw) > 0 {
		if err := json.Unmarshal(inputRaw, &item.Input); err != nil {
			return domain.AIGeneration{}, err
		}
	}
	item.Output = map[string]any{}
	if len(outputRaw) > 0 {
		if err := json.Unmarshal(outputRaw, &item.Output); err != nil {
			return domain.AIGeneration{}, err
		}
	}
	return item, nil
}

var _ Store = (*PostgresStore)(nil)
