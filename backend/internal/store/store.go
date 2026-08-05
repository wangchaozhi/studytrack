package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"studytrack/backend/internal/models"
)

var ErrNotFound = errors.New("not found")

type Store struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Store { return &Store{db: db} }

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, nickname, timezone string) (models.User, error) {
	var u models.User
	err := s.db.QueryRow(ctx, `
		INSERT INTO users(email,password_hash,nickname,timezone)
		VALUES(lower($1),$2,$3,$4)
		RETURNING id::text,email,nickname,timezone,created_at,updated_at`, email, passwordHash, nickname, timezone).
		Scan(&u.ID, &u.Email, &u.Nickname, &u.Timezone, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (models.User, string, error) {
	var u models.User
	var hash string
	err := s.db.QueryRow(ctx, `SELECT id::text,email,password_hash,nickname,timezone,created_at,updated_at FROM users WHERE email=lower($1)`, email).
		Scan(&u.ID, &u.Email, &hash, &u.Nickname, &u.Timezone, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, "", ErrNotFound
	}
	return u, hash, err
}

func (s *Store) GetUser(ctx context.Context, id string) (models.User, error) {
	var u models.User
	err := s.db.QueryRow(ctx, `SELECT id::text,email,nickname,timezone,created_at,updated_at FROM users WHERE id=$1`, id).
		Scan(&u.ID, &u.Email, &u.Nickname, &u.Timezone, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *Store) UpdateUser(ctx context.Context, id, nickname, timezone string) (models.User, error) {
	var u models.User
	err := s.db.QueryRow(ctx, `UPDATE users SET nickname=$2, timezone=$3 WHERE id=$1 RETURNING id::text,email,nickname,timezone,created_at,updated_at`, id, nickname, timezone).
		Scan(&u.ID, &u.Email, &u.Nickname, &u.Timezone, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func scanGoal(row pgx.Row) (models.Goal, error) {
	var g models.Goal
	err := row.Scan(&g.ID, &g.UserID, &g.Name, &g.Type, &g.StartDate, &g.ExamDate, &g.Status, &g.TargetScore, &g.Progress, &g.Settings, &g.CreatedAt, &g.UpdatedAt)
	return g, err
}

const goalSelect = `SELECT id::text,user_id::text,name,type,to_char(start_date,'YYYY-MM-DD'),to_char(exam_date,'YYYY-MM-DD'),status,target_score::float8,progress::float8,settings,created_at,updated_at FROM goals`

func (s *Store) ListGoals(ctx context.Context, userID string) ([]models.Goal, error) {
	rows, err := s.db.Query(ctx, goalSelect+` WHERE user_id=$1 ORDER BY status='active' DESC, exam_date`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Goal{}
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, g)
	}
	return items, rows.Err()
}

func (s *Store) CreateGoal(ctx context.Context, userID string, g models.Goal) (models.Goal, error) {
	if len(g.Settings) == 0 {
		g.Settings = json.RawMessage(`{}`)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return g, err
	}
	defer tx.Rollback(ctx)
	created, err := scanGoal(tx.QueryRow(ctx, `
		INSERT INTO goals(user_id,name,type,start_date,exam_date,status,target_score,progress,settings)
		VALUES($1,$2,$3,$4,$5,COALESCE(NULLIF($6,''),'active'),$7,$8,$9)
		RETURNING id::text,user_id::text,name,type,to_char(start_date,'YYYY-MM-DD'),to_char(exam_date,'YYYY-MM-DD'),status,target_score::float8,progress::float8,settings,created_at,updated_at`,
		userID, g.Name, g.Type, g.StartDate, g.ExamDate, g.Status, g.TargetScore, g.Progress, g.Settings))
	if err != nil {
		return g, err
	}
	if err := createGoalTemplate(ctx, tx, created); err != nil {
		return g, err
	}
	if err := tx.Commit(ctx); err != nil {
		return g, err
	}
	return created, nil
}

func createGoalTemplate(ctx context.Context, tx pgx.Tx, g models.Goal) error {
	phases := []string{"基础阶段", "强化阶段", "冲刺阶段"}
	type templateSubject struct {
		name, category, color string
		target                float64
	}
	var settings struct {
		Courses []struct {
			Name        string  `json:"name"`
			TargetScore float64 `json:"target_score"`
		} `json:"courses"`
		PoliticsTarget     float64 `json:"politics_target"`
		EnglishTarget      float64 `json:"english_target"`
		MathTarget         float64 `json:"math_target"`
		ProfessionalTarget float64 `json:"professional_target"`
		ListeningTarget    float64 `json:"listening_target"`
		ReadingTarget      float64 `json:"reading_target"`
		WritingTarget      float64 `json:"writing_target"`
		SpeakingTarget     float64 `json:"speaking_target"`
	}
	_ = json.Unmarshal(g.Settings, &settings)
	subjects := []templateSubject{}
	switch g.Type {
	case "degree":
		phases = []string{"资料整理", "基础复习", "章节练习", "模拟考试", "考前冲刺"}
		colors := []string{"#4f46e5", "#0f766e", "#d97706", "#dc2626", "#2563eb", "#db2777"}
		for i, course := range settings.Courses {
			if strings.TrimSpace(course.Name) != "" {
				subjects = append(subjects, templateSubject{strings.TrimSpace(course.Name), "degree", colors[i%len(colors)], course.TargetScore})
			}
		}
		if len(subjects) == 0 {
			subjects = []templateSubject{{"专业课程一", "degree", "#4f46e5", 0}, {"专业课程二", "degree", "#0f766e", 0}}
		}
	case "postgraduate":
		phases = []string{"基础轮", "强化轮", "真题轮", "冲刺轮"}
		subjects = []templateSubject{{"政治", "politics", "#dc2626", settings.PoliticsTarget}, {"英语", "english", "#2563eb", settings.EnglishTarget}, {"数学", "math", "#7c3aed", settings.MathTarget}, {"专业课", "professional", "#059669", settings.ProfessionalTarget}}
	case "ielts":
		phases = []string{"词汇基础", "听力专项", "阅读专项", "写作专项", "口语专项", "整套模考"}
		subjects = []templateSubject{{"听力", "listening", "#2563eb", settings.ListeningTarget}, {"阅读", "reading", "#7c3aed", settings.ReadingTarget}, {"写作", "writing", "#d97706", settings.WritingTarget}, {"口语", "speaking", "#059669", settings.SpeakingTarget}, {"词汇", "vocabulary", "#db2777", 0}}
	}
	start, _ := time.Parse("2006-01-02", g.StartDate)
	end, _ := time.Parse("2006-01-02", g.ExamDate)
	if !end.After(start) {
		end = start.AddDate(0, 3, 0)
	}
	days := int(end.Sub(start).Hours() / 24)
	if days < len(phases) {
		days = len(phases)
	}
	for i, name := range phases {
		phaseStart := start.AddDate(0, 0, days*i/len(phases))
		phaseEnd := start.AddDate(0, 0, days*(i+1)/len(phases))
		status := "pending"
		if i == 0 {
			status = "active"
		}
		_, err := tx.Exec(ctx, `INSERT INTO phases(goal_id,name,start_date,end_date,sort_order,status) VALUES($1,$2,$3,$4,$5,$6)`, g.ID, name, phaseStart, phaseEnd, i, status)
		if err != nil {
			return err
		}
	}
	for _, sub := range subjects {
		_, err := tx.Exec(ctx, `INSERT INTO subjects(goal_id,name,category,color,target_score) VALUES($1,$2,$3,$4,$5)`, g.ID, sub.name, sub.category, sub.color, sub.target)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetGoal(ctx context.Context, userID, goalID string) (models.Goal, error) {
	g, err := scanGoal(s.db.QueryRow(ctx, goalSelect+` WHERE user_id=$1 AND id=$2`, userID, goalID))
	if errors.Is(err, pgx.ErrNoRows) {
		return g, ErrNotFound
	}
	return g, err
}

func (s *Store) UpdateGoal(ctx context.Context, userID string, g models.Goal) (models.Goal, error) {
	if len(g.Settings) == 0 {
		g.Settings = json.RawMessage(`{}`)
	}
	updated, err := scanGoal(s.db.QueryRow(ctx, `
		UPDATE goals SET name=$3,type=$4,start_date=$5,exam_date=$6,status=$7,target_score=$8,progress=$9,settings=$10
		WHERE user_id=$1 AND id=$2
		RETURNING id::text,user_id::text,name,type,to_char(start_date,'YYYY-MM-DD'),to_char(exam_date,'YYYY-MM-DD'),status,target_score::float8,progress::float8,settings,created_at,updated_at`,
		userID, g.ID, g.Name, g.Type, g.StartDate, g.ExamDate, g.Status, g.TargetScore, g.Progress, g.Settings))
	if errors.Is(err, pgx.ErrNoRows) {
		return updated, ErrNotFound
	}
	return updated, err
}

func (s *Store) DeleteGoal(ctx context.Context, userID, goalID string) error {
	result, err := s.db.Exec(ctx, `DELETE FROM goals WHERE user_id=$1 AND id=$2`, userID, goalID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListPhases(ctx context.Context, userID, goalID string) ([]models.Phase, error) {
	rows, err := s.db.Query(ctx, `SELECT p.id::text,p.goal_id::text,p.name,to_char(p.start_date,'YYYY-MM-DD'),to_char(p.end_date,'YYYY-MM-DD'),p.sort_order,p.status,p.created_at FROM phases p JOIN goals g ON g.id=p.goal_id WHERE g.user_id=$1 AND p.goal_id=$2 ORDER BY p.sort_order`, userID, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Phase{}
	for rows.Next() {
		var x models.Phase
		if err := rows.Scan(&x.ID, &x.GoalID, &x.Name, &x.StartDate, &x.EndDate, &x.SortOrder, &x.Status, &x.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func (s *Store) CreatePhase(ctx context.Context, userID string, p models.Phase) (models.Phase, error) {
	var x models.Phase
	err := s.db.QueryRow(ctx, `INSERT INTO phases(goal_id,name,start_date,end_date,sort_order,status) SELECT $2,$3,$4,$5,$6,$7 FROM goals WHERE id=$2 AND user_id=$1 RETURNING id::text,goal_id::text,name,to_char(start_date,'YYYY-MM-DD'),to_char(end_date,'YYYY-MM-DD'),sort_order,status,created_at`, userID, p.GoalID, p.Name, p.StartDate, p.EndDate, p.SortOrder, p.Status).Scan(&x.ID, &x.GoalID, &x.Name, &x.StartDate, &x.EndDate, &x.SortOrder, &x.Status, &x.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return x, ErrNotFound
	}
	return x, err
}

func (s *Store) ListSubjects(ctx context.Context, userID, goalID string) ([]models.Subject, error) {
	rows, err := s.db.Query(ctx, `SELECT s.id::text,s.goal_id::text,s.name,s.category,s.target_score::float8,s.current_score::float8,s.planned_minutes,s.color,s.created_at FROM subjects s JOIN goals g ON g.id=s.goal_id WHERE g.user_id=$1 AND s.goal_id=$2 ORDER BY s.created_at`, userID, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Subject{}
	for rows.Next() {
		var x models.Subject
		if err := rows.Scan(&x.ID, &x.GoalID, &x.Name, &x.Category, &x.TargetScore, &x.CurrentScore, &x.PlannedMinutes, &x.Color, &x.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func (s *Store) CreateSubject(ctx context.Context, userID string, x models.Subject) (models.Subject, error) {
	var y models.Subject
	err := s.db.QueryRow(ctx, `INSERT INTO subjects(goal_id,name,category,target_score,current_score,planned_minutes,color) SELECT $2,$3,$4,$5,$6,$7,$8 FROM goals WHERE id=$2 AND user_id=$1 RETURNING id::text,goal_id::text,name,category,target_score::float8,current_score::float8,planned_minutes,color,created_at`, userID, x.GoalID, x.Name, x.Category, x.TargetScore, x.CurrentScore, x.PlannedMinutes, x.Color).Scan(&y.ID, &y.GoalID, &y.Name, &y.Category, &y.TargetScore, &y.CurrentScore, &y.PlannedMinutes, &y.Color, &y.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return y, ErrNotFound
	}
	return y, err
}

func (s *Store) UpdateSubject(ctx context.Context, userID string, x models.Subject) (models.Subject, error) {
	var y models.Subject
	err := s.db.QueryRow(ctx, `UPDATE subjects s SET name=$3,category=$4,target_score=$5,current_score=$6,planned_minutes=$7,color=$8 FROM goals g WHERE s.id=$2 AND s.goal_id=g.id AND g.user_id=$1 RETURNING s.id::text,s.goal_id::text,s.name,s.category,s.target_score::float8,s.current_score::float8,s.planned_minutes,s.color,s.created_at`, userID, x.ID, x.Name, x.Category, x.TargetScore, x.CurrentScore, x.PlannedMinutes, x.Color).Scan(&y.ID, &y.GoalID, &y.Name, &y.Category, &y.TargetScore, &y.CurrentScore, &y.PlannedMinutes, &y.Color, &y.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return y, ErrNotFound
	}
	return y, err
}

type TaskFilter struct{ Date, GoalID, Status, From, To string }

func scanTask(row pgx.Row) (models.Task, error) {
	var x models.Task
	err := row.Scan(&x.ID, &x.UserID, &x.GoalID, &x.PhaseID, &x.SubjectID, &x.KnowledgePointID, &x.Title, &x.Description, &x.TaskType, &x.Priority, &x.Status, &x.PlannedDate, &x.EstimatedMinutes, &x.ActualMinutes, &x.CompletedAt, &x.CreatedAt, &x.UpdatedAt, &x.GoalName, &x.SubjectName)
	return x, err
}

const taskSelect = `SELECT t.id::text,t.user_id::text,t.goal_id::text,t.phase_id::text,t.subject_id::text,t.knowledge_point_id::text,t.title,t.description,t.task_type,t.priority,t.status,to_char(t.planned_date,'YYYY-MM-DD'),t.estimated_minutes,t.actual_minutes,t.completed_at,t.created_at,t.updated_at,g.name,COALESCE(s.name,'') FROM tasks t JOIN goals g ON g.id=t.goal_id LEFT JOIN subjects s ON s.id=t.subject_id`

func (s *Store) ListTasks(ctx context.Context, userID string, f TaskFilter) ([]models.Task, error) {
	_, _ = s.db.Exec(ctx, `UPDATE tasks SET status='overdue' WHERE user_id=$1 AND planned_date<(now() AT TIME ZONE COALESCE((SELECT timezone FROM users WHERE id=$1),'UTC'))::date AND status IN ('pending','in_progress')`, userID)
	q := taskSelect + ` WHERE t.user_id=$1`
	args := []any{userID}
	n := 2
	add := func(cond string, v any) { q += fmt.Sprintf(" AND "+cond, n); args = append(args, v); n++ }
	if f.Date != "" {
		add("t.planned_date=$%d", f.Date)
	}
	if f.GoalID != "" {
		add("t.goal_id=$%d", f.GoalID)
	}
	if f.Status != "" {
		add("t.status=$%d", f.Status)
	}
	if f.From != "" {
		add("t.planned_date >= $%d", f.From)
	}
	if f.To != "" {
		add("t.planned_date <= $%d", f.To)
	}
	q += ` ORDER BY t.planned_date,t.status='completed',CASE t.priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END,t.created_at`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Task{}
	for rows.Next() {
		x, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func (s *Store) CreateTask(ctx context.Context, userID string, x models.Task) (models.Task, error) {
	q := `WITH inserted AS (
	INSERT INTO tasks(user_id,goal_id,phase_id,subject_id,knowledge_point_id,title,description,task_type,priority,status,planned_date,estimated_minutes,actual_minutes)
	SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,COALESCE(NULLIF($10,''),'pending'),$11,$12,$13
	WHERE EXISTS(SELECT 1 FROM goals WHERE id=$2 AND user_id=$1)
	AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM phases WHERE id=$3 AND goal_id=$2))
	AND ($4::uuid IS NULL OR EXISTS(SELECT 1 FROM subjects WHERE id=$4 AND goal_id=$2))
	AND ($5::uuid IS NULL OR EXISTS(SELECT 1 FROM knowledge_points WHERE id=$5 AND subject_id=$4))
	RETURNING *)
	SELECT t.id::text,t.user_id::text,t.goal_id::text,t.phase_id::text,t.subject_id::text,t.knowledge_point_id::text,t.title,t.description,t.task_type,t.priority,t.status,to_char(t.planned_date,'YYYY-MM-DD'),t.estimated_minutes,t.actual_minutes,t.completed_at,t.created_at,t.updated_at,g.name,COALESCE(s.name,'') FROM inserted t JOIN goals g ON g.id=t.goal_id LEFT JOIN subjects s ON s.id=t.subject_id`
	y, err := scanTask(s.db.QueryRow(ctx, q, userID, x.GoalID, x.PhaseID, x.SubjectID, x.KnowledgePointID, x.Title, x.Description, x.TaskType, x.Priority, x.Status, x.PlannedDate, x.EstimatedMinutes, x.ActualMinutes))
	if errors.Is(err, pgx.ErrNoRows) {
		return y, ErrNotFound
	}
	return y, err
}

func (s *Store) UpdateTask(ctx context.Context, userID string, x models.Task) (models.Task, error) {
	q := `WITH updated AS (
	UPDATE tasks SET goal_id=$3,phase_id=$4,subject_id=$5,knowledge_point_id=$6,title=$7,description=$8,task_type=$9,priority=$10,status=$11,planned_date=$12,estimated_minutes=$13,actual_minutes=$14,
	completed_at=CASE WHEN $11='completed' THEN COALESCE(completed_at,now()) ELSE NULL END
	WHERE id=$2 AND user_id=$1
	AND EXISTS(SELECT 1 FROM goals WHERE id=$3 AND user_id=$1)
	AND ($4::uuid IS NULL OR EXISTS(SELECT 1 FROM phases WHERE id=$4 AND goal_id=$3))
	AND ($5::uuid IS NULL OR EXISTS(SELECT 1 FROM subjects WHERE id=$5 AND goal_id=$3))
	AND ($6::uuid IS NULL OR EXISTS(SELECT 1 FROM knowledge_points WHERE id=$6 AND subject_id=$5))
	RETURNING *)
	SELECT t.id::text,t.user_id::text,t.goal_id::text,t.phase_id::text,t.subject_id::text,t.knowledge_point_id::text,t.title,t.description,t.task_type,t.priority,t.status,to_char(t.planned_date,'YYYY-MM-DD'),t.estimated_minutes,t.actual_minutes,t.completed_at,t.created_at,t.updated_at,g.name,COALESCE(s.name,'') FROM updated t JOIN goals g ON g.id=t.goal_id LEFT JOIN subjects s ON s.id=t.subject_id`
	y, err := scanTask(s.db.QueryRow(ctx, q, userID, x.ID, x.GoalID, x.PhaseID, x.SubjectID, x.KnowledgePointID, x.Title, x.Description, x.TaskType, x.Priority, x.Status, x.PlannedDate, x.EstimatedMinutes, x.ActualMinutes))
	if errors.Is(err, pgx.ErrNoRows) {
		return y, ErrNotFound
	}
	return y, err
}

func (s *Store) UpdateTaskStatus(ctx context.Context, userID, taskID, status string) (models.Task, error) {
	q := `WITH updated AS (UPDATE tasks SET status=$3,completed_at=CASE WHEN $3='completed' THEN now() ELSE NULL END WHERE id=$2 AND user_id=$1 RETURNING *) SELECT t.id::text,t.user_id::text,t.goal_id::text,t.phase_id::text,t.subject_id::text,t.knowledge_point_id::text,t.title,t.description,t.task_type,t.priority,t.status,to_char(t.planned_date,'YYYY-MM-DD'),t.estimated_minutes,t.actual_minutes,t.completed_at,t.created_at,t.updated_at,g.name,COALESCE(s.name,'') FROM updated t JOIN goals g ON g.id=t.goal_id LEFT JOIN subjects s ON s.id=t.subject_id`
	x, err := scanTask(s.db.QueryRow(ctx, q, userID, taskID, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return x, ErrNotFound
	}
	return x, err
}

func (s *Store) DeleteTask(ctx context.Context, userID, taskID string) error {
	r, err := s.db.Exec(ctx, `DELETE FROM tasks WHERE id=$2 AND user_id=$1`, userID, taskID)
	if err != nil {
		return err
	}
	if r.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanSession(row pgx.Row) (models.StudySession, error) {
	var x models.StudySession
	err := row.Scan(&x.ID, &x.UserID, &x.GoalID, &x.TaskID, &x.SubjectID, &x.StartedAt, &x.EndedAt, &x.DurationMinutes, &x.FocusScore, &x.MasteryScore, &x.Notes, &x.CreatedAt)
	return x, err
}

const sessionSelect = `SELECT id::text,user_id::text,goal_id::text,task_id::text,subject_id::text,started_at,ended_at,duration_minutes,focus_score,mastery_score,notes,created_at FROM study_sessions`

func (s *Store) StartSession(ctx context.Context, userID string, x models.StudySession) (models.StudySession, error) {
	y, err := scanSession(s.db.QueryRow(ctx, `INSERT INTO study_sessions(user_id,goal_id,task_id,subject_id,started_at) SELECT $1,$2,$3,$4,COALESCE($5,now()) WHERE EXISTS(SELECT 1 FROM goals WHERE id=$2 AND user_id=$1)
	AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM tasks WHERE id=$3 AND user_id=$1 AND goal_id=$2))
	AND ($4::uuid IS NULL OR EXISTS(SELECT 1 FROM subjects WHERE id=$4 AND goal_id=$2))
	RETURNING id::text,user_id::text,goal_id::text,task_id::text,subject_id::text,started_at,ended_at,duration_minutes,focus_score,mastery_score,notes,created_at`, userID, x.GoalID, x.TaskID, x.SubjectID, nullableTime(x.StartedAt)))
	if errors.Is(err, pgx.ErrNoRows) {
		return y, ErrNotFound
	}
	return y, err
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func (s *Store) FinishSession(ctx context.Context, userID string, x models.StudySession) (models.StudySession, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return x, err
	}
	defer tx.Rollback(ctx)
	ended := time.Now()
	if x.EndedAt != nil {
		ended = *x.EndedAt
	}
	var y models.StudySession
	err = tx.QueryRow(ctx, `UPDATE study_sessions SET ended_at=$3,duration_minutes=CASE WHEN $4>0 THEN $4 ELSE GREATEST(1,ROUND(EXTRACT(EPOCH FROM ($3-started_at))/60)::int) END,focus_score=$5,mastery_score=$6,notes=$7 WHERE id=$2 AND user_id=$1 RETURNING id::text,user_id::text,goal_id::text,task_id::text,subject_id::text,started_at,ended_at,duration_minutes,focus_score,mastery_score,notes,created_at`, userID, x.ID, ended, x.DurationMinutes, x.FocusScore, x.MasteryScore, x.Notes).Scan(&y.ID, &y.UserID, &y.GoalID, &y.TaskID, &y.SubjectID, &y.StartedAt, &y.EndedAt, &y.DurationMinutes, &y.FocusScore, &y.MasteryScore, &y.Notes, &y.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return y, ErrNotFound
	}
	if err != nil {
		return y, err
	}
	if y.TaskID != nil {
		_, err = tx.Exec(ctx, `UPDATE tasks SET actual_minutes=actual_minutes+$2,status=CASE WHEN status='pending' THEN 'in_progress' ELSE status END WHERE id=$1`, *y.TaskID, y.DurationMinutes)
		if err != nil {
			return y, err
		}
	}
	if y.SubjectID != nil && y.TaskID != nil && x.MasteryScore > 0 {
		_, err = tx.Exec(ctx, `UPDATE knowledge_points SET mastery=LEAST(100,(mastery*review_count+$2*20)/(review_count+1)),review_count=review_count+1,review_status=CASE WHEN ((mastery*review_count+$2*20)/(review_count+1))>=80 THEN 'mastered' ELSE 'learning' END,next_review_at=now()+CASE WHEN $2>=4 THEN interval '7 days' WHEN $2=3 THEN interval '3 days' ELSE interval '1 day' END WHERE subject_id=$1 AND id=(SELECT knowledge_point_id FROM tasks WHERE id=$3)`, *y.SubjectID, x.MasteryScore, *y.TaskID)
		if err != nil {
			return y, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return y, err
	}
	return y, nil
}

func (s *Store) ListSessions(ctx context.Context, userID, goalID string, limit int) ([]models.StudySession, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := sessionSelect + ` WHERE user_id=$1`
	args := []any{userID}
	if goalID != "" {
		q += ` AND goal_id=$2`
		args = append(args, goalID)
	}
	q += ` ORDER BY started_at DESC LIMIT ` + fmt.Sprint(limit)
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.StudySession{}
	for rows.Next() {
		x, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func scanKnowledge(row pgx.Row) (models.KnowledgePoint, error) {
	var x models.KnowledgePoint
	err := row.Scan(&x.ID, &x.SubjectID, &x.ParentID, &x.Name, &x.Importance, &x.Mastery, &x.ReviewStatus, &x.NextReviewAt, &x.ReviewCount, &x.WrongCount, &x.CreatedAt)
	return x, err
}

const knowledgeSelect = `SELECT k.id::text,k.subject_id::text,k.parent_id::text,k.name,k.importance,k.mastery::float8,k.review_status,k.next_review_at,k.review_count,k.wrong_count,k.created_at FROM knowledge_points k`

func (s *Store) ListKnowledge(ctx context.Context, userID, subjectID string) ([]models.KnowledgePoint, error) {
	rows, err := s.db.Query(ctx, knowledgeSelect+` JOIN subjects s ON s.id=k.subject_id JOIN goals g ON g.id=s.goal_id WHERE g.user_id=$1 AND k.subject_id=$2 ORDER BY k.parent_id NULLS FIRST,k.created_at`, userID, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.KnowledgePoint{}
	for rows.Next() {
		x, err := scanKnowledge(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func (s *Store) CreateKnowledge(ctx context.Context, userID string, x models.KnowledgePoint) (models.KnowledgePoint, error) {
	q := `INSERT INTO knowledge_points(subject_id,parent_id,name,importance,mastery,review_status,next_review_at,review_count,wrong_count) SELECT $2,$3,$4,$5,$6,$7,$8,$9,$10 FROM subjects s JOIN goals g ON g.id=s.goal_id WHERE s.id=$2 AND g.user_id=$1
	AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM knowledge_points p WHERE p.id=$3 AND p.subject_id=$2))
	RETURNING id::text,subject_id::text,parent_id::text,name,importance,mastery::float8,review_status,next_review_at,review_count,wrong_count,created_at`
	y, err := scanKnowledge(s.db.QueryRow(ctx, q, userID, x.SubjectID, x.ParentID, x.Name, x.Importance, x.Mastery, x.ReviewStatus, x.NextReviewAt, x.ReviewCount, x.WrongCount))
	if errors.Is(err, pgx.ErrNoRows) {
		return y, ErrNotFound
	}
	return y, err
}

func (s *Store) UpdateKnowledge(ctx context.Context, userID string, x models.KnowledgePoint) (models.KnowledgePoint, error) {
	q := `UPDATE knowledge_points k SET parent_id=$3,name=$4,importance=$5,mastery=$6,review_status=$7,next_review_at=$8,review_count=$9,wrong_count=$10 FROM subjects s JOIN goals g ON g.id=s.goal_id WHERE k.id=$2 AND k.subject_id=s.id AND g.user_id=$1
	AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM knowledge_points p WHERE p.id=$3 AND p.subject_id=k.subject_id AND p.id<>k.id))
	RETURNING k.id::text,k.subject_id::text,k.parent_id::text,k.name,k.importance,k.mastery::float8,k.review_status,k.next_review_at,k.review_count,k.wrong_count,k.created_at`
	y, err := scanKnowledge(s.db.QueryRow(ctx, q, userID, x.ID, x.ParentID, x.Name, x.Importance, x.Mastery, x.ReviewStatus, x.NextReviewAt, x.ReviewCount, x.WrongCount))
	if errors.Is(err, pgx.ErrNoRows) {
		return y, ErrNotFound
	}
	return y, err
}

func (s *Store) ReviewKnowledge(ctx context.Context, userID, id string, mastery float64, wrongDelta int) (models.KnowledgePoint, error) {
	q := `UPDATE knowledge_points k SET mastery=$3,wrong_count=GREATEST(0,wrong_count+$4),review_count=review_count+1,review_status=CASE WHEN $3>=80 THEN 'mastered' WHEN $3>=1 THEN 'learning' ELSE 'unlearned' END,next_review_at=now()+CASE WHEN $3>=90 THEN interval '14 days' WHEN $3>=75 THEN interval '7 days' WHEN $3>=50 THEN interval '3 days' ELSE interval '1 day' END FROM subjects s JOIN goals g ON g.id=s.goal_id WHERE k.id=$2 AND k.subject_id=s.id AND g.user_id=$1 RETURNING k.id::text,k.subject_id::text,k.parent_id::text,k.name,k.importance,k.mastery::float8,k.review_status,k.next_review_at,k.review_count,k.wrong_count,k.created_at`
	y, err := scanKnowledge(s.db.QueryRow(ctx, q, userID, id, mastery, wrongDelta))
	if errors.Is(err, pgx.ErrNoRows) {
		return y, ErrNotFound
	}
	return y, err
}

func scanAssessment(row pgx.Row) (models.Assessment, error) {
	var x models.Assessment
	err := row.Scan(&x.ID, &x.GoalID, &x.SubjectID, &x.Name, &x.AssessmentType, &x.ExamDate, &x.Score, &x.FullScore, &x.DurationMinutes, &x.Details, &x.CreatedAt)
	return x, err
}

const assessmentSelect = `SELECT a.id::text,a.goal_id::text,a.subject_id::text,a.name,a.assessment_type,to_char(a.exam_date,'YYYY-MM-DD'),a.score::float8,a.full_score::float8,a.duration_minutes,a.details,a.created_at FROM assessments a`

func (s *Store) ListAssessments(ctx context.Context, userID, goalID string) ([]models.Assessment, error) {
	q := assessmentSelect + ` JOIN goals g ON g.id=a.goal_id WHERE g.user_id=$1`
	args := []any{userID}
	if goalID != "" {
		q += ` AND a.goal_id=$2`
		args = append(args, goalID)
	}
	q += ` ORDER BY a.exam_date DESC,a.created_at DESC`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Assessment{}
	for rows.Next() {
		x, err := scanAssessment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func (s *Store) CreateAssessment(ctx context.Context, userID string, x models.Assessment) (models.Assessment, error) {
	if len(x.Details) == 0 {
		x.Details = json.RawMessage(`{}`)
	}
	q := `INSERT INTO assessments(goal_id,subject_id,name,assessment_type,exam_date,score,full_score,duration_minutes,details) SELECT $2,$3,$4,$5,$6,$7,$8,$9,$10 FROM goals WHERE id=$2 AND user_id=$1
	AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM subjects WHERE id=$3 AND goal_id=$2))
	RETURNING id::text,goal_id::text,subject_id::text,name,assessment_type,to_char(exam_date,'YYYY-MM-DD'),score::float8,full_score::float8,duration_minutes,details,created_at`
	y, err := scanAssessment(s.db.QueryRow(ctx, q, userID, x.GoalID, x.SubjectID, x.Name, x.AssessmentType, x.ExamDate, x.Score, x.FullScore, x.DurationMinutes, x.Details))
	if errors.Is(err, pgx.ErrNoRows) {
		return y, ErrNotFound
	}
	return y, err
}

func (s *Store) UpdateAssessment(ctx context.Context, userID string, x models.Assessment) (models.Assessment, error) {
	if len(x.Details) == 0 {
		x.Details = json.RawMessage(`{}`)
	}
	q := `UPDATE assessments a SET subject_id=$3,name=$4,assessment_type=$5,exam_date=$6,score=$7,full_score=$8,duration_minutes=$9,details=$10 FROM goals g WHERE a.id=$2 AND a.goal_id=g.id AND g.user_id=$1
	AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM subjects WHERE id=$3 AND goal_id=a.goal_id))
	RETURNING a.id::text,a.goal_id::text,a.subject_id::text,a.name,a.assessment_type,to_char(a.exam_date,'YYYY-MM-DD'),a.score::float8,a.full_score::float8,a.duration_minutes,a.details,a.created_at`
	y, err := scanAssessment(s.db.QueryRow(ctx, q, userID, x.ID, x.SubjectID, x.Name, x.AssessmentType, x.ExamDate, x.Score, x.FullScore, x.DurationMinutes, x.Details))
	if errors.Is(err, pgx.ErrNoRows) {
		return y, ErrNotFound
	}
	return y, err
}

func (s *Store) DeleteAssessment(ctx context.Context, userID, id string) error {
	r, err := s.db.Exec(ctx, `DELETE FROM assessments a USING goals g WHERE a.id=$2 AND a.goal_id=g.id AND g.user_id=$1`, userID, id)
	if err != nil {
		return err
	}
	if r.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListReviews(ctx context.Context, userID, goalID string) ([]models.WeeklyReview, error) {
	q := `SELECT r.id::text,r.goal_id::text,to_char(r.week_start,'YYYY-MM-DD'),r.planned_minutes,r.actual_minutes,r.completion_rate::float8,r.summary,r.problems,r.next_week_focus,r.created_at,r.updated_at FROM weekly_reviews r JOIN goals g ON g.id=r.goal_id WHERE g.user_id=$1`
	args := []any{userID}
	if goalID != "" {
		q += ` AND r.goal_id=$2`
		args = append(args, goalID)
	}
	q += ` ORDER BY r.week_start DESC`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.WeeklyReview{}
	for rows.Next() {
		var x models.WeeklyReview
		if err := rows.Scan(&x.ID, &x.GoalID, &x.WeekStart, &x.PlannedMinutes, &x.ActualMinutes, &x.CompletionRate, &x.Summary, &x.Problems, &x.NextWeekFocus, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func (s *Store) UpsertReview(ctx context.Context, userID string, x models.WeeklyReview) (models.WeeklyReview, error) {
	var y models.WeeklyReview
	q := `INSERT INTO weekly_reviews(goal_id,week_start,planned_minutes,actual_minutes,completion_rate,summary,problems,next_week_focus) SELECT $2,$3,$4,$5,$6,$7,$8,$9 FROM goals WHERE id=$2 AND user_id=$1 ON CONFLICT(goal_id,week_start) DO UPDATE SET planned_minutes=excluded.planned_minutes,actual_minutes=excluded.actual_minutes,completion_rate=excluded.completion_rate,summary=excluded.summary,problems=excluded.problems,next_week_focus=excluded.next_week_focus RETURNING id::text,goal_id::text,to_char(week_start,'YYYY-MM-DD'),planned_minutes,actual_minutes,completion_rate::float8,summary,problems,next_week_focus,created_at,updated_at`
	err := s.db.QueryRow(ctx, q, userID, x.GoalID, x.WeekStart, x.PlannedMinutes, x.ActualMinutes, x.CompletionRate, x.Summary, x.Problems, x.NextWeekFocus).Scan(&y.ID, &y.GoalID, &y.WeekStart, &y.PlannedMinutes, &y.ActualMinutes, &y.CompletionRate, &y.Summary, &y.Problems, &y.NextWeekFocus, &y.CreatedAt, &y.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return y, ErrNotFound
	}
	return y, err
}

func (s *Store) Dashboard(ctx context.Context, userID, goalID string) (models.Dashboard, error) {
	var d models.Dashboard
	tz, now := s.userCalendar(ctx, userID)
	goals, err := s.ListGoals(ctx, userID)
	if err != nil {
		return d, err
	}
	d.Goals = goals
	today := now.Format("2006-01-02")
	tasks, err := s.ListTasks(ctx, userID, TaskFilter{Date: today, GoalID: goalID})
	if err != nil {
		return d, err
	}
	d.TodayTasks = tasks
	d.TodayTotal = len(tasks)
	for _, t := range tasks {
		if t.Status == "completed" {
			d.TodayCompleted++
		}
	}
	weekStart := now.AddDate(0, 0, -(int(now.Weekday())+6)%7).Format("2006-01-02")
	todayFilter, weekFilter := "", ""
	todayArgs := []any{userID, tz, today}
	weekArgs := []any{userID, tz, weekStart}
	if goalID != "" {
		todayFilter, weekFilter = ` AND goal_id=$4`, ` AND goal_id=$4`
		todayArgs = append(todayArgs, goalID)
		weekArgs = append(weekArgs, goalID)
	}
	_ = s.db.QueryRow(ctx, `SELECT COALESCE(SUM(duration_minutes),0)::int FROM study_sessions WHERE user_id=$1 AND (started_at AT TIME ZONE $2)::date=$3::date`+todayFilter, todayArgs...).Scan(&d.TodayMinutes)
	_ = s.db.QueryRow(ctx, `SELECT COALESCE(SUM(duration_minutes),0)::int FROM study_sessions WHERE user_id=$1 AND (started_at AT TIME ZONE $2)::date>=$3::date`+weekFilter, weekArgs...).Scan(&d.WeekMinutes)
	taskArgs := []any{userID, today}
	taskFilter := ""
	if goalID != "" {
		taskArgs = append(taskArgs, goalID)
		taskFilter = ` AND goal_id=$3`
	}
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM tasks WHERE user_id=$1 AND planned_date<$2::date AND status NOT IN ('completed','skipped')`+taskFilter, taskArgs...).Scan(&d.OverdueTasks)
	d.StreakDays = s.streak(ctx, userID, goalID, tz, now)
	d.SubjectProgress, _ = s.subjectProgress(ctx, userID, goalID)
	d.WeakKnowledge, _ = s.weakKnowledge(ctx, userID, goalID, 6)
	d.DailyStudyTrend, _ = s.studyTrend(ctx, userID, goalID, 14)
	assessments, _ := s.ListAssessments(ctx, userID, goalID)
	if len(assessments) > 5 {
		assessments = assessments[:5]
	}
	d.RecentAssessments = assessments
	return d, nil
}

func (s *Store) userCalendar(ctx context.Context, userID string) (string, time.Time) {
	tz := "Asia/Shanghai"
	_ = s.db.QueryRow(ctx, `SELECT timezone FROM users WHERE id=$1`, userID).Scan(&tz)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
		tz = "UTC"
	}
	return tz, time.Now().In(loc)
}

func (s *Store) streak(ctx context.Context, userID, goalID, tz string, now time.Time) int {
	q := `SELECT DISTINCT (started_at AT TIME ZONE $2)::date FROM study_sessions WHERE user_id=$1`
	args := []any{userID, tz}
	if goalID != "" {
		q += ` AND goal_id=$3`
		args = append(args, goalID)
	}
	q += ` ORDER BY 1 DESC LIMIT 366`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return 0
	}
	defer rows.Close()
	dates := map[string]bool{}
	for rows.Next() {
		var t time.Time
		if rows.Scan(&t) == nil {
			dates[t.Format("2006-01-02")] = true
		}
	}
	day := now
	if !dates[day.Format("2006-01-02")] {
		day = day.AddDate(0, 0, -1)
	}
	count := 0
	for dates[day.Format("2006-01-02")] {
		count++
		day = day.AddDate(0, 0, -1)
	}
	return count
}

func (s *Store) subjectProgress(ctx context.Context, userID, goalID string) ([]models.SubjectProgress, error) {
	q := `WITH task_data AS (
		SELECT subject_id,COALESCE(100.0*COUNT(*) FILTER(WHERE status='completed')/NULLIF(COUNT(*),0),0)::float8 progress
		FROM tasks WHERE user_id=$1`
	args := []any{userID}
	if goalID != "" {
		q += ` AND goal_id=$2`
		args = append(args, goalID)
	}
	q += ` GROUP BY subject_id), session_data AS (
		SELECT subject_id,COALESCE(SUM(duration_minutes),0)::int minutes
		FROM study_sessions WHERE user_id=$1`
	if goalID != "" {
		q += ` AND goal_id=$2`
	}
	q += ` GROUP BY subject_id)
		SELECT s.id::text,s.name,COALESCE(td.progress,0),COALESCE(sd.minutes,0),s.color
		FROM subjects s JOIN goals g ON g.id=s.goal_id
		LEFT JOIN task_data td ON td.subject_id=s.id
		LEFT JOIN session_data sd ON sd.subject_id=s.id
		WHERE g.user_id=$1`
	if goalID != "" {
		q += ` AND s.goal_id=$2`
	}
	q += ` ORDER BY s.name`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.SubjectProgress{}
	for rows.Next() {
		var x models.SubjectProgress
		if err := rows.Scan(&x.SubjectID, &x.SubjectName, &x.Progress, &x.Minutes, &x.Color); err != nil {
			return nil, err
		}
		x.Progress = math.Round(x.Progress*10) / 10
		items = append(items, x)
	}
	return items, rows.Err()
}

func (s *Store) weakKnowledge(ctx context.Context, userID, goalID string, limit int) ([]models.KnowledgePoint, error) {
	q := knowledgeSelect + ` JOIN subjects s ON s.id=k.subject_id JOIN goals g ON g.id=s.goal_id WHERE g.user_id=$1`
	args := []any{userID}
	if goalID != "" {
		q += ` AND g.id=$2`
		args = append(args, goalID)
	}
	q += ` ORDER BY k.mastery ASC,k.wrong_count DESC LIMIT ` + fmt.Sprint(limit)
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.KnowledgePoint{}
	for rows.Next() {
		x, err := scanKnowledge(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func (s *Store) studyTrend(ctx context.Context, userID, goalID string, days int) ([]models.DailyMetric, error) {
	tz, now := s.userCalendar(ctx, userID)
	q := `WITH dates AS (SELECT generate_series($3::date-($2::int-1),$3::date,interval '1 day')::date d), data AS (SELECT (started_at AT TIME ZONE $4)::date d,SUM(duration_minutes)::int minutes FROM study_sessions WHERE user_id=$1`
	args := []any{userID, days, now.Format("2006-01-02"), tz}
	if goalID != "" {
		q += ` AND goal_id=$5`
		args = append(args, goalID)
	}
	q += ` GROUP BY 1) SELECT to_char(dates.d,'MM-DD'),COALESCE(data.minutes,0)::int FROM dates LEFT JOIN data USING(d) ORDER BY dates.d`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.DailyMetric{}
	for rows.Next() {
		var x models.DailyMetric
		if err := rows.Scan(&x.Date, &x.Minutes); err != nil {
			return nil, err
		}
		x.Value = float64(x.Minutes)
		items = append(items, x)
	}
	return items, rows.Err()
}

func (s *Store) Analytics(ctx context.Context, userID, goalID string) (models.AnalyticsOverview, error) {
	var a models.AnalyticsOverview
	filter := ""
	args := []any{userID}
	if goalID != "" {
		filter = ` AND goal_id=$2`
		args = append(args, goalID)
	}
	_ = s.db.QueryRow(ctx, `SELECT COALESCE(SUM(duration_minutes),0)::int,COALESCE(AVG(NULLIF(focus_score,0)),0)::float8,COALESCE(AVG(NULLIF(mastery_score,0)),0)::float8,COALESCE(AVG(NULLIF(duration_minutes,0)),0)::float8 FROM study_sessions WHERE user_id=$1`+filter, args...).Scan(&a.TotalMinutes, &a.AverageFocusScore, &a.AverageMasteryScore, &a.AverageSessionMinutes)
	var total int
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*)::int,COUNT(*) FILTER(WHERE status='completed')::int FROM tasks WHERE user_id=$1`+filter, args...).Scan(&total, &a.CompletedTasks)
	if total > 0 {
		a.CompletionRate = math.Round(float64(a.CompletedTasks)*1000/float64(total)) / 10
	}
	tz, localNow := s.userCalendar(ctx, userID)
	overdueArgs := []any{userID, localNow.Format("2006-01-02")}
	overdueFilter := ""
	if goalID != "" {
		overdueArgs = append(overdueArgs, goalID)
		overdueFilter = ` AND goal_id=$3`
	}
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM tasks WHERE user_id=$1 AND planned_date<$2::date AND status NOT IN ('completed','skipped')`+overdueFilter, overdueArgs...).Scan(&a.OverdueTasks)
	dateFilter := ""
	dateArgs := []any{userID, tz}
	if goalID != "" {
		dateFilter = ` AND goal_id=$3`
		dateArgs = append(dateArgs, goalID)
	}
	_ = s.db.QueryRow(ctx, `SELECT COALESCE(to_char((started_at AT TIME ZONE $2)::date,'YYYY-MM-DD'),''),COALESCE(SUM(duration_minutes),0)::int FROM study_sessions WHERE user_id=$1`+dateFilter+` GROUP BY (started_at AT TIME ZONE $2)::date ORDER BY SUM(duration_minutes) DESC LIMIT 1`, dateArgs...).Scan(&a.BestStudyDate, &a.BestStudyMinutes)
	_ = s.db.QueryRow(ctx, `SELECT COALESCE(s.name,'') FROM study_sessions ss JOIN subjects s ON s.id=ss.subject_id WHERE ss.user_id=$1`+strings.ReplaceAll(filter, "goal_id", "ss.goal_id")+` AND ss.focus_score>0 GROUP BY s.id,s.name ORDER BY AVG(ss.focus_score) DESC,SUM(ss.duration_minutes) DESC LIMIT 1`, args...).Scan(&a.MostFocusedSubject)
	a.StudyTime, _ = s.studyTrend(ctx, userID, goalID, 30)
	a.SubjectProgress, _ = s.subjectProgress(ctx, userID, goalID)
	a.WeakKnowledge, _ = s.weakKnowledge(ctx, userID, goalID, 10)
	a.ScoreTrends, _ = s.ListAssessments(ctx, userID, goalID)
	if len(a.ScoreTrends) > 20 {
		a.ScoreTrends = a.ScoreTrends[:20]
	}
	a.TaskCompletion, _ = s.taskCompletionTrend(ctx, userID, goalID, 14)
	return a, nil
}

func (s *Store) taskCompletionTrend(ctx context.Context, userID, goalID string, days int) ([]models.DailyMetric, error) {
	_, now := s.userCalendar(ctx, userID)
	q := `WITH dates AS (SELECT generate_series($3::date-($2::int-1),$3::date,interval '1 day')::date d), data AS (SELECT planned_date d,COUNT(*) FILTER(WHERE status='completed')::int c FROM tasks WHERE user_id=$1`
	args := []any{userID, days, now.Format("2006-01-02")}
	if goalID != "" {
		q += ` AND goal_id=$4`
		args = append(args, goalID)
	}
	q += ` GROUP BY 1) SELECT to_char(dates.d,'MM-DD'),COALESCE(data.c,0)::int FROM dates LEFT JOIN data USING(d) ORDER BY dates.d`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.DailyMetric{}
	for rows.Next() {
		var x models.DailyMetric
		if err := rows.Scan(&x.Date, &x.Count); err != nil {
			return nil, err
		}
		x.Value = float64(x.Count)
		items = append(items, x)
	}
	return items, rows.Err()
}

func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }
