export type GoalType = 'degree' | 'postgraduate' | 'ielts'
export type TaskStatus = 'pending' | 'in_progress' | 'completed' | 'overdue' | 'skipped'

export interface User {
  id: string
  email: string
  nickname: string
  timezone: string
  created_at: string
  updated_at: string
}

export interface Goal {
  id: string
  name: string
  type: GoalType
  start_date: string
  exam_date: string
  status: 'active' | 'paused' | 'completed' | 'archived'
  target_score: number
  progress: number
  settings: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface Phase {
  id: string
  goal_id: string
  name: string
  start_date: string
  end_date: string
  sort_order: number
  status: 'pending' | 'active' | 'completed'
}

export interface Subject {
  id: string
  goal_id: string
  name: string
  category: string
  target_score: number
  current_score: number
  planned_minutes: number
  color: string
}

export interface Task {
  id: string
  goal_id: string
  phase_id?: string | null
  subject_id?: string | null
  knowledge_point_id?: string | null
  title: string
  description: string
  task_type: string
  priority: 'low' | 'medium' | 'high'
  status: TaskStatus
  planned_date: string
  estimated_minutes: number
  actual_minutes: number
  completed_at?: string | null
  goal_name?: string
  subject_name?: string
}

export interface KnowledgePoint {
  id: string
  subject_id: string
  parent_id?: string | null
  name: string
  importance: 'normal' | 'important' | 'core'
  mastery: number
  review_status: 'unlearned' | 'learning' | 'mastered' | 'review'
  next_review_at?: string | null
  review_count: number
  wrong_count: number
}

export interface StudySession {
  id: string
  goal_id: string
  task_id?: string | null
  subject_id?: string | null
  started_at: string
  ended_at?: string | null
  duration_minutes: number
  focus_score: number
  mastery_score: number
  notes: string
}

export interface Assessment {
  id: string
  goal_id: string
  subject_id?: string | null
  name: string
  assessment_type: 'practice' | 'mock' | 'exam' | 'ielts'
  exam_date: string
  score: number
  full_score: number
  duration_minutes: number
  details: Record<string, unknown>
}

export interface WeeklyReview {
  id: string
  goal_id: string
  week_start: string
  planned_minutes: number
  actual_minutes: number
  completion_rate: number
  summary: string
  problems: string
  next_week_focus: string
  created_at?: string
  updated_at?: string
}

export interface DailyMetric {
  date: string
  value: number
  minutes?: number
  count?: number
}

export interface SubjectProgress {
  subject_id: string
  subject_name: string
  progress: number
  minutes: number
  color: string
}

export interface DashboardData {
  today_minutes: number
  week_minutes: number
  today_completed: number
  today_total: number
  streak_days: number
  overdue_tasks: number
  goals: Goal[]
  today_tasks: Task[]
  subject_progress: SubjectProgress[]
  weak_knowledge: KnowledgePoint[]
  daily_study_trend: DailyMetric[]
  recent_assessments: Assessment[]
}

export interface AnalyticsOverview {
  total_minutes: number
  completed_tasks: number
  completion_rate: number
  average_focus_score: number
  average_mastery_score: number
  average_session_minutes: number
  overdue_tasks: number
  best_study_date: string
  best_study_minutes: number
  most_focused_subject: string
  study_time: DailyMetric[]
  task_completion: DailyMetric[]
  subject_progress: SubjectProgress[]
  score_trends: Assessment[]
  weak_knowledge: KnowledgePoint[]
}
