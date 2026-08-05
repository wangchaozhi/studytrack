import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import { Card, Empty, ErrorBox, Loading, MiniBars, PageHeader, Progress, StatCard, daysUntil, goalTypeLabel, minutesText, statusLabel } from '../components/UI'
import { useAuthStore } from '../store/auth'

export function DashboardPage() {
  const qc = useQueryClient()
  const user = useAuthStore(s => s.user)
  const { data, error, isLoading } = useQuery({ queryKey: ['dashboard'], queryFn: () => api.dashboard() })
  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: 'completed' | 'pending' }) => api.updateTaskStatus(id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dashboard'] }),
  })

  if (isLoading) return <Loading />
  if (error || !data) return <ErrorBox error={error} />

  const hour = Number(new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', hour12: false, timeZone: user?.timezone || 'Asia/Shanghai' }).format(new Date()).slice(0, 2))
  const greeting = hour < 6 ? '夜深了' : hour < 12 ? '上午好' : hour < 18 ? '下午好' : '晚上好'
  return <>
    <PageHeader title={`${greeting}，${user?.nickname || '同学'}`} subtitle="专注完成当前最重要的一件事。" action={<Link className="primary-button" to="/focus">开始专注</Link>} />
    <div className="stats-grid">
      <StatCard icon="◷" label="今日学习" value={minutesText(data.today_minutes)} hint={`本周 ${minutesText(data.week_minutes)}`} />
      <StatCard icon="✓" label="今日任务" value={`${data.today_completed}/${data.today_total}`} hint={`${data.today_total ? Math.round(data.today_completed / data.today_total * 100) : 0}% 已完成`} />
      <StatCard icon="↗" label="连续学习" value={`${data.streak_days} 天`} hint="保持节奏" />
      <StatCard icon="!" label="逾期任务" value={data.overdue_tasks} hint={data.overdue_tasks ? '建议今天重新安排' : '计划状态良好'} />
    </div>

    <div className="dashboard-grid">
      <Card className="span-2">
        <div className="card-title"><div><h2>今日任务</h2><p>按优先级完成计划</p></div><Link to="/tasks">查看全部</Link></div>
        {data.today_tasks.length === 0 ? <Empty text="今天还没有任务，先安排一个学习事项" /> : <div className="task-list compact">
          {data.today_tasks.map((task) => <div className={`task-row ${task.status === 'completed' ? 'done' : ''}`} key={task.id}>
            <button className="check-button" onClick={() => statusMutation.mutate({ id: task.id, status: task.status === 'completed' ? 'pending' : 'completed' })}>{task.status === 'completed' ? '✓' : ''}</button>
            <div className="task-main"><b>{task.title}</b><small>{task.subject_name || task.goal_name} · 预计 {task.estimated_minutes} 分钟</small></div>
            <span className={`badge ${task.priority}`}>{task.priority === 'high' ? '高优先' : statusLabel[task.status]}</span>
          </div>)}
        </div>}
      </Card>

      <Card>
        <div className="card-title"><div><h2>考试倒计时</h2><p>当前进行中的目标</p></div></div>
        {data.goals.length === 0 ? <Empty text="创建目标后显示倒计时" /> : <div className="countdown-list">{data.goals.filter(g => g.status === 'active').slice(0, 4).map(goal => <Link to={`/goals/${goal.id}`} key={goal.id} className="countdown-item"><div><b>{goal.name}</b><small>{goalTypeLabel[goal.type]} · {goal.exam_date}</small></div><strong>{daysUntil(goal.exam_date)}<small>天</small></strong></Link>)}</div>}
      </Card>

      <Card className="span-2">
        <div className="card-title"><div><h2>近 14 天学习趋势</h2><p>每日实际专注分钟数</p></div></div>
        <MiniBars data={data.daily_study_trend} />
      </Card>

      <Card>
        <div className="card-title"><div><h2>科目进度</h2><p>基于任务完成情况</p></div><Link to="/analytics">详情</Link></div>
        {data.subject_progress.length === 0 ? <Empty /> : <div className="progress-list">{data.subject_progress.slice(0, 6).map(s => <div key={s.subject_id}><div className="progress-head"><span>{s.subject_name}</span><b>{Math.round(s.progress)}%</b></div><Progress value={s.progress} /></div>)}</div>}
      </Card>

      <Card className="span-2">
        <div className="card-title"><div><h2>薄弱知识点</h2><p>优先安排复习和练习</p></div><Link to="/knowledge">管理知识点</Link></div>
        {data.weak_knowledge.length === 0 ? <Empty text="添加知识点并记录掌握度后显示" /> : <div className="weak-grid">{data.weak_knowledge.map(k => <div className="weak-item" key={k.id}><span className="knowledge-icon">◇</span><div><b>{k.name}</b><small>错题 {k.wrong_count} · 复习 {k.review_count} 次</small></div><strong>{Math.round(k.mastery)}%</strong></div>)}</div>}
      </Card>

      <Card>
        <div className="card-title"><div><h2>最近成绩</h2><p>关注与目标的差距</p></div><Link to="/assessments">记录成绩</Link></div>
        {data.recent_assessments.length === 0 ? <Empty /> : <div className="score-list">{data.recent_assessments.slice(0, 5).map(a => <div key={a.id}><div><b>{a.name}</b><small>{a.exam_date}</small></div><strong>{a.score}<small>/{a.full_score}</small></strong></div>)}</div>}
      </Card>
    </div>
  </>
}
