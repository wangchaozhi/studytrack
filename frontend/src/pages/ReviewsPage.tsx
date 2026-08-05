import { FormEvent, useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import { Card, Empty, ErrorBox, Loading, PageHeader, dateValue, minutesText } from '../components/UI'

function currentWeekStart() {
  const d = new Date(); const day = d.getDay() || 7; d.setDate(d.getDate() - day + 1); return dateValue(d)
}
function weekEnd(start: string) { const d = new Date(`${start}T12:00:00`); d.setDate(d.getDate() + 6); return dateValue(d) }

export function ReviewsPage() {
  const [goalId, setGoalId] = useState('')
  const [weekStart, setWeekStart] = useState(currentWeekStart())
  const qc = useQueryClient()
  const goals = useQuery({ queryKey: ['goals'], queryFn: api.goals })
  useEffect(() => { if (!goalId && goals.data?.length) setGoalId(goals.data[0].id) }, [goalId, goals.data])
  const reviews = useQuery({ queryKey: ['reviews', goalId], queryFn: () => api.reviews(goalId || undefined), enabled: !!goalId })
  const tasks = useQuery({ queryKey: ['review-tasks', goalId, weekStart], queryFn: () => api.tasks({ goal_id: goalId, from: weekStart, to: weekEnd(weekStart) }), enabled: !!goalId })
  const sessions = useQuery({ queryKey: ['review-sessions', goalId], queryFn: () => api.sessions(goalId), enabled: !!goalId })
  const existing = reviews.data?.find(review => review.week_start === weekStart)
  const metrics = useMemo(() => {
    const list = tasks.data || []
    const planned = list.reduce((s, t) => s + t.estimated_minutes, 0)
    const complete = list.filter(t => t.status === 'completed')
    const actual = (sessions.data || []).filter(session => {
      const day = dateValue(new Date(session.started_at)); return day >= weekStart && day <= weekEnd(weekStart)
    }).reduce((sum, session) => sum + session.duration_minutes, 0)
    return { planned, actual, completion: list.length ? complete.length / list.length * 100 : 0, overdue: list.filter(t => t.status === 'overdue').length }
  }, [tasks.data, sessions.data, weekStart])
  const save = useMutation({ mutationFn: api.saveReview, onSuccess: () => qc.invalidateQueries({ queryKey: ['reviews', goalId] }) })

  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault(); const f = new FormData(e.currentTarget)
    save.mutate({ goal_id: goalId, week_start: weekStart, planned_minutes: Number(f.get('planned_minutes') || 0), actual_minutes: Number(f.get('actual_minutes') || 0), completion_rate: Number(f.get('completion_rate') || 0), summary: String(f.get('summary') || ''), problems: String(f.get('problems') || ''), next_week_focus: String(f.get('next_week_focus') || '') })
  }

  if (goals.isLoading) return <Loading />
  return <>
    <PageHeader title="每周复盘" subtitle="总结执行结果，识别问题，并明确下周重点。" action={<select value={goalId} onChange={e => setGoalId(e.target.value)}><option value="">请选择目标</option>{goals.data?.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}</select>} />
    {!goalId ? <Card><Empty text="先创建并选择一个学习目标" /></Card> : <>
      <Card className="filter-card"><label>周开始日期<input type="date" value={weekStart} onChange={e => setWeekStart(e.target.value)} /></label><div className="filter-summary"><b>{weekStart}</b><span>至 {weekEnd(weekStart)}</span></div><Link className="ghost-button" to="/plans">调整学习计划</Link></Card>
      <div className="week-summary card"><div><span>系统统计计划</span><strong>{minutesText(metrics.planned)}</strong></div><div><span>系统统计实际</span><strong>{minutesText(metrics.actual)}</strong></div><div><span>任务完成率</span><strong>{metrics.completion.toFixed(0)}%</strong></div><div><span>逾期任务</span><strong>{metrics.overdue}</strong></div></div>
      <div className="two-column review-layout">
        <Card><div className="card-title"><div><h2>{existing ? '编辑本周复盘' : '填写本周复盘'}</h2><p>{weekStart} 开始的一周；统计值可按实际情况修正</p></div></div><form key={`${weekStart}-${existing?.updated_at || 'new'}-${metrics.planned}-${metrics.actual}`} className="form-stack" onSubmit={submit}>
          <div className="form-grid three"><label>计划分钟<input name="planned_minutes" type="number" min="0" defaultValue={existing?.planned_minutes ?? metrics.planned} /></label><label>实际分钟<input name="actual_minutes" type="number" min="0" defaultValue={existing?.actual_minutes ?? metrics.actual} /></label><label>完成率 %<input name="completion_rate" type="number" min="0" max="100" step="0.1" defaultValue={existing?.completion_rate ?? metrics.completion.toFixed(1)} /></label></div>
          <label>本周做得好的地方<textarea name="summary" rows={4} defaultValue={existing?.summary} placeholder="例如：上午学习效率较高，数学任务完成稳定" /></label>
          <label>问题与阻碍<textarea name="problems" rows={4} defaultValue={existing?.problems} placeholder="例如：专业课任务拆分过大，连续两天延期" /></label>
          <label>下周重点<textarea name="next_week_focus" rows={4} defaultValue={existing?.next_week_focus} placeholder="例如：优先突破概率论，并完成两套英语阅读真题" /></label>
          {save.error && <ErrorBox error={save.error} />}{save.isSuccess && <div className="success-box">复盘已保存</div>}
          <button className="primary-button full-button">{existing ? '更新本周复盘' : '保存本周复盘'}</button>
        </form></Card>
        <Card><div className="card-title"><div><h2>历史复盘</h2><p>点击日期可重新查看和修改</p></div></div>{reviews.isLoading ? <Loading /> : (reviews.data || []).length === 0 ? <Empty text="保存本周复盘后显示" /> : <div className="review-list">{reviews.data?.map(r => <article key={r.id} onClick={() => setWeekStart(r.week_start)} role="button" tabIndex={0}><div><b>{r.week_start}</b><span>{r.completion_rate.toFixed(0)}% 完成</span></div><p>{r.summary || '未填写总结'}</p>{r.problems && <small>问题：{r.problems}</small>}{r.next_week_focus && <small>下周：{r.next_week_focus}</small>}</article>)}</div>}</Card>
      </div>
    </>}
  </>
}
