import { FormEvent, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import { useAuthStore } from '../store/auth'

export function LoginPage() {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate()
  const { setToken, setUser, setHydrated } = useAuthStore()

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setLoading(true)
    const form = new FormData(event.currentTarget)
    try {
      const result = mode === 'login'
        ? await api.login({ email: String(form.get('email')), password: String(form.get('password')) })
        : await api.register({ email: String(form.get('email')), password: String(form.get('password')), nickname: String(form.get('nickname')), timezone: Intl.DateTimeFormat().resolvedOptions().timeZone })
      setToken(result.token)
      setUser(result.user)
      setHydrated(true)
      navigate('/dashboard')
    } catch (e) {
      setError(e instanceof Error ? e.message : '操作失败')
    } finally {
      setLoading(false)
    }
  }

  return <div className="auth-page">
    <div className="auth-visual">
      <div className="auth-copy">
        <span className="auth-logo">迹</span>
        <h1>把长期目标，变成每天可完成的计划。</h1>
        <p>统一追踪学位复习、考研和雅思，从学习投入到知识掌握，再到模考成绩。</p>
        <div className="feature-row"><span>✓ 目标阶段</span><span>✓ 专注计时</span><span>✓ 成绩分析</span></div>
      </div>
    </div>
    <div className="auth-form-wrap">
      <form className="auth-form" onSubmit={submit}>
        <div className="brand auth-brand"><span className="brand-mark">迹</span><div><b>学迹</b><small>StudyTrack</small></div></div>
        <h2>{mode === 'login' ? '欢迎回来' : '创建学习账户'}</h2>
        <p>{mode === 'login' ? '登录后继续今天的学习计划。' : '现在开始建立你的第一个学习目标。'}</p>
        {mode === 'register' && <label>昵称<input name="nickname" placeholder="你的昵称" required /></label>}
        <label>邮箱<input type="email" name="email" placeholder="name@example.com" required /></label>
        <label>密码<input type="password" name="password" minLength={8} placeholder="至少 8 位" required /></label>
        {error && <div className="error-box">{error}</div>}
        <button className="primary-button full-button" disabled={loading}>{loading ? '处理中…' : mode === 'login' ? '登录' : '注册并开始'}</button>
        <button type="button" className="text-button" onClick={() => { setMode(mode === 'login' ? 'register' : 'login'); setError('') }}>
          {mode === 'login' ? '没有账户？立即注册' : '已有账户？返回登录'}
        </button>
      </form>
    </div>
  </div>
}
