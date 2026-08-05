package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"studytrack/backend/internal/auth"
	"studytrack/backend/internal/models"
	"studytrack/backend/internal/store"
)

type contextKey string

const userIDKey contextKey = "userID"

type Server struct {
	store   *store.Store
	auth    *auth.Service
	logger  *slog.Logger
	origins map[string]bool
	mux     *http.ServeMux
}

func New(dataStore *store.Store, authService *auth.Service, logger *slog.Logger, origins []string) *Server {
	s := &Server{store: dataStore, auth: authService, logger: logger, origins: map[string]bool{}, mux: http.NewServeMux()}
	for _, origin := range origins {
		s.origins[origin] = true
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.recoverer(s.cors(s.logging(s.mux)))
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now()})
	})
	s.mux.HandleFunc("POST /api/v1/auth/register", s.register)
	s.mux.HandleFunc("POST /api/v1/auth/login", s.login)
	s.mux.Handle("POST /api/v1/auth/refresh", s.requireAuth(http.HandlerFunc(s.refresh)))
	s.mux.Handle("GET /api/v1/users/me", s.requireAuth(http.HandlerFunc(s.me)))
	s.mux.Handle("PUT /api/v1/users/me", s.requireAuth(http.HandlerFunc(s.updateMe)))

	s.mux.Handle("GET /api/v1/dashboard", s.requireAuth(http.HandlerFunc(s.dashboard)))
	s.mux.Handle("GET /api/v1/goals", s.requireAuth(http.HandlerFunc(s.listGoals)))
	s.mux.Handle("POST /api/v1/goals", s.requireAuth(http.HandlerFunc(s.createGoal)))
	s.mux.Handle("GET /api/v1/goals/{id}", s.requireAuth(http.HandlerFunc(s.getGoal)))
	s.mux.Handle("PUT /api/v1/goals/{id}", s.requireAuth(http.HandlerFunc(s.updateGoal)))
	s.mux.Handle("DELETE /api/v1/goals/{id}", s.requireAuth(http.HandlerFunc(s.deleteGoal)))
	s.mux.Handle("GET /api/v1/goals/{id}/dashboard", s.requireAuth(http.HandlerFunc(s.goalDashboard)))
	s.mux.Handle("GET /api/v1/goals/{id}/phases", s.requireAuth(http.HandlerFunc(s.listPhases)))
	s.mux.Handle("POST /api/v1/goals/{id}/phases", s.requireAuth(http.HandlerFunc(s.createPhase)))
	s.mux.Handle("GET /api/v1/goals/{id}/subjects", s.requireAuth(http.HandlerFunc(s.listSubjects)))
	s.mux.Handle("POST /api/v1/goals/{id}/subjects", s.requireAuth(http.HandlerFunc(s.createSubject)))
	s.mux.Handle("PUT /api/v1/subjects/{id}", s.requireAuth(http.HandlerFunc(s.updateSubject)))

	s.mux.Handle("GET /api/v1/tasks", s.requireAuth(http.HandlerFunc(s.listTasks)))
	s.mux.Handle("POST /api/v1/tasks", s.requireAuth(http.HandlerFunc(s.createTask)))
	s.mux.Handle("PUT /api/v1/tasks/{id}", s.requireAuth(http.HandlerFunc(s.updateTask)))
	s.mux.Handle("PATCH /api/v1/tasks/{id}/status", s.requireAuth(http.HandlerFunc(s.updateTaskStatus)))
	s.mux.Handle("DELETE /api/v1/tasks/{id}", s.requireAuth(http.HandlerFunc(s.deleteTask)))

	s.mux.Handle("POST /api/v1/study-sessions/start", s.requireAuth(http.HandlerFunc(s.startSession)))
	s.mux.Handle("POST /api/v1/study-sessions/{id}/finish", s.requireAuth(http.HandlerFunc(s.finishSession)))
	s.mux.Handle("GET /api/v1/study-sessions", s.requireAuth(http.HandlerFunc(s.listSessions)))

	s.mux.Handle("GET /api/v1/subjects/{id}/knowledge-points", s.requireAuth(http.HandlerFunc(s.listKnowledge)))
	s.mux.Handle("POST /api/v1/knowledge-points", s.requireAuth(http.HandlerFunc(s.createKnowledge)))
	s.mux.Handle("PUT /api/v1/knowledge-points/{id}", s.requireAuth(http.HandlerFunc(s.updateKnowledge)))
	s.mux.Handle("POST /api/v1/knowledge-points/{id}/review", s.requireAuth(http.HandlerFunc(s.reviewKnowledge)))

	s.mux.Handle("GET /api/v1/assessments", s.requireAuth(http.HandlerFunc(s.listAssessments)))
	s.mux.Handle("GET /api/v1/goals/{id}/assessments", s.requireAuth(http.HandlerFunc(s.listGoalAssessments)))
	s.mux.Handle("POST /api/v1/assessments", s.requireAuth(http.HandlerFunc(s.createAssessment)))
	s.mux.Handle("PUT /api/v1/assessments/{id}", s.requireAuth(http.HandlerFunc(s.updateAssessment)))
	s.mux.Handle("DELETE /api/v1/assessments/{id}", s.requireAuth(http.HandlerFunc(s.deleteAssessment)))

	s.mux.Handle("GET /api/v1/weekly-reviews", s.requireAuth(http.HandlerFunc(s.listReviews)))
	s.mux.Handle("GET /api/v1/goals/{id}/reviews", s.requireAuth(http.HandlerFunc(s.listGoalReviews)))
	s.mux.Handle("POST /api/v1/weekly-reviews", s.requireAuth(http.HandlerFunc(s.upsertReview)))
	s.mux.Handle("GET /api/v1/analytics/overview", s.requireAuth(http.HandlerFunc(s.analytics)))
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var body struct{ Email, Password, Nickname, Timezone string }
	if !decodeJSON(w, r, &body) {
		return
	}
	body.Email = store.NormalizeEmail(body.Email)
	body.Nickname = strings.TrimSpace(body.Nickname)
	if body.Email == "" || !strings.Contains(body.Email, "@") {
		writeError(w, http.StatusBadRequest, "请输入有效邮箱")
		return
	}
	if body.Nickname == "" {
		body.Nickname = strings.Split(body.Email, "@")[0]
	}
	if body.Timezone == "" {
		body.Timezone = "Asia/Shanghai"
	}
	if _, err := time.LoadLocation(body.Timezone); err != nil {
		writeError(w, 400, "时区无效")
		return
	}
	hash, err := s.auth.HashPassword(body.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	u, err := s.store.CreateUser(r.Context(), body.Email, hash, body.Nickname, body.Timezone)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "邮箱已注册")
			return
		}
		s.internal(w, err)
		return
	}
	token, err := s.auth.IssueToken(u.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": token, "user": u})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct{ Email, Password string }
	if !decodeJSON(w, r, &body) {
		return
	}
	u, hash, err := s.store.GetUserByEmail(r.Context(), body.Email)
	if err != nil || s.auth.ComparePassword(hash, body.Password) != nil {
		writeError(w, http.StatusUnauthorized, "邮箱或密码错误")
		return
	}
	token, err := s.auth.IssueToken(u.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": u})
}
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	token, err := s.auth.IssueToken(userID(r))
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, err := s.store.GetUser(r.Context(), userID(r))
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}
func (s *Server) updateMe(w http.ResponseWriter, r *http.Request) {
	var body struct{ Nickname, Timezone string }
	if !decodeJSON(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Nickname) == "" {
		writeError(w, 400, "昵称不能为空")
		return
	}
	if body.Timezone == "" {
		body.Timezone = "Asia/Shanghai"
	}
	if _, err := time.LoadLocation(body.Timezone); err != nil {
		writeError(w, 400, "时区无效")
		return
	}
	u, err := s.store.UpdateUser(r.Context(), userID(r), body.Nickname, body.Timezone)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 200, u)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	d, err := s.store.Dashboard(r.Context(), userID(r), r.URL.Query().Get("goal_id"))
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 200, d)
}
func (s *Server) goalDashboard(w http.ResponseWriter, r *http.Request) {
	d, err := s.store.Dashboard(r.Context(), userID(r), r.PathValue("id"))
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 200, d)
}
func (s *Server) listGoals(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListGoals(r.Context(), userID(r))
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 200, items)
}
func (s *Server) createGoal(w http.ResponseWriter, r *http.Request) {
	var x models.Goal
	if !decodeJSON(w, r, &x) {
		return
	}
	if err := validateGoal(x); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	y, err := s.store.CreateGoal(r.Context(), userID(r), x)
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 201, y)
}
func (s *Server) getGoal(w http.ResponseWriter, r *http.Request) {
	x, err := s.store.GetGoal(r.Context(), userID(r), r.PathValue("id"))
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 200, x)
}
func (s *Server) updateGoal(w http.ResponseWriter, r *http.Request) {
	var x models.Goal
	if !decodeJSON(w, r, &x) {
		return
	}
	x.ID = r.PathValue("id")
	if err := validateGoal(x); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	y, err := s.store.UpdateGoal(r.Context(), userID(r), x)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 200, y)
}
func (s *Server) deleteGoal(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteGoal(r.Context(), userID(r), r.PathValue("id")); err != nil {
		s.handleStoreError(w, err)
		return
	}
	w.WriteHeader(204)
}
func validateGoal(x models.Goal) error {
	if strings.TrimSpace(x.Name) == "" {
		return errors.New("目标名称不能为空")
	}
	if x.Type != "degree" && x.Type != "postgraduate" && x.Type != "ielts" {
		return errors.New("目标类型无效")
	}
	start, err := time.Parse("2006-01-02", x.StartDate)
	if err != nil {
		return errors.New("开始日期无效")
	}
	exam, err := time.Parse("2006-01-02", x.ExamDate)
	if err != nil {
		return errors.New("考试日期无效")
	}
	if exam.Before(start) {
		return errors.New("考试日期不能早于开始日期")
	}
	return nil
}

