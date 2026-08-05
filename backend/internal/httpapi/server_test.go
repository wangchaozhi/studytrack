package httpapi

import (
	"testing"

	"studytrack/backend/internal/models"
)

func TestValidateGoal(t *testing.T) {
	valid := models.Goal{Name: "考研", Type: "postgraduate", StartDate: "2026-08-05", ExamDate: "2026-12-26"}
	if err := validateGoal(valid); err != nil {
		t.Fatalf("valid goal rejected: %v", err)
	}
	cases := []models.Goal{
		{Type: "postgraduate", StartDate: "2026-08-05", ExamDate: "2026-12-26"},
		{Name: "目标", Type: "unknown", StartDate: "2026-08-05", ExamDate: "2026-12-26"},
		{Name: "目标", Type: "ielts", StartDate: "2026-12-26", ExamDate: "2026-08-05"},
	}
	for _, item := range cases {
		if validateGoal(item) == nil {
			t.Fatalf("invalid goal accepted: %+v", item)
		}
	}
}

func TestNormalizeAndValidateTask(t *testing.T) {
	task := models.Task{GoalID: "goal", Title: "复习概率论", PlannedDate: "2026-08-05"}
	normalizeTask(&task)
	if task.TaskType != "study" || task.Priority != "medium" || task.Status != "pending" || task.EstimatedMinutes != 30 {
		t.Fatalf("unexpected defaults: %+v", task)
	}
	if err := validateTask(task); err != nil {
		t.Fatalf("valid task rejected: %v", err)
	}
	task.PlannedDate = "invalid"
	if validateTask(task) == nil {
		t.Fatal("invalid date accepted")
	}
}

func TestNormalizeAndValidateAssessment(t *testing.T) {
	item := models.Assessment{GoalID: "goal", Name: "模拟卷", ExamDate: "2026-08-05"}
	normalizeAssessment(&item)
	if item.AssessmentType != "practice" || item.FullScore != 100 {
		t.Fatalf("unexpected defaults: %+v", item)
	}
	if err := validateAssessment(item); err != nil {
		t.Fatalf("valid assessment rejected: %v", err)
	}
}
