import { FormEvent, useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { Goal, GoalType } from '../types'
import { Card, Empty, ErrorBox, Loading, Modal, PageHeader, Progress, daysUntil, goalTypeLabel, statusLabel, today } from '../components/UI'

const templates: Record<GoalType, { title: string; description: string }> = {
  degree: { title: '学位复习', description: '追踪课程、章节、知识点与期末成绩' },
  postgraduate: { title: '考研', description: '管理政治、英语、数学和专业课复习轮次' },
  ielts: { title: '雅思', description: '追踪听说读写、词汇和单项分数' },
}

function numberValue(form: FormData, name: string) { return Number(form.get(name) || 0) }
function setting(goal: Goal | null, key: string, fallback: string | number = '') { return goal?.settings?.[key] ?? fallback }

export function GoalsPage() {
  const [show, setShow] = useState(false)
  const [editing, setEditing] = useState<Goal | null>(null)
  const [type, setType] = useState<GoalType>('postgraduate')
  const qc = useQueryClient()
  const query = useQuery({ queryKey: ['goals'], queryFn: api.goals })
  const refresh = () => { qc.invalidateQueries({ queryKey: ['goals'] }); qc.invalidateQueries({ queryKey: ['dashboard'] }) }
  const save = useMutation({
    mutationFn: (body: Partial<Goal>) => editing ? api.updateGoal(editing.id, body) : api.createGoal(body),
    onSuccess: () => { refresh(); setShow(false); setEditing(null) },
  })
  const remove = useMutation({ mutationFn: api.deleteGoal, onSuccess: refresh })

  useEffect(() => { if (editing) setType(editing.type) }, [editing])
  function openCreate() { setEditing(null); setType('postgraduate'); setShow(true) }
  function openEdit(goal: Goal) { setEditing(goal); setType(goal.type); setShow(true) }

  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = new FormData(e.currentTarget)
    const targetScore = numberValue(f, 'target_score')
    const settings: Record<string, unknown> = { ...(editing?.settings || {}) }
    if (type === 'postgraduate') {
      Object.assign(settings, {
        school: f.get('school'), major: f.get('major'), daily_minutes: numberValue(f, 'daily_minutes'),
        politics_target: numberValue(f, 'politics_target'), english_target: numberValue(f, 'english_target'),
        math_target: numberValue(f, 'math_target'), professional_target: numberValue(f, 'professional_target'),
      })
    }
    if (type === 'ielts') {
      Object.assign(settings, {
        current_score: numberValue(f, 'current_score'), target_band: targetScore, daily_vocabulary: numberValue(f, 'daily_vocabulary'),
        listening_target: numberValue(f, 'listening_target'), reading_target: numberValue(f, 'reading_target'),
        writing_target: numberValue(f, 'writing_target'), speaking_target: numberValue(f, 'speaking_target'),
      })
    }
    if (type === 'degree') {
      const courses = String(f.get('courses') || '').split(/\r?\n/).map(line => {
        const [name, target] = line.split(/[:：]/)
        return { name: name?.trim(), target_score: Number(target || 0) }
      }).filter(course => course.name)
      Object.assign(settings, { semester: f.get('semester'), weekly_minutes: numberValue(f, 'weekly_minutes'), courses })
    }
    save.mutate({
      name: String(f.get('name')), type, start_date: String(f.get('start_date')), exam_date: String(f.get('exam_date')),
      status: String(f.get('status') || 'active') as Goal['status'], target_score: targetScore,
      progress: editing?.progress || 0, settings,
    })
  }

  if (query.isLoading) return <Loading />
  if (query.error) return <ErrorBox error={query.error} />
  const goals = query.data || []
  const coursesText = Array.isArray(editing?.settings?.courses)
    ? (editing!.settings.courses as Array<{ name: string; target_score?: number }>).map(c => `${c.name}${c.target_score ? `：${c.target_score}` : ''}`).join('\n') : ''

  return <>
    <PageHeader title="我的学习目标" subtitle="用阶段、科目和任务拆解长期考试目标。" action={<button className="primary-button" onClick={openCreate}>＋ 创建目标</button>} />
    {goals.length === 0 ? <Card><Empty text="还没有学习目标，创建考研、雅思或学位复习计划" /><div className="center-actions"><button className="primary-button" onClick={openCreate}>创建第一个目标</button></div></Card> : <div className="goal-grid">
      {goals.map(goal => <article className="goal-card card" key={goal.id}>
        <Link to={`/goals/${goal.id}`} className="goal-card-link">
          <div className="goal-top"><span className={`goal-type ${goal.type}`}>{goalTypeLabel[goal.type]}</span><span className={`badge ${goal.status}`}>{statusLabel[goal.status] || goal.status}</span></div>
          <h2>{goal.name}</h2><p>考试日期 {goal.exam_date}</p>
          <div className="goal-count"><strong>{daysUntil(goal.exam_date)}</strong><span>天后考试</span></div>
          <div className="progress-head"><span>总体进度</span><b>{Math.round(goal.progress)}%</b></div><Progress value={goal.progress} />
        </Link>
        <div className="row-actions"><button className="ghost-button small" onClick={() => openEdit(goal)}>编辑</button><button className="ghost-button small danger" onClick={() => confirm(`确定删除“${goal.name}”及其全部学习数据？`) && remove.mutate(goal.id)}>删除</button></div>
      </article>)}
    </div>}

    {show && <Modal title={editing ? '编辑学习目标' : '创建学习目标'} onClose={() => { setShow(false); setEditing(null) }}><form className="form-stack" onSubmit={submit}>
      <div className="template-grid">{(Object.keys(templates) as GoalType[]).map(key => <button type="button" disabled={!!editing} className={`template-card ${type === key ? 'selected' : ''}`} onClick={() => setType(key)} key={key}><b>{templates[key].title}</b><small>{templates[key].description}</small></button>)}</div>
      <label>目标名称<input name="name" defaultValue={editing?.name} placeholder={type === 'postgraduate' ? '2027 计算机考研' : type === 'ielts' ? '雅思 7 分计划' : '本学期期末复习'} required /></label>
      <div className="form-grid"><label>开始日期<input name="start_date" type="date" defaultValue={editing?.start_date || today()} required /></label><label>考试日期<input name="exam_date" type="date" min={editing?.start_date || today()} defaultValue={editing?.exam_date} required /></label></div>
      {editing && <label>状态<select name="status" defaultValue={editing.status}><option value="active">进行中</option><option value="paused">已暂停</option><option value="completed">已完成</option><option value="archived">已归档</option></select></label>}
      {type === 'postgraduate' && <>
        <div className="form-grid"><label>目标院校<input name="school" defaultValue={String(setting(editing, 'school'))} placeholder="XX大学" /></label><label>目标专业<input name="major" defaultValue={String(setting(editing, 'major'))} placeholder="计算机科学与技术" /></label></div>
        <div className="form-grid three"><label>政治目标<input name="politics_target" type="number" defaultValue={Number(setting(editing, 'politics_target', 70))} /></label><label>英语目标<input name="english_target" type="number" defaultValue={Number(setting(editing, 'english_target', 75))} /></label><label>数学目标<input name="math_target" type="number" defaultValue={Number(setting(editing, 'math_target', 120))} /></label></div>
        <div className="form-grid"><label>专业课目标<input name="professional_target" type="number" defaultValue={Number(setting(editing, 'professional_target', 125))} /></label><label>每日可用时间（分钟）<input name="daily_minutes" type="number" min="0" defaultValue={Number(setting(editing, 'daily_minutes', 360))} /></label></div>
      </>}
      {type === 'degree' && <><div className="form-grid"><label>学期<input name="semester" defaultValue={String(setting(editing, 'semester'))} placeholder="2026 秋季" /></label><label>每周可用时间（分钟）<input name="weekly_minutes" type="number" min="0" defaultValue={Number(setting(editing, 'weekly_minutes', 1200))} /></label></div><label>课程与目标成绩<textarea name="courses" rows={4} defaultValue={coursesText} placeholder={'高等数学：90\n大学英语：85'} /><small>每行一门课程，可在冒号后填写目标成绩。</small></label></>}
      {type === 'ielts' && <>
        <div className="form-grid"><label>当前总分<input name="current_score" type="number" step="0.5" defaultValue={Number(setting(editing, 'current_score', 5.5))} /></label><label>目标总分<input name="target_score" type="number" step="0.5" defaultValue={editing?.target_score || 7} /></label></div>
        <div className="form-grid"><label>听力目标<input name="listening_target" type="number" step="0.5" defaultValue={Number(setting(editing, 'listening_target', 7))} /></label><label>阅读目标<input name="reading_target" type="number" step="0.5" defaultValue={Number(setting(editing, 'reading_target', 7))} /></label></div>
        <div className="form-grid"><label>写作目标<input name="writing_target" type="number" step="0.5" defaultValue={Number(setting(editing, 'writing_target', 6.5))} /></label><label>口语目标<input name="speaking_target" type="number" step="0.5" defaultValue={Number(setting(editing, 'speaking_target', 6.5))} /></label></div>
        <label>每日单词量<input name="daily_vocabulary" type="number" min="0" defaultValue={Number(setting(editing, 'daily_vocabulary', 80))} /></label>
      </>}
      {type !== 'ielts' && <label>目标总分<input name="target_score" type="number" defaultValue={editing?.target_score || (type === 'postgraduate' ? 390 : 85)} /></label>}
      {save.error && <ErrorBox error={save.error} />}
      <div className="modal-actions"><button type="button" className="ghost-button" onClick={() => { setShow(false); setEditing(null) }}>取消</button><button className="primary-button" disabled={save.isPending}>{editing ? '保存修改' : '创建并生成模板'}</button></div>
    </form></Modal>}
  </>
}
