import { FormEvent, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Subject } from '../types'
import { Card, Empty, ErrorBox, Loading, Modal, PageHeader, Progress, StatCard, daysUntil, goalTypeLabel, minutesText, today } from '../components/UI'

export function GoalDetailPage() {
  const { goalId = '' } = useParams()
  const [showSubject, setShowSubject] = useState(false)
  const [editingSubject, setEditingSubject] = useState<Subject | null>(null)
  const [showPhase, setShowPhase] = useState(false)
  const qc = useQueryClient()
  const goal = useQuery({ queryKey: ['goal', goalId], queryFn: () => api.goal(goalId), enabled: !!goalId })
  const phases = useQuery({ queryKey: ['phases', goalId], queryFn: () => api.phases(goalId), enabled: !!goalId })
  const subjects = useQuery({ queryKey: ['subjects', goalId], queryFn: () => api.subjects(goalId), enabled: !!goalId })
  const dashboard = useQuery({ queryKey: ['goal-dashboard', goalId], queryFn: () => api.dashboard(goalId), enabled: !!goalId })
  const saveSubject = useMutation({ mutationFn: (body: Record<string, unknown>) => editingSubject ? api.updateSubject(editingSubject.id, body) : api.createSubject(goalId, body), onSuccess: () => { qc.invalidateQueries({ queryKey: ['subjects', goalId] }); setShowSubject(false); setEditingSubject(null) } })
  const createPhase = useMutation({ mutationFn: (body: Record<string, unknown>) => api.createPhase(goalId, body), onSuccess: () => { qc.invalidateQueries({ queryKey: ['phases', goalId] }); setShowPhase(false) } })

  if (goal.isLoading || phases.isLoading || subjects.isLoading || dashboard.isLoading) return <Loading />
  if (goal.error || !goal.data) return <ErrorBox error={goal.error} />
  const g = goal.data
  const d = dashboard.data
  const descriptor = g.type === 'postgraduate' ? [g.settings.school, g.settings.major].filter(Boolean).join(' · ') : g.type === 'degree' ? String(g.settings.semester || '学位课程复习') : `当前 ${g.settings.current_score || '-'} · 目标 ${g.settings.target_band || g.target_score}`

  function submitSubject(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = new FormData(e.currentTarget)
    saveSubject.mutate({ name: f.get('name'), category: f.get('category') || 'general', target_score: Number(f.get('target_score') || 0), current_score: Number(f.get('current_score') || 0), planned_minutes: Number(f.get('planned_minutes') || 0), color: f.get('color') || '#4f46e5' })
  }
  function submitPhase(e: FormEvent<HTMLFormElement>) {
    e.preventDefault(); const f = new FormData(e.currentTarget)
    createPhase.mutate({ name: f.get('name'), start_date: f.get('start_date'), end_date: f.get('end_date'), sort_order: Number(f.get('sort_order') || phases.data?.length || 0), status: f.get('status') || 'pending' })
  }

  return <>
    <PageHeader title={g.name} subtitle={`${goalTypeLabel[g.type]} · ${g.exam_date} · 距离考试 ${daysUntil(g.exam_date)} 天`} action={<Link className="primary-button" to={`/tasks?goal_id=${g.id}`}>安排任务</Link>} />
    <div className="goal-hero card">
      <div><span className={`goal-type ${g.type}`}>{goalTypeLabel[g.type]}</span><h2>目标分数 {g.target_score || '未设置'}</h2><p>{descriptor}</p><p>从 {g.start_date} 开始，按阶段持续推进。</p></div>
      <div className="hero-progress"><strong>{Math.round(g.progress)}%</strong><Progress value={g.progress} /></div>
    </div>
    <div className="stats-grid three">
      <StatCard icon="◷" label="本周学习" value={minutesText(d?.week_minutes || 0)} />
      <StatCard icon="✓" label="今日完成" value={`${d?.today_completed || 0}/${d?.today_total || 0}`} />
      <StatCard icon="◇" label="薄弱知识点" value={d?.weak_knowledge.length || 0} />
    </div>
    <div className="two-column">
      <Card>
        <div className="card-title"><div><h2>复习阶段</h2><p>系统已按目标日期自动划分</p></div><button className="text-button" onClick={() => setShowPhase(true)}>＋ 添加</button></div>
        <div className="timeline">{(phases.data || []).map((p, i) => <div className={`timeline-item ${p.status}`} key={p.id}><span>{i + 1}</span><div><b>{p.name}</b><small>{p.start_date} — {p.end_date}</small></div><em>{p.status === 'active' ? '当前' : p.status === 'completed' ? '完成' : '待开始'}</em></div>)}</div>
      </Card>
      <Card>
        <div className="card-title"><div><h2>科目进度</h2><p>目标分数与学习投入</p></div><button className="text-button" onClick={() => { setEditingSubject(null); setShowSubject(true) }}>＋ 添加</button></div>
        {(subjects.data || []).length === 0 ? <Empty /> : <div className="subject-list">{subjects.data?.map(s => {
          const p = d?.subject_progress.find(x => x.subject_id === s.id)
          return <div className="subject-row" key={s.id}><span className="subject-dot" style={{ background: s.color }} /><div><b>{s.name}</b><small>目标 {s.target_score || '-'} · 当前 {s.current_score || '-'} · 已学习 {minutesText(p?.minutes || 0)}</small><Progress value={p?.progress || 0} /></div><strong>{Math.round(p?.progress || 0)}%</strong><button className="ghost-button small" onClick={() => { setEditingSubject(s); setShowSubject(true) }}>编辑</button></div>
        })}</div>}
      </Card>
    </div>
    <div className="quick-actions card"><Link to={`/tasks?goal_id=${goalId}`}>✓ 管理任务</Link><Link to="/focus">◷ 开始专注</Link><Link to={`/knowledge?goal_id=${goalId}`}>◇ 知识点</Link><Link to={`/assessments?goal_id=${goalId}`}>▥ 记录成绩</Link></div>

    {showSubject && <Modal title={editingSubject ? '编辑科目' : '添加科目'} onClose={() => { setShowSubject(false); setEditingSubject(null) }}><form className="form-stack" onSubmit={submitSubject}>
      <label>科目名称<input name="name" defaultValue={editingSubject?.name} required placeholder="例如：数据结构" /></label>
      <div className="form-grid"><label>分类<input name="category" defaultValue={editingSubject?.category || 'general'} /></label><label>目标分数<input name="target_score" type="number" step="0.5" defaultValue={editingSubject?.target_score} /></label></div>
      <div className="form-grid"><label>当前分数<input name="current_score" type="number" step="0.5" defaultValue={editingSubject?.current_score} /></label><label>每周计划分钟<input name="planned_minutes" type="number" defaultValue={editingSubject?.planned_minutes || 300} /></label></div>
      <label>标识颜色<input name="color" type="color" defaultValue={editingSubject?.color || '#4f46e5'} /></label>
      {saveSubject.error && <ErrorBox error={saveSubject.error} />}
      <div className="modal-actions"><button type="button" className="ghost-button" onClick={() => { setShowSubject(false); setEditingSubject(null) }}>取消</button><button className="primary-button">保存科目</button></div>
    </form></Modal>}
    {showPhase && <Modal title="添加复习阶段" onClose={() => setShowPhase(false)}><form className="form-stack" onSubmit={submitPhase}>
      <label>阶段名称<input name="name" required placeholder="例如：二轮强化" /></label>
      <div className="form-grid"><label>开始日期<input name="start_date" type="date" min={g.start_date} max={g.exam_date} defaultValue={today()} required /></label><label>结束日期<input name="end_date" type="date" min={g.start_date} max={g.exam_date} defaultValue={g.exam_date} required /></label></div>
      <div className="form-grid"><label>顺序<input name="sort_order" type="number" min="0" defaultValue={phases.data?.length || 0} /></label><label>状态<select name="status"><option value="pending">待开始</option><option value="active">进行中</option><option value="completed">已完成</option></select></label></div>
      {createPhase.error && <ErrorBox error={createPhase.error} />}
      <div className="modal-actions"><button type="button" className="ghost-button" onClick={() => setShowPhase(false)}>取消</button><button className="primary-button">保存阶段</button></div>
    </form></Modal>}
  </>
}
