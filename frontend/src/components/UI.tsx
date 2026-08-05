import type { ReactNode } from 'react'

export function PageHeader({ title, subtitle, action }: { title: string; subtitle?: string; action?: ReactNode }) {
  return <div className="page-header"><div><h1>{title}</h1>{subtitle && <p>{subtitle}</p>}</div>{action}</div>
}

export function Card({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <section className={`card ${className}`}>{children}</section>
}

export function StatCard({ label, value, hint, icon }: { label: string; value: string | number; hint?: string; icon?: string }) {
  return <Card className="stat-card"><div className="stat-icon">{icon}</div><div><span>{label}</span><strong>{value}</strong>{hint && <small>{hint}</small>}</div></Card>
}

export function Progress({ value, label }: { value: number; label?: string }) {
  const safe = Math.max(0, Math.min(100, value || 0))
  return <div className="progress-wrap"><div className="progress-track"><div className="progress-fill" style={{ width: `${safe}%` }} /></div>{label && <span>{label}</span>}</div>
}

export function Empty({ text = '暂无数据' }: { text?: string }) {
  return <div className="empty"><span>◇</span><p>{text}</p></div>
}

export function Loading() { return <div className="loading"><div className="spinner" /></div> }

export function ErrorBox({ error }: { error: unknown }) {
  return <div className="error-box">{error instanceof Error ? error.message : '加载失败'}</div>
}

export function Modal({ title, children, onClose }: { title: string; children: ReactNode; onClose: () => void }) {
  return <div className="modal-backdrop" onMouseDown={onClose}><div className="modal" onMouseDown={(e) => e.stopPropagation()}><div className="modal-head"><h2>{title}</h2><button className="icon-button" onClick={onClose}>×</button></div>{children}</div></div>
}

export function MiniBars({ data }: { data: { date: string; value: number }[] }) {
  const max = Math.max(...data.map((x) => x.value), 1)
  return <div className="mini-bars">{data.map((x) => <div className="bar-item" key={x.date} title={`${x.date}: ${x.value}`}><div className="bar" style={{ height: `${Math.max(5, x.value / max * 100)}%` }} /><small>{x.date.slice(-2)}</small></div>)}</div>
}

export const goalTypeLabel: Record<string, string> = { degree: '学位复习', postgraduate: '考研', ielts: '雅思' }
export const statusLabel: Record<string, string> = { pending: '待开始', in_progress: '进行中', completed: '已完成', overdue: '已逾期', skipped: '已跳过', active: '进行中', paused: '已暂停', archived: '已归档' }
export function dateValue(date = new Date()) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
export const today = () => dateValue()
export const minutesText = (minutes: number) => minutes >= 60 ? `${Math.floor(minutes / 60)}小时${minutes % 60 ? `${minutes % 60}分` : ''}` : `${minutes}分钟`
export const daysUntil = (date: string) => Math.max(0, Math.ceil((new Date(`${date}T23:59:59`).getTime() - Date.now()) / 86_400_000))
