import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'
import { Card, Empty, ErrorBox, Loading, PageHeader, dateValue, minutesText } from '../components/UI'

function startOfWeek(date: Date) {
  const d = new Date(date)
  const day = d.getDay() || 7
  d.setDate(d.getDate() - day + 1)
  return d
}
function iso(d: Date) { return dateValue(d) }

export function PlansPage() {
  const [offset, setOffset] = useState(0)
  const [goalId, setGoalId] = useState('')
  const base = useMemo(() => { const d = startOfWeek(new Date()); d.setDate(d.getDate() + offset * 7); return d }, [offset])
  const days = Array.from({ length: 7 }, (_, i) => { const d = new Date(base); d.setDate(d.getDate() + i); return d })
  const goals = useQuery({ queryKey: ['goals'], queryFn: api.goals })
  const tasks = useQuery({ queryKey: ['plan-tasks', iso(days[0]), iso(days[6]), goalId], queryFn: () => api.tasks({ from: iso(days[0]), to: iso(days[6]), goal_id: goalId || undefined }) })
  const total = (tasks.data || []).reduce((sum, t) => sum + t.estimated_minutes, 0)
  const done = (tasks.data || []).filter(t => t.status === 'completed').reduce((sum, t) => sum + t.estimated_minutes, 0)

  return <>
    <PageHeader title="学习计划" subtitle="以周为单位平衡各科目投入，及时处理计划偏差。" />
    <Card className="plan-toolbar"><div><button className="ghost-button" onClick={() => setOffset(offset - 1)}>← 上一周</button><button className="ghost-button" onClick={() => setOffset(0)}>本周</button><button className="ghost-button" onClick={() => setOffset(offset + 1)}>下一周 →</button></div><b>{iso(days[0])} — {iso(days[6])}</b><select value={goalId} onChange={e => setGoalId(e.target.value)}><option value="">全部目标</option>{goals.data?.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}</select></Card>
    <div className="week-summary card"><div><span>本周计划</span><strong>{minutesText(total)}</strong></div><div><span>已完成</span><strong>{minutesText(done)}</strong></div><div><span>完成率</span><strong>{total ? Math.round(done / total * 100) : 0}%</strong></div></div>
    {tasks.isLoading ? <Loading /> : tasks.error ? <ErrorBox error={tasks.error} /> : <div className="week-grid">{days.map(day => {
      const list = (tasks.data || []).filter(t => t.planned_date === iso(day))
      return <Card className="day-column" key={iso(day)}><div className={`day-head ${iso(day) === iso(new Date()) ? 'today' : ''}`}><b>{day.toLocaleDateString('zh-CN', { weekday: 'short' })}</b><span>{day.getMonth() + 1}/{day.getDate()}</span></div>{list.length === 0 ? <Empty text="无任务" /> : list.map(t => <div className={`plan-task ${t.status}`} key={t.id}><b>{t.title}</b><small>{t.subject_name || t.goal_name}</small><span>{t.estimated_minutes} 分钟</span></div>)}</Card>
    })}</div>}
  </>
}
