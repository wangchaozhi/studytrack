import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'
import { Card, Empty, ErrorBox, Loading, MiniBars, PageHeader, Progress, StatCard, minutesText } from '../components/UI'

export function AnalyticsPage() {
  const [goalId, setGoalId] = useState('')
  const goals = useQuery({ queryKey: ['goals'], queryFn: api.goals })
  const analytics = useQuery({ queryKey: ['analytics', goalId], queryFn: () => api.analytics(goalId || undefined) })
  if (analytics.isLoading) return <Loading />
  if (analytics.error || !analytics.data) return <ErrorBox error={analytics.error} />
  const a = analytics.data
  const scoreData = a.score_trends.slice().reverse().map(x => ({ date: x.exam_date.slice(5), value: x.full_score ? Math.round(x.score / x.full_score * 100) : x.score }))
  return <>
    <PageHeader title="数据分析" subtitle="同时观察学习投入、计划执行和考试结果。" action={<select value={goalId} onChange={e => setGoalId(e.target.value)}><option value="">全部目标</option>{goals.data?.map(g => <option value={g.id} key={g.id}>{g.name}</option>)}</select>} />
    <div className="stats-grid">
      <StatCard icon="◷" label="累计学习" value={minutesText(a.total_minutes)} />
      <StatCard icon="✓" label="完成任务" value={a.completed_tasks} />
      <StatCard icon="↗" label="任务完成率" value={`${a.completion_rate.toFixed(1)}%`} />
      <StatCard icon="◎" label="平均专注度" value={`${a.average_focus_score.toFixed(1)} / 5`} />
    </div>
    <div className="stats-grid">
      <StatCard icon="◇" label="平均掌握度" value={`${a.average_mastery_score.toFixed(1)} / 5`} />
      <StatCard icon="◴" label="平均单次专注" value={minutesText(Math.round(a.average_session_minutes))} />
      <StatCard icon="!" label="逾期任务" value={a.overdue_tasks} />
      <StatCard icon="↑" label="高效日期 / 科目" value={a.best_study_date ? `${a.best_study_date.slice(5)} · ${a.most_focused_subject || '综合'}` : '暂无'} />
    </div>
    <div className="analytics-grid">
      <Card className="span-2"><div className="card-title"><div><h2>近 30 天学习投入</h2><p>每日实际学习分钟数</p></div></div><MiniBars data={a.study_time} /></Card>
      <Card><div className="card-title"><div><h2>任务完成趋势</h2><p>近 14 天完成数量</p></div></div><MiniBars data={a.task_completion} /></Card>
      <Card><div className="card-title"><div><h2>成绩得分率</h2><p>最近 20 次练习与模考</p></div></div>{scoreData.length ? <MiniBars data={scoreData} /> : <Empty />}</Card>
      <Card className="span-2"><div className="card-title"><div><h2>科目执行进度</h2><p>任务完成率与累计学习时间</p></div></div>{a.subject_progress.length ? <div className="progress-list large">{a.subject_progress.map(s => <div key={s.subject_id}><div className="progress-head"><span>{s.subject_name}<small> · {minutesText(s.minutes)}</small></span><b>{Math.round(s.progress)}%</b></div><Progress value={s.progress} /></div>)}</div> : <Empty />}</Card>
      <Card><div className="card-title"><div><h2>最薄弱知识点</h2><p>建议优先安排复习</p></div></div>{a.weak_knowledge.length ? <div className="rank-list">{a.weak_knowledge.slice(0, 8).map((k, i) => <div key={k.id}><span>{i + 1}</span><div><b>{k.name}</b><small>错题 {k.wrong_count} · 复习 {k.review_count}</small></div><strong>{Math.round(k.mastery)}%</strong></div>)}</div> : <Empty />}</Card>
    </div>
  </>
}
