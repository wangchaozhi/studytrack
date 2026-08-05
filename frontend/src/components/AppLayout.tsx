import { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/auth'

const nav = [
  ['◫', '首页', '/dashboard'],
  ['◎', '我的目标', '/goals'],
  ['✓', '今日任务', '/tasks'],
  ['▦', '学习计划', '/plans'],
  ['◷', '专注学习', '/focus'],
  ['◇', '知识点', '/knowledge'],
  ['▥', '模考成绩', '/assessments'],
  ['⌁', '数据分析', '/analytics'],
  ['✎', '每周复盘', '/reviews'],
  ['⚙', '系统设置', '/settings'],
]

export function AppLayout() {
  const [open, setOpen] = useState(false)
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()
  return (
    <div className="app-shell">
      <aside className={`sidebar ${open ? 'open' : ''}`}>
        <div className="brand"><span className="brand-mark">迹</span><div><b>学迹</b><small>StudyTrack</small></div></div>
        <nav>
          {nav.map(([icon, label, path]) => (
            <NavLink key={path} to={path} onClick={() => setOpen(false)} className={({ isActive }) => isActive ? 'active' : ''}>
              <span>{icon}</span>{label}
            </NavLink>
          ))}
        </nav>
        <button className="user-card" onClick={() => navigate('/settings')}>
          <span className="avatar">{user?.nickname?.slice(0, 1) || '学'}</span>
          <span><b>{user?.nickname}</b><small>{user?.email}</small></span>
        </button>
      </aside>
      {open && <button aria-label="关闭菜单" className="sidebar-overlay" onClick={() => setOpen(false)} />}
      <main className="main-area">
        <header className="topbar">
          <button className="icon-button menu-button" onClick={() => setOpen(true)}>☰</button>
          <div className="topbar-spacer" />
          <span className="today-label">{new Date().toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'short' })}</span>
          <button className="ghost-button" onClick={() => { logout(); navigate('/login') }}>退出</button>
        </header>
        <div className="page-wrap"><Outlet /></div>
      </main>
    </div>
  )
}
