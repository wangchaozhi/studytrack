CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    nickname TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS goals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('degree', 'postgraduate', 'ielts')),
    start_date DATE NOT NULL,
    exam_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed', 'archived')),
    target_score NUMERIC(8,2) NOT NULL DEFAULT 0,
    progress NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_goals_user ON goals(user_id);

CREATE TABLE IF NOT EXISTS phases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'completed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_phases_goal ON phases(goal_id, sort_order);

CREATE TABLE IF NOT EXISTS subjects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'general',
    target_score NUMERIC(8,2) NOT NULL DEFAULT 0,
    current_score NUMERIC(8,2) NOT NULL DEFAULT 0,
    planned_minutes INT NOT NULL DEFAULT 0,
    color TEXT NOT NULL DEFAULT '#4f46e5',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_subjects_goal ON subjects(goal_id);

CREATE TABLE IF NOT EXISTS knowledge_points (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES knowledge_points(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    importance TEXT NOT NULL DEFAULT 'normal' CHECK (importance IN ('normal', 'important', 'core')),
    mastery NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (mastery >= 0 AND mastery <= 100),
    review_status TEXT NOT NULL DEFAULT 'unlearned' CHECK (review_status IN ('unlearned', 'learning', 'mastered', 'review')),
    next_review_at TIMESTAMPTZ,
    review_count INT NOT NULL DEFAULT 0,
    wrong_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_knowledge_subject ON knowledge_points(subject_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_review ON knowledge_points(next_review_at);

CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    phase_id UUID REFERENCES phases(id) ON DELETE SET NULL,
    subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL,
    knowledge_point_id UUID REFERENCES knowledge_points(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    task_type TEXT NOT NULL DEFAULT 'study' CHECK (task_type IN ('study', 'practice', 'review', 'mock', 'vocabulary', 'other')),
    priority TEXT NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'overdue', 'skipped')),
    planned_date DATE NOT NULL,
    estimated_minutes INT NOT NULL DEFAULT 30 CHECK (estimated_minutes >= 0),
    actual_minutes INT NOT NULL DEFAULT 0 CHECK (actual_minutes >= 0),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_tasks_user_date ON tasks(user_id, planned_date);
CREATE INDEX IF NOT EXISTS idx_tasks_goal ON tasks(goal_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

CREATE TABLE IF NOT EXISTS study_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    task_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ,
    duration_minutes INT NOT NULL DEFAULT 0,
    focus_score INT NOT NULL DEFAULT 0 CHECK (focus_score BETWEEN 0 AND 5),
    mastery_score INT NOT NULL DEFAULT 0 CHECK (mastery_score BETWEEN 0 AND 5),
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_start ON study_sessions(user_id, started_at DESC);

CREATE TABLE IF NOT EXISTS assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    assessment_type TEXT NOT NULL DEFAULT 'practice' CHECK (assessment_type IN ('practice', 'mock', 'exam', 'ielts')),
    exam_date DATE NOT NULL,
    score NUMERIC(8,2) NOT NULL DEFAULT 0,
    full_score NUMERIC(8,2) NOT NULL DEFAULT 100,
    duration_minutes INT NOT NULL DEFAULT 0,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_assessments_goal_date ON assessments(goal_id, exam_date DESC);

CREATE TABLE IF NOT EXISTS weekly_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    week_start DATE NOT NULL,
    planned_minutes INT NOT NULL DEFAULT 0,
    actual_minutes INT NOT NULL DEFAULT 0,
    completion_rate NUMERIC(5,2) NOT NULL DEFAULT 0,
    summary TEXT NOT NULL DEFAULT '',
    problems TEXT NOT NULL DEFAULT '',
    next_week_focus TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(goal_id, week_start)
);

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS users_updated_at ON users;
CREATE TRIGGER users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();
DROP TRIGGER IF EXISTS goals_updated_at ON goals;
CREATE TRIGGER goals_updated_at BEFORE UPDATE ON goals FOR EACH ROW EXECUTE FUNCTION set_updated_at();
DROP TRIGGER IF EXISTS tasks_updated_at ON tasks;
CREATE TRIGGER tasks_updated_at BEFORE UPDATE ON tasks FOR EACH ROW EXECUTE FUNCTION set_updated_at();
DROP TRIGGER IF EXISTS reviews_updated_at ON weekly_reviews;
CREATE TRIGGER reviews_updated_at BEFORE UPDATE ON weekly_reviews FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE OR REPLACE FUNCTION recalculate_goal_progress() RETURNS TRIGGER AS $$
DECLARE
  current_goal UUID;
  previous_goal UUID;
BEGIN
  IF TG_OP = 'DELETE' THEN
    current_goal := OLD.goal_id;
  ELSE
    current_goal := NEW.goal_id;
  END IF;

  UPDATE goals
  SET progress = COALESCE((
    SELECT ROUND(100.0 * COUNT(*) FILTER (WHERE status = 'completed') / NULLIF(COUNT(*), 0), 2)
    FROM tasks
    WHERE goal_id = current_goal
  ), 0)
  WHERE id = current_goal;

  IF TG_OP = 'UPDATE' AND OLD.goal_id IS DISTINCT FROM NEW.goal_id THEN
    previous_goal := OLD.goal_id;
    UPDATE goals
    SET progress = COALESCE((
      SELECT ROUND(100.0 * COUNT(*) FILTER (WHERE status = 'completed') / NULLIF(COUNT(*), 0), 2)
      FROM tasks
      WHERE goal_id = previous_goal
    ), 0)
    WHERE id = previous_goal;
  END IF;

  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS tasks_goal_progress ON tasks;
CREATE TRIGGER tasks_goal_progress
AFTER INSERT OR UPDATE OF status, goal_id OR DELETE ON tasks
FOR EACH ROW EXECUTE FUNCTION recalculate_goal_progress();
