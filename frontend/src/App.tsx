import { useEffect } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { api } from './api/client'
import { AppLayout } from './components/AppLayout'
import { useAuthStore } from './store/auth'
import { LoginPage } from './pages/LoginPage'
import { DashboardPage } from './pages/DashboardPage'
import { GoalsPage } from './pages/GoalsPage'
import { GoalDetailPage } from './pages/GoalDetailPage'
import { TasksPage } from './pages/TasksPage'
import { PlansPage } from './pages/PlansPage'
import { FocusPage } from './pages/FocusPage'
import { KnowledgePage } from './pages/KnowledgePage'
import { AssessmentsPage } from './pages/AssessmentsPage'
import { AnalyticsPage } from './pages/AnalyticsPage'
import { ReviewsPage } from './pages/ReviewsPage'
import { SettingsPage } from './pages/SettingsPage'

function Protected({ children }: { children: React.ReactNode }) {
  const { user, hydrated } = useAuthStore()
  if (!hydrated) return <div className="center-screen"><div className="spinner" /></div>
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  const { setUser, setHydrated } = useAuthStore()

  useEffect(() => {
    const token = localStorage.getItem('studytrack_token')
    if (!token) {
      setHydrated(true)
      return
    }
    api.me().then(setUser).catch(() => {
      localStorage.removeItem('studytrack_token')
      setUser(null)
    }).finally(() => setHydrated(true))
  }, [setHydrated, setUser])

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/" element={<Protected><AppLayout /></Protected>}>
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<DashboardPage />} />
        <Route path="goals" element={<GoalsPage />} />
        <Route path="goals/:goalId" element={<GoalDetailPage />} />
        <Route path="tasks" element={<TasksPage />} />
        <Route path="plans" element={<PlansPage />} />
        <Route path="focus" element={<FocusPage />} />
        <Route path="knowledge" element={<KnowledgePage />} />
        <Route path="assessments" element={<AssessmentsPage />} />
        <Route path="analytics" element={<AnalyticsPage />} />
        <Route path="reviews" element={<ReviewsPage />} />
        <Route path="settings" element={<SettingsPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
