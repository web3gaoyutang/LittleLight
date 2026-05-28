package domain

import "time"

type ID string

type UserProfile struct {
	ID             ID        `json:"id"`
	Name           string    `json:"name"`
	School         string    `json:"school"`
	Stage          string    `json:"stage"`
	Subject        string    `json:"subject"`
	IsHeadTeacher  bool      `json:"isHeadTeacher"`
	ProStatus      string    `json:"proStatus"`
	ReminderPolicy string    `json:"reminderPolicy"`
	CreatedAt      time.Time `json:"createdAt"`
}

type WechatSession struct {
	UserID       ID          `json:"userId"`
	SessionToken string      `json:"sessionToken"`
	OpenID       string      `json:"openId"`
	Profile      UserProfile `json:"profile"`
	ExpiresAt    time.Time   `json:"expiresAt"`
}

type Course struct {
	ID        ID        `json:"id"`
	Title     string    `json:"title"`
	ClassName string    `json:"className"`
	Location  string    `json:"location"`
	Weekday   int       `json:"weekday"`
	StartTime string    `json:"startTime"`
	EndTime   string    `json:"endTime"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"createdAt"`
}

type Reminder struct {
	ID        ID         `json:"id"`
	Title     string     `json:"title"`
	Category  string     `json:"category"`
	RemindAt  time.Time  `json:"remindAt"`
	Status    string     `json:"status"`
	Note      string     `json:"note"`
	ParentID  *ID        `json:"parentId,omitempty"`
	CourseID  *ID        `json:"courseId,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	DoneAt    *time.Time `json:"doneAt,omitempty"`
}

type ParentProfile struct {
	ID                 ID        `json:"id"`
	StudentName        string    `json:"studentName"`
	ClassName          string    `json:"className"`
	ParentName         string    `json:"parentName"`
	Relationship       string    `json:"relationship"`
	Contact            string    `json:"contact"`
	CommunicationStyle string    `json:"communicationStyle"`
	RiskLevel          string    `json:"riskLevel"`
	ImportantNotes     string    `json:"importantNotes"`
	NextAction         string    `json:"nextAction"`
	CreatedAt          time.Time `json:"createdAt"`
}

type CommunicationRecord struct {
	ID         ID        `json:"id"`
	ParentID   ID        `json:"parentId"`
	Student    string    `json:"student"`
	Channel    string    `json:"channel"`
	Reason     string    `json:"reason"`
	Summary    string    `json:"summary"`
	Result     string    `json:"result"`
	RiskLevel  string    `json:"riskLevel"`
	FollowUpAt time.Time `json:"followUpAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

type HealingEntry struct {
	ID        ID        `json:"id"`
	Type      string    `json:"type"`
	Mood      string    `json:"mood"`
	Content   string    `json:"content"`
	AIReply   string    `json:"aiReply"`
	CreatedAt time.Time `json:"createdAt"`
}

type Favorite struct {
	ID        ID        `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	SourceID  *ID       `json:"sourceId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type AIDraft struct {
	ID      ID     `json:"id"`
	Version string `json:"version"`
	Tone    string `json:"tone"`
	Style   string `json:"style"`
	Content string `json:"content"`
	Safety  string `json:"safety"`
}

type AIGeneration struct {
	ID          ID             `json:"id"`
	Scenario    string         `json:"scenario"`
	Input       map[string]any `json:"input"`
	Output      map[string]any `json:"output"`
	SafetyLabel string         `json:"safetyLabel"`
	TokenUsage  int            `json:"tokenUsage"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type DashboardSummary struct {
	TodayLabel     string          `json:"todayLabel"`
	CoursesCount   int             `json:"coursesCount"`
	RemindersCount int             `json:"remindersCount"`
	FollowUpsCount int             `json:"followUpsCount"`
	NextCourse     *Course         `json:"nextCourse,omitempty"`
	Reminders      []Reminder      `json:"reminders"`
	FocusParents   []ParentProfile `json:"focusParents"`
	Rhythm         RhythmState     `json:"rhythm"`
}

type RhythmState struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
}