func (s *Server) listPhases(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListPhases(r.Context(), userID(r), r.PathValue("id"))
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 200, items)
}
func (s *Server) createPhase(w http.ResponseWriter, r *http.Request) {
	var x models.Phase
	if !decodeJSON(w, r, &x) {
		return
	}
	x.GoalID = r.PathValue("id")
	if x.Status == "" {
		x.Status = "pending"
	}
	if strings.TrimSpace(x.Name) == "" {
		writeError(w, 400, "阶段名称不能为空")
		return
	}
	start, startErr := time.Parse("2006-01-02", x.StartDate)
	end, endErr := time.Parse("2006-01-02", x.EndDate)
	if startErr != nil || endErr != nil || end.Before(start) {
		writeError(w, 400, "阶段日期无效")
		return
	}
	if !map[string]bool{"pending": true, "active": true, "completed": true}[x.Status] {
		writeError(w, 400, "阶段状态无效")
		return
	}
	y, err := s.store.CreatePhase(r.Context(), userID(r), x)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 201, y)
}
func (s *Server) listSubjects(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListSubjects(r.Context(), userID(r), r.PathValue("id"))
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 200, items)
}
func (s *Server) createSubject(w http.ResponseWriter, r *http.Request) {
	var x models.Subject
	if !decodeJSON(w, r, &x) {
		return
	}
	x.GoalID = r.PathValue("id")
	if x.Color == "" {
		x.Color = "#4f46e5"
	}
	if strings.TrimSpace(x.Name) == "" {
		writeError(w, 400, "科目名称不能为空")
		return
	}
	y, err := s.store.CreateSubject(r.Context(), userID(r), x)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 201, y)
}
func (s *Server) updateSubject(w http.ResponseWriter, r *http.Request) {
	var x models.Subject
	if !decodeJSON(w, r, &x) {
		return
	}
	x.ID = r.PathValue("id")
	if x.Color == "" {
		x.Color = "#4f46e5"
	}
	if strings.TrimSpace(x.Name) == "" {
		writeError(w, 400, "科目名称不能为空")
		return
	}
	y, err := s.store.UpdateSubject(r.Context(), userID(r), x)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 200, y)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := s.store.ListTasks(r.Context(), userID(r), store.TaskFilter{Date: q.Get("date"), GoalID: q.Get("goal_id"), Status: q.Get("status"), From: q.Get("from"), To: q.Get("to")})
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 200, items)
}
func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var x models.Task
	if !decodeJSON(w, r, &x) {
		return
	}
	normalizeTask(&x)
	if err := validateTask(x); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	y, err := s.store.CreateTask(r.Context(), userID(r), x)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 201, y)
}
func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	var x models.Task
	if !decodeJSON(w, r, &x) {
		return
	}
	x.ID = r.PathValue("id")
	normalizeTask(&x)
	if err := validateTask(x); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	y, err := s.store.UpdateTask(r.Context(), userID(r), x)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 200, y)
}
func normalizeTask(x *models.Task) {
	if x.TaskType == "" {
		x.TaskType = "study"
	}
	if x.Priority == "" {
		x.Priority = "medium"
	}
	if x.Status == "" {
		x.Status = "pending"
	}
	if x.EstimatedMinutes <= 0 {
		x.EstimatedMinutes = 30
	}
}
func validateTask(x models.Task) error {
	if x.GoalID == "" {
		return errors.New("请选择学习目标")
	}
	if strings.TrimSpace(x.Title) == "" {
		return errors.New("任务名称不能为空")
	}
	if _, err := time.Parse("2006-01-02", x.PlannedDate); err != nil {
		return errors.New("计划日期无效")
	}
	if !map[string]bool{"study": true, "practice": true, "review": true, "mock": true, "vocabulary": true, "other": true}[x.TaskType] {
		return errors.New("任务类型无效")
	}
	if !map[string]bool{"low": true, "medium": true, "high": true}[x.Priority] {
		return errors.New("任务优先级无效")
	}
	if !map[string]bool{"pending": true, "in_progress": true, "completed": true, "overdue": true, "skipped": true}[x.Status] {
		return errors.New("任务状态无效")
	}
	if x.EstimatedMinutes < 0 || x.ActualMinutes < 0 {
		return errors.New("任务时长不能为负数")
	}
	return nil
}
func (s *Server) updateTaskStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	valid := map[string]bool{"pending": true, "in_progress": true, "completed": true, "overdue": true, "skipped": true}
	if !valid[body.Status] {
		writeError(w, 400, "任务状态无效")
		return
	}
	x, err := s.store.UpdateTaskStatus(r.Context(), userID(r), r.PathValue("id"), body.Status)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 200, x)
}
func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteTask(r.Context(), userID(r), r.PathValue("id")); err != nil {
		s.handleStoreError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request) {
	var x models.StudySession
	if !decodeJSON(w, r, &x) {
		return
	}
	if x.GoalID == "" {
		writeError(w, 400, "请选择学习目标")
		return
	}
	y, err := s.store.StartSession(r.Context(), userID(r), x)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 201, y)
}
func (s *Server) finishSession(w http.ResponseWriter, r *http.Request) {
	var x models.StudySession
	if !decodeJSON(w, r, &x) {
		return
	}
	x.ID = r.PathValue("id")
	if x.FocusScore < 1 || x.FocusScore > 5 || x.MasteryScore < 1 || x.MasteryScore > 5 {
		writeError(w, 400, "专注度和掌握度需为 1-5")
		return
	}
	y, err := s.store.FinishSession(r.Context(), userID(r), x)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 200, y)
}
func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.store.ListSessions(r.Context(), userID(r), r.URL.Query().Get("goal_id"), limit)
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 200, items)
}

