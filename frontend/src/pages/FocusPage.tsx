import { FormEvent, useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { api } from '../api/client'
import type { StudySession } from '../types'
import { Card, Empty, ErrorBox, Modal, PageHeader, minutesText, today } from '../components/UI'

interface ActiveFocus {
  session: StudySession
  taskTitle: string
  startedAt: number
  elapsedSeconds: number
  notes?: string
}

const storageKey = 'studytrack_active_focus'

export function FocusPage() {
  const qc = useQueryClient()
  const [params] = useSearchParams()
  const [active, setActive] = useState<ActiveFocus | null>(() => {
    try { return JSON.parse(localStorage.getItem(storageKey) || 'null') } catch { return null }
  })
  const [seconds, setSeconds] = useState(() => active?.elapsedSeconds ?? (active ? Math.max(0, Math.floor((Date.now() - active.startedAt) / 1000)) : 0))
  const [paused, setPaused] = useState(false)
  const [showFinish, setShowFinish] = useState(false)
  const [goalId, setGoalId] = useState('')
  const [taskId, setTaskId] = useState(params.get('task_id') || '')
  const [subjectId, setSubjectId] = useState('')
  const goals = useQuery({ queryKey: ['goals'], queryFn: api.goals })
  const tasks = useQuery({ queryKey: ['focus-tasks'], queryFn: () => api.tasks({}) })
  const subjects = useQuery({ queryKey: ['subjects', goalId], queryFn: () => api.subjects(goalId), enabled: !!goalId && !taskId })
  const sessions = useQuery({ queryKey: ['sessions'], queryFn: () => api.sessions(), enabled: !active })
  const dashboard = useQuery({ queryKey: ['dashboard'], queryFn: () => api.dashboard() })
  const start = useMutation({
    mutationFn: ({ taskId }: { taskId?: string }) => {
      const task = tasks.data?.find(t => t.id === taskId)
      return api.startSession({ goal_id: task?.goal_id || goalId, task_id: task?.id || null, subject_id: task?.subject_id || subjectId || null })
    },
    onSuccess: (session, vars) => {
      const task = tasks.data?.find(t => t.id === vars.taskId)
      const value = { session, taskTitle: task?.title || '自由学习', startedAt: Date.now(), elapsedSeconds: 0, notes: '' }
      localStorage.setItem(storageKey, JSON.stringify(value))
      setActive(value); setSeconds(0)
    },
  })
  const finish = useMutation({
    mutationFn: (body: Partial<StudySession>) => api.finishSession(active!.session.id, body),
    onSuccess: () => {
      localStorage.removeItem(storageKey); setActive(null); setSeconds(0); setShowFinish(false)
      qc.invalidateQueries({ queryKey: ['dashboard'] }); qc.invalidateQueries({ queryKey: ['sessions'] }); qc.invalidateQueries({ queryKey: ['tasks'] })
    },
  })

  useEffect(() => {
    if (!active || paused) return
    const id = window.setInterval(() => {
      setSeconds((previous) => {
        const next = previous + 1
        if (next % 5 === 0) localStorage.setItem(storageKey, JSON.stringify({ ...active, elapsedSeconds: next }))
        return next
      })
    }, 1000)
    return () => clearInterval(id)
  }, [active, paused])
  useEffect(() => {
    const selected = tasks.data?.find(t => t.id === taskId)
    if (selected) setGoalId(selected.goal_id)
  }, [taskId, tasks.data])

  const timeText = useMemo(() => {
    const h = Math.floor(seconds / 3600).toString().padStart(2, '0')
    const m = Math.floor((seconds % 3600) / 60).toString().padStart(2, '0')
    const s = (seconds % 60).toString().padStart(2, '0')
    return `${h}:${m}:${s}`
  }, [seconds])

  function begin(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = new FormData(e.currentTarget)
    const selectedTaskId = String(f.get('task_id') || '') || undefined
    if (!selectedTaskId && !goalId) return
    start.mutate({ taskId: selectedTaskId })
  }

  function complete(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = new FormData(e.currentTarget)
    finish.mutate({
      duration_minutes: Math.max(1, Number(f.get('duration_minutes') || Math.round(seconds / 60))),
      focus_score: Number(f.get('focus_score')),
      mastery_score: Number(f.get('mastery_score')),
      notes: String(f.get('notes') || ''),
      ended_at: new Date().toISOString(),
    })
  }
  function updateNotes(notes: string) {
    if (!active) return
    const next = { ...active, notes, elapsedSeconds: seconds }
    setActive(next); localStorage.setItem(storageKey, JSON.stringify(next))
  }

  return <>
    <PageHeader title="专注学习" subtitle="一次只做一件事，结束后记录专注度和掌握情况。" />
    {active ? <div className="focus-layout">
      <Card className="focus-card">
        <span className="focus-label">正在学习</span>
        <h2>{active.taskTitle}</h2>
        <div className="focus-clock">{timeText}</div>
        <div className="focus-meta"><span>今日累计 {minutesText((dashboard.data?.today_minutes || 0) + Math.round(seconds / 60))}</span><span>预计 {tasks.data?.find(t => t.id === active.session.task_id)?.estimated_minutes || '-'} 分钟</span></div>
        <label className="focus-notes">临时笔记<textarea rows={3} value={active.notes || ''} onChange={e => updateNotes(e.target.value)} placeholder="随手记下问题和下一步行动" /></label>
        <div className="focus-actions"><button className="ghost-button" onClick={() => { localStorage.setItem(storageKey, JSON.stringify({ ...active, elapsedSeconds: seconds })); setPaused(!paused) }}>{paused ? '继续' : '暂停'}</button><button className="primary-button" onClick={() => setShowFinish(true)}>完成学习</button></div>
        <p>{paused ? '计时已暂停' : '保持专注，完成后再处理其他事项。'}</p>
      </Card>
      <Card><div className="card-title"><div><h2>专注提示</h2><p>降低中断，提高有效学习时间</p></div></div><ul className="tip-list"><li>关闭无关通知和网页。</li><li>遇到问题先写在临时笔记中。</li><li>每 45–60 分钟安排短暂休息。</li><li>结束时如实评估掌握程度。</li></ul></Card>
    </div> : <div className="two-column focus-start-layout">
      <Card><div className="card-title"><div><h2>开始一次专注</h2><p>选择今日任务，或进行自由学习</p></div></div>
        <form className="form-stack" onSubmit={begin}>
          <label>学习任务<select name="task_id" value={taskId} onChange={e => { setTaskId(e.target.value); setSubjectId('') }}><option value="">自由学习</option>{tasks.data?.filter(t => t.status !== 'completed' && t.status !== 'skipped').map(t => <option key={t.id} value={t.id}>{t.planned_date === today() ? '今日 · ' : ''}{t.title} · {t.estimated_minutes} 分钟</option>)}</select></label>
          <label>自由学习所属目标<select value={goalId} disabled={!!taskId} onChange={e => { setGoalId(e.target.value); setSubjectId('') }}><option value="">请选择</option>{goals.data?.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}</select></label>
          {!taskId && <label>学习科目<select value={subjectId} onChange={e => setSubjectId(e.target.value)}><option value="">不关联科目</option>{subjects.data?.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}</select></label>}
          {start.error && <ErrorBox error={start.error} />}
          <button className="primary-button full-button" disabled={start.isPending || (!goalId && !taskId)}>开始计时</button>
        </form>
      </Card>
      <Card><div className="card-title"><div><h2>最近学习记录</h2><p>回顾投入与专注状态</p></div></div>{(sessions.data || []).length === 0 ? <Empty text="完成专注后显示记录" /> : <div className="session-list">{sessions.data?.slice(0, 8).map(s => <div key={s.id}><div><b>{new Date(s.started_at).toLocaleDateString('zh-CN')}</b><small>专注 {s.focus_score || '-'} / 5 · 掌握 {s.mastery_score || '-'} / 5</small></div><strong>{minutesText(s.duration_minutes)}</strong></div>)}</div>}</Card>
    </div>}

    {showFinish && <Modal title="完成本次学习" onClose={() => setShowFinish(false)}><form className="form-stack" onSubmit={complete}>
      <label>实际学习时间（分钟）<input name="duration_minutes" type="number" min="1" defaultValue={Math.max(1, Math.round(seconds / 60))} required /><small>已自动计算，可按实际情况修正。</small></label>
      <label>专注程度<select name="focus_score" defaultValue="4"><option value="1">1 - 经常分心</option><option value="2">2 - 较多中断</option><option value="3">3 - 基本专注</option><option value="4">4 - 很专注</option><option value="5">5 - 深度专注</option></select></label>
      <label>掌握程度<select name="mastery_score" defaultValue="3"><option value="1">1 - 仍不理解</option><option value="2">2 - 有些模糊</option><option value="3">3 - 基本掌握</option><option value="4">4 - 熟练掌握</option><option value="5">5 - 可以讲解</option></select></label>
      <label>学习笔记<textarea name="notes" rows={4} defaultValue={active?.notes || ''} placeholder="遇到的问题、需要复习的内容或下一步行动" /></label>
      {finish.error && <ErrorBox error={finish.error} />}
      <div className="modal-actions"><button type="button" className="ghost-button" onClick={() => setShowFinish(false)}>继续学习</button><button className="primary-button" disabled={finish.isPending}>保存并结束</button></div>
    </form></Modal>}
  </>
}
