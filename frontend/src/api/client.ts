import type {
  AnalyticsOverview,
  Assessment,
  DashboardData,
  Goal,
  KnowledgePoint,
  Phase,
  StudySession,
  Subject,
  Task,
  User,
  WeeklyReview,
} from '../types'

const API_URL = import.meta.env.VITE_API_URL || '/api/v1'

export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('studytrack_token')
  const response = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  })

  if (response.status === 204) return undefined as T
  const data = await response.json().catch(() => ({}))
  if (!response.ok) {
    if (response.status === 401) localStorage.removeItem('studytrack_token')
    throw new ApiError(data.error || '请求失败', response.status)
  }
  return data as T
}

const json = (body: unknown): RequestInit => ({ body: JSON.stringify(body) })

export const api = {
  register: (body: { email: string; password: string; nickname: string; timezone?: string }) =>
    request<{ token: string; user: User }>('/auth/register', { method: 'POST', ...json(body) }),
  login: (body: { email: string; password: string }) =>
    request<{ token: string; user: User }>('/auth/login', { method: 'POST', ...json(body) }),
  me: () => request<User>('/users/me'),
  updateMe: (body: Pick<User, 'nickname' | 'timezone'>) => request<User>('/users/me', { method: 'PUT', ...json(body) }),

  dashboard: (goalId?: string) => request<DashboardData>(`/dashboard${goalId ? `?goal_id=${goalId}` : ''}`),
  goals: () => request<Goal[]>('/goals'),
  goal: (id: string) => request<Goal>(`/goals/${id}`),
  createGoal: (body: Partial<Goal>) => request<Goal>('/goals', { method: 'POST', ...json(body) }),
  updateGoal: (id: string, body: Partial<Goal>) => request<Goal>(`/goals/${id}`, { method: 'PUT', ...json(body) }),
  deleteGoal: (id: string) => request<void>(`/goals/${id}`, { method: 'DELETE' }),

  phases: (goalId: string) => request<Phase[]>(`/goals/${goalId}/phases`),
  createPhase: (goalId: string, body: Partial<Phase>) => request<Phase>(`/goals/${goalId}/phases`, { method: 'POST', ...json(body) }),
  subjects: (goalId: string) => request<Subject[]>(`/goals/${goalId}/subjects`),
  createSubject: (goalId: string, body: Partial<Subject>) => request<Subject>(`/goals/${goalId}/subjects`, { method: 'POST', ...json(body) }),
  updateSubject: (id: string, body: Partial<Subject>) => request<Subject>(`/subjects/${id}`, { method: 'PUT', ...json(body) }),

  tasks: (params: Record<string, string | undefined> = {}) => {
    const query = new URLSearchParams(Object.entries(params).filter(([, v]) => Boolean(v)) as [string, string][]).toString()
    return request<Task[]>(`/tasks${query ? `?${query}` : ''}`)
  },
  createTask: (body: Partial<Task>) => request<Task>('/tasks', { method: 'POST', ...json(body) }),
  updateTask: (id: string, body: Partial<Task>) => request<Task>(`/tasks/${id}`, { method: 'PUT', ...json(body) }),
  updateTaskStatus: (id: string, status: Task['status']) => request<Task>(`/tasks/${id}/status`, { method: 'PATCH', ...json({ status }) }),
  deleteTask: (id: string) => request<void>(`/tasks/${id}`, { method: 'DELETE' }),

  startSession: (body: Partial<StudySession>) => request<StudySession>('/study-sessions/start', { method: 'POST', ...json(body) }),
  finishSession: (id: string, body: Partial<StudySession>) => request<StudySession>(`/study-sessions/${id}/finish`, { method: 'POST', ...json(body) }),
  sessions: (goalId?: string) => request<StudySession[]>(`/study-sessions${goalId ? `?goal_id=${goalId}` : ''}`),

  knowledge: (subjectId: string) => request<KnowledgePoint[]>(`/subjects/${subjectId}/knowledge-points`),
  createKnowledge: (body: Partial<KnowledgePoint>) => request<KnowledgePoint>('/knowledge-points', { method: 'POST', ...json(body) }),
  updateKnowledge: (id: string, body: Partial<KnowledgePoint>) => request<KnowledgePoint>(`/knowledge-points/${id}`, { method: 'PUT', ...json(body) }),
  reviewKnowledge: (id: string, mastery: number, wrongDelta = 0) => request<KnowledgePoint>(`/knowledge-points/${id}/review`, { method: 'POST', ...json({ mastery, wrong_delta: wrongDelta }) }),

  assessments: (goalId?: string) => request<Assessment[]>(`/assessments${goalId ? `?goal_id=${goalId}` : ''}`),
  createAssessment: (body: Partial<Assessment>) => request<Assessment>('/assessments', { method: 'POST', ...json(body) }),
  updateAssessment: (id: string, body: Partial<Assessment>) => request<Assessment>(`/assessments/${id}`, { method: 'PUT', ...json(body) }),
  deleteAssessment: (id: string) => request<void>(`/assessments/${id}`, { method: 'DELETE' }),

  reviews: (goalId?: string) => request<WeeklyReview[]>(`/weekly-reviews${goalId ? `?goal_id=${goalId}` : ''}`),
  saveReview: (body: Partial<WeeklyReview>) => request<WeeklyReview>('/weekly-reviews', { method: 'POST', ...json(body) }),
  analytics: (goalId?: string) => request<AnalyticsOverview>(`/analytics/overview${goalId ? `?goal_id=${goalId}` : ''}`),
}