func (s *Server) listKnowledge(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListKnowledge(r.Context(), userID(r), r.PathValue("id"))
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 200, items)
}
func (s *Server) createKnowledge(w http.ResponseWriter, r *http.Request) {
	var x models.KnowledgePoint
	if !decodeJSON(w, r, &x) {
		return
	}
	if x.Name == "" || x.SubjectID == "" {
		writeError(w, 400, "科目和知识点名称不能为空")
		return
	}
	if x.Importance == "" {
		x.Importance = "normal"
	}
	if x.ReviewStatus == "" {
		x.ReviewStatus = "unlearned"
	}
	if err := validateKnowledge(x); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	y, err := s.store.CreateKnowledge(r.Context(), userID(r), x)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 201, y)
}
func (s *Server) updateKnowledge(w http.ResponseWriter, r *http.Request) {
	var x models.KnowledgePoint
	if !decodeJSON(w, r, &x) {
		return
	}
	x.ID = r.PathValue("id")
	if err := validateKnowledge(x); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	y, err := s.store.UpdateKnowledge(r.Context(), userID(r), x)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 200, y)
}
func validateKnowledge(x models.KnowledgePoint) error {
	if strings.TrimSpace(x.Name) == "" || x.SubjectID == "" {
		return errors.New("科目和知识点名称不能为空")
	}
	if x.Mastery < 0 || x.Mastery > 100 || x.WrongCount < 0 || x.ReviewCount < 0 {
		return errors.New("知识点数值无效")
	}
	if !map[string]bool{"normal": true, "important": true, "core": true}[x.Importance] {
		return errors.New("重要程度无效")
	}
	if !map[string]bool{"unlearned": true, "learning": true, "mastered": true, "review": true}[x.ReviewStatus] {
		return errors.New("复习状态无效")
	}
	return nil
}
func (s *Server) reviewKnowledge(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Mastery    float64 `json:"mastery"`
		WrongDelta int     `json:"wrong_delta"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Mastery < 0 || body.Mastery > 100 {
		writeError(w, 400, "掌握度需为 0-100")
		return
	}
	y, err := s.store.ReviewKnowledge(r.Context(), userID(r), r.PathValue("id"), body.Mastery, body.WrongDelta)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 200, y)
}

func (s *Server) listAssessments(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListAssessments(r.Context(), userID(r), r.URL.Query().Get("goal_id"))
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 200, items)
}
func (s *Server) listGoalAssessments(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListAssessments(r.Context(), userID(r), r.PathValue("id"))
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 200, items)
}
func (s *Server) createAssessment(w http.ResponseWriter, r *http.Request) {
	var x models.Assessment
	if !decodeJSON(w, r, &x) {
		return
	}
	normalizeAssessment(&x)
	if err := validateAssessment(x); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	y, err := s.store.CreateAssessment(r.Context(), userID(r), x)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 201, y)
}
func (s *Server) updateAssessment(w http.ResponseWriter, r *http.Request) {
	var x models.Assessment
	if !decodeJSON(w, r, &x) {
		return
	}
	x.ID = r.PathValue("id")
	normalizeAssessment(&x)
	if err := validateAssessment(x); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	y, err := s.store.UpdateAssessment(r.Context(), userID(r), x)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 200, y)
}
func normalizeAssessment(x *models.Assessment) {
	if x.AssessmentType == "" {
		x.AssessmentType = "practice"
	}
	if x.FullScore <= 0 {
		x.FullScore = 100
	}
	if len(x.Details) == 0 {
		x.Details = json.RawMessage(`{}`)
	}
}
func validateAssessment(x models.Assessment) error {
	if x.GoalID == "" || strings.TrimSpace(x.Name) == "" {
		return errors.New("目标和考试名称不能为空")
	}
	if _, err := time.Parse("2006-01-02", x.ExamDate); err != nil {
		return errors.New("考试日期无效")
	}
	if !map[string]bool{"practice": true, "mock": true, "exam": true, "ielts": true}[x.AssessmentType] {
		return errors.New("考试类型无效")
	}
	if x.Score < 0 || x.FullScore <= 0 || x.DurationMinutes < 0 {
		return errors.New("成绩或用时无效")
	}
	return nil
}
func (s *Server) deleteAssessment(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteAssessment(r.Context(), userID(r), r.PathValue("id")); err != nil {
		s.handleStoreError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (s *Server) listReviews(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListReviews(r.Context(), userID(r), r.URL.Query().Get("goal_id"))
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 200, items)
}
func (s *Server) listGoalReviews(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListReviews(r.Context(), userID(r), r.PathValue("id"))
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 200, items)
}
func (s *Server) upsertReview(w http.ResponseWriter, r *http.Request) {
	var x models.WeeklyReview
	if !decodeJSON(w, r, &x) {
		return
	}
	if x.GoalID == "" || x.WeekStart == "" {
		writeError(w, 400, "目标和周开始日期不能为空")
		return
	}
	if _, err := time.Parse("2006-01-02", x.WeekStart); err != nil || x.PlannedMinutes < 0 || x.ActualMinutes < 0 || x.CompletionRate < 0 || x.CompletionRate > 100 {
		writeError(w, 400, "复盘日期或统计值无效")
		return
	}
	y, err := s.store.UpsertReview(r.Context(), userID(r), x)
	if err != nil {
		s.handleStoreError(w, err)
		return
	}
	writeJSON(w, 200, y)
}
func (s *Server) analytics(w http.ResponseWriter, r *http.Request) {
	x, err := s.store.Analytics(r.Context(), userID(r), r.URL.Query().Get("goal_id"))
	if err != nil {
		s.internal(w, err)
		return
	}
	writeJSON(w, 200, x)
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, 401, "请先登录")
			return
		}
		claims, err := s.auth.ParseToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			writeError(w, 401, "登录已过期，请重新登录")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, claims.UserID)))
	})
}
func userID(r *http.Request) string { v, _ := r.Context().Value(userIDKey).(string); return v }

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		writeError(w, 400, "请求数据格式错误: "+err.Error())
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func (s *Server) handleStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, 404, "数据不存在")
		return
	}
	s.internal(w, err)
}
func (s *Server) internal(w http.ResponseWriter, err error) {
	s.logger.Error("request failed", "error", err)
	writeError(w, 500, "服务器处理失败")
}
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (s.origins[origin] || s.origins["*"]) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String())
	})
}
func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Error("panic recovered", "error", fmt.Sprint(rec))
				writeError(w, 500, "服务器处理失败")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
