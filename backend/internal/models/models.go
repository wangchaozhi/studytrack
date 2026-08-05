package models

import (
	"encoding/json"
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Nickname  string    `json:"nickname"`
	Timezone  string    `json:"timezone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Goal struct {
	ID          string          `json:"id"`
	UserID      string          `json:"user_id,omitempty"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	StartDate   string          `json:"start_date"`
	ExamDate    string          `json:"exam_date"`
	Status      string          `json:"status"`
	TargetScore float64         `json:"target_score"`
	Progress    float64         `json:"progress"`
	Settings    json.RawMessage `json:"settings"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type Phase struct {
	ID        string    `json:"id"`
	GoalID    string    `json:"goal_id"`
	Name      string    `json:"name"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	SortOrder int       `json:"sort_order"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Subject struct {
	ID             string    `json:"id"`
	GoalID         string    `json:"goal_id"`
	Name           string    `json:"name"`
	Category       string    `json:"category"`
	TargetScore    float64   `json:"target_score"`
	CurrentScore   float64   `json:"current_score"`
	PlannedMinutes int       `json:"planned_minutes"`
	Color          string    `json:"color"`
	CreatedAt      time.Time `json:"created_at"`
}

type KnowledgePoint struct {
	ID           string     `json:"id"`
	SubjectID    string     `json:"subject_id"`
	ParentID     *string    `json:"parent_id"`
	Name         string     `json:"name"`
	Importance   string     `json:"importance"`
	Mastery      float64    `json:"mastery"`
	ReviewStatus string     `json:"review_status"`
	NextReviewAt *time.Time `json:"next_review_at"`
	ReviewCount  int        `json:"review_count"`
	WrongCount   int        `json:"wrong_count"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Task struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id,omitempty"`
	GoalID           string     `json:"goal_id"`
	PhaseID          *string    `json:"phase_id"`
	SubjectID        *string    `json:"subject_id"`
	KnowledgePointID *string    `json:"knowledge_point_id"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	TaskType         string     `json:"task_type"`
	Priority         string     `json:"priority"`
	Status           string     `json:"status"`
	PlannedDate      string     `json:"planned_date"`
	EstimatedMinutes int        `json:"estimated_minutes"`
	ActualMinutes    int        `json:"actual_minutes"`
	CompletedAt      *time.Time `json:"completed_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	GoalName         string     `json:"goal_name,omitempty"`
	SubjectName      string     `json:"subject_name,omitempty"`
}

type StudySession struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id,omitempty"`
	GoalID          string     `json:"goal_id"`
	TaskID          *string    `json:"task_id"`
	SubjectID       *string    `json:"subject_id"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at"`
	DurationMinutes int        `json:"duration_minutes"`
	FocusScore      int        `json:"focus_score"`
	MasteryScore    int        `json:"mastery_score"`
	Notes           string     `json:"notes"`
	CreatedAt       time.Time  `json:"created_at"`
}

type Assessment struct {
	ID              string          `json:"id"`
	GoalID          string          `json:"goal_id"`
	SubjectID       *string         `json:"subject_id"`
	Name            string          `json:"name"`
	AssessmentType  string          `json:"assessment_type"`
	ExamDate        string          `json:"exam_date"`
	Score           float64         `json:"score"`
	FullScore       float64         `json:"full_score"`
	DurationMinutes int             `json:"duration_minutes"`
	Details         json.RawMessage `json:"details"`
	CreatedAt       time.Time       `json:"created_at"`
}

type WeeklyReview struct {
	ID             string    `json:"id"`
	GoalID         string    `json:"goal_id"`
	WeekStart      string    `json:"week_start"`
	PlannedMinutes int       `json:"planned_minutes"`
	ActualMinutes  int       `json:"actual_minutes"`
	CompletionRate float64   `json:"completion_rate"`
	Summary        string    `json:"summary"`
	Problems       string    `json:"problems"`
	NextWeekFocus  string    `json:"next_week_focus"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Dashboard struct {
	TodayMinutes      int               `json:"today_minutes"`
	WeekMinutes       int               `json:"week_minutes"`
	TodayCompleted    int               `json:"today_completed"`
	TodayTotal        int               `json:"today_total"`
	StreakDays        int               `json:"streak_days"`
	OverdueTasks      int               `json:"overdue_tasks"`
	Goals             []Goal            `json:"goals"`
	TodayTasks        []Task            `json:"today_tasks"`
	SubjectProgress   []SubjectProgress `json:"subject_progress"`
	WeakKnowledge     []KnowledgePoint  `json:"weak_knowledge"`
	DailyStudyTrend   []DailyMetric     `json:"daily_study_trend"`
	RecentAssessments []Assessment      `json:"recent_assessments"`
}

type SubjectProgress struct {
	SubjectID   string  `json:"subject_id"`
	SubjectName string  `json:"subject_name"`
	Progress    float64 `json:"progress"`
	Minutes     int     `json:"minutes"`
	Color       string  `json:"color"`
}

type DailyMetric struct {
	Date    string  `json:"date"`
	Value   float64 `json:"value"`
	Minutes int     `json:"minutes,omitempty"`
	Count   int     `json:"count,omitempty"`
}

type AnalyticsOverview struct {
	TotalMinutes          int               `json:"total_minutes"`
	CompletedTasks        int               `json:"completed_tasks"`
	CompletionRate        float64           `json:"completion_rate"`
	AverageFocusScore     float64           `json:"average_focus_score"`
	AverageMasteryScore   float64           `json:"average_mastery_score"`
	AverageSessionMinutes float64           `json:"average_session_minutes"`
	OverdueTasks          int               `json:"overdue_tasks"`
	BestStudyDate         string            `json:"best_study_date"`
	BestStudyMinutes      int               `json:"best_study_minutes"`
	MostFocusedSubject    string            `json:"most_focused_subject"`
	StudyTime             []DailyMetric     `json:"study_time"`
	TaskCompletion        []DailyMetric     `json:"task_completion"`
	SubjectProgress       []SubjectProgress `json:"subject_progress"`
	ScoreTrends           []Assessment      `json:"score_trends"`
	WeakKnowledge         []KnowledgePoint  `json:"weak_knowledge"`
}
