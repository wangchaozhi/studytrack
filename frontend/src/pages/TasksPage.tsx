import { FormEvent, useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Task } from '../types'
import { Card, Empty, ErrorBox, Loading, Modal, PageHeader, statusLabel, today } from '../components/UI'

export function TasksPage() {
  const [params] = useSearchParams()
  const [goalId, setGoalId] = useState(params.get('goal_id') || '')
  const [date, setDate] = useState(params.get('date') || today())
  const [status, setStatus] = useState(params.get('status') || '')
  const [show, setShow] = useState(false)
  const [editing, setEditing] = useState<Task | null>(null)
  const [formGoalId, setFormGoalId] = useState(goalId)
  const [formSubjectId, setFormSubjectId] = useState('')
  const qc = useQueryClient()
  const goals = useQuery({ queryKey: ['goals'], queryFn: api.goals })
  const subjects = useQuery({ queryKey: ['subjects', formGoalId], queryFn: () => api.subjects(formGoalId), enabled: !!formGoalId })
  const phases = useQuery({ queryKey: ['phases', formGoalId], queryFn: () => api.phases(formGoalId), enabled: !!formGoalId })
  const knowledge = useQuery({ queryKey: ['knowledge', formSubjectId], queryFn: () => api.knowledge(formSubjectId), enabled: !!formSubjectId })
  const tasks = useQuery({ queryKey: ['tasks', date, goalId, status], queryFn: () => api.tasks({ date, goal_id: goalId || undefined, status: status || undefined }) })
  const refresh = () => { qc.invalidateQueries({ queryKey: ['tasks'] }); qc.invalidateQueries({ queryKey: ['dashboard'] }); qc.invalidateQueries({ queryKey: ['analytics'] }) }
  const save = useMutation({ mutationFn: (body: Partial<Task>) => editing ? api.updateTask(editing.id, body) : api.createTask(body), onSuccess: () => { refresh(); closeModal() } })
  const updateStatus = useMutation({ mutationFn: ({ id, value }: { id: string; value: Task['status'] }) => api.updateTaskStatus(id, value), onSuccess: refresh })
  const remove = useMutation({ mutationFn: api.deleteTask, onSuccess: refresh })

  useEffect(() => { if (!formGoalId && goals.data?.length) setFormGoalId(goalId || goals.data[0].id) }, [formGoalId, goalId, goals.data])
  const counts = useMemo(() => {
    const list = tasks.data || []
    return { total: list.length, done: list.filter(x => x.status === 'completed').length, minutes: list.reduce((sum, x) => sum + x.estimated_minutes, 0) }
  }, [tasks.data])

  function openCreate() { setEditing(null); setFormGoalId(goalId || goals.data?.[0]?.id || ''); setFormSubjectId(''); setShow(true) }
  function openEdit(task: Task) { setEditing(task); setFormGoalId(task.goal_id); setFormSubjectId(task.subject_id || ''); setShow(true) }
  function closeModal() { setShow(false); setEditing(null) }
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault(); const f = new FormData(e.currentTarget)
    save.mutate({
      goal_id: String(f.get('goal_id')), phase_id: String(f.get('phase_id') || '') || null,
      subject_id: String(f.get('subject_id') || '') || null, knowledge_point_id: String(f.get('knowledge_point_id') || '') || null,
      title: String(f.get('title')), description: String(f.get('description') || ''), task_type: String(f.get('task_type')),
      priority: String(f.get('priority')) as Task['priority'], status: String(f.get('status') || 'pending') as Task['status'],
      planned_date: String(f.get('planned_date')), estimated_minutes: Number(f.get('estimated_minutes') || 0), actual_minutes: Number(f.get('actual_minutes') || 0),
    })
  }

  if (goals.isLoading) return <Loading />
  return <>
    <PageHeader title="今日任务" subtitle="把计划落到清晰、可完成的学习事项。" action={<button className="primary-button" onClick={openCreate} disabled={!goals.data?.length}>＋ 新建任务</button>} />
    <Card className="filter-card">
      <label>日期<input type="date" value={date} onChange={e => setDate(e.target.value)} /></label>
      <label>学习目标<select value={goalId} onChange={e => setGoalId(e.target.value)}><option value="">全部目标</option>{goals.data?.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}</select></label>
      <label>状态<select value={status} onChange={e => setStatus(e.target.value)}><option value="">全部状态</option><option value="pending">待开始</option><option value="in_progress">进行中</option><option value="completed">已完成</option><option value="overdue">已逾期</option><option value="skipped">已跳过</option></select></label>
      <div className="filter-summary"><b>{counts.done}/{counts.total}</b><span>完成 · 计划 {counts.minutes} 分钟</span></div>
    </Card>
    {tasks.isLoading ? <Loading /> : tasks.error ? <ErrorBox error={tasks.error} /> : (tasks.data || []).length === 0 ? <Card><Empty text="这个日期还没有任务" /></Card> : <Card className="task-board">
      <div className="task-list">{tasks.data?.map(task => <div className={`task-row large ${task.status === 'completed' ? 'done' : ''}`} key={task.id}>
        <button className="check-button" onClick={() => updateStatus.mutate({ id: task.id, value: task.status === 'completed' ? 'pending' : 'completed' })}>{task.status === 'completed' ? '✓' : ''}</button>
        <div className="task-main"><b>{task.title}</b><small>{task.goal_name}{task.subject_name ? ` · ${task.subject_name}` : ''} · {task.task_type} · 预计 {task.estimated_minutes} 分钟 · 实际 {task.actual_minutes} 分钟</small>{task.description && <p>{task.description}</p>}</div>
        <span className={`badge ${task.priority}`}>{task.status === 'overdue' ? '已逾期' : task.priority === 'high' ? '高优先' : statusLabel[task.status]}</span>
        <select aria-label="任务状态" className="status-select" value={task.status} onChange={e => updateStatus.mutate({ id: task.id, value: e.target.value as Task['status'] })}><option value="pending">待开始</option><option value="in_progress">进行中</option><option value="completed">已完成</option><option value="overdue">已逾期</option><option value="skipped">已跳过</option></select>
        <div className="task-actions"><Link className="ghost-button small" to={`/focus?task_id=${task.id}`}>专注</Link><button className="ghost-button small" onClick={() => openEdit(task)}>编辑</button><button aria-label="删除任务" className="icon-button danger" onClick={() => confirm('确定删除此任务？') && remove.mutate(task.id)}>×</button></div>
      </div>)}</div>
    </Card>}

    {show && <Modal title={editing ? '编辑学习任务' : '新建学习任务'} onClose={closeModal}><form className="form-stack" onSubmit={submit}>
      <label>所属目标<select name="goal_id" required value={formGoalId} onChange={e => { setFormGoalId(e.target.value); setFormSubjectId('') }}><option value="">请选择</option>{goals.data?.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}</select></label>
      <label>任务名称<input name="title" defaultValue={editing?.title} required placeholder="例如：完成高数极限练习 20 题" /></label>
      <div className="form-grid"><label>科目<select name="subject_id" value={formSubjectId} onChange={e => setFormSubjectId(e.target.value)}><option value="">不关联</option>{subjects.data?.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}</select></label><label>阶段<select name="phase_id" defaultValue={editing?.phase_id || ''}><option value="">不关联</option>{phases.data?.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}</select></label></div>
      <label>关联知识点<select name="knowledge_point_id" defaultValue={editing?.knowledge_point_id || ''} disabled={!formSubjectId}><option value="">不关联</option>{knowledge.data?.map(k => <option key={k.id} value={k.id}>{k.parent_id ? '　└ ' : ''}{k.name}</option>)}</select></label>
      <div className="form-grid three"><label>任务类型<select name="task_type" defaultValue={editing?.task_type || 'study'}><option value="study">学习</option><option value="practice">练习</option><option value="review">复习</option><option value="mock">模考</option><option value="vocabulary">词汇</option><option value="other">其他</option></select></label><label>优先级<select name="priority" defaultValue={editing?.priority || 'medium'}><option value="medium">中</option><option value="high">高</option><option value="low">低</option></select></label><label>状态<select name="status" defaultValue={editing?.status || 'pending'}><option value="pending">待开始</option><option value="in_progress">进行中</option><option value="completed">已完成</option><option value="overdue">已逾期</option><option value="skipped">已跳过</option></select></label></div>
      <div className="form-grid three"><label>计划日期<input type="date" name="planned_date" defaultValue={editing?.planned_date || date} required /></label><label>预计分钟<input type="number" name="estimated_minutes" defaultValue={editing?.estimated_minutes || 45} min="0" /></label><label>实际分钟<input type="number" name="actual_minutes" defaultValue={editing?.actual_minutes || 0} min="0" /></label></div>
      <label>说明<textarea name="description" rows={3} defaultValue={editing?.description} placeholder="完成标准、资料页码或注意事项" /></label>
      {save.error && <ErrorBox error={save.error} />}
      <div className="modal-actions"><button type="button" className="ghost-button" onClick={closeModal}>取消</button><button className="primary-button" disabled={save.isPending}>{editing ? '保存修改' : '保存任务'}</button></div>
    </form></Modal>}
  </>
}
