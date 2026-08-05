import { FormEvent, useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Assessment } from '../types'
import { Card, Empty, ErrorBox, MiniBars, Modal, PageHeader, today } from '../components/UI'

function detailNumber(assessment: Assessment | null, key: string, fallback = 0) { return Number(assessment?.details?.[key] ?? fallback) }

export function AssessmentsPage() {
  const [params] = useSearchParams()
  const [goalId, setGoalId] = useState(params.get('goal_id') || '')
  const [show, setShow] = useState(false)
  const [editing, setEditing] = useState<Assessment | null>(null)
  const [assessmentType, setAssessmentType] = useState<Assessment['assessment_type']>('practice')
  const qc = useQueryClient()
  const goals = useQuery({ queryKey: ['goals'], queryFn: api.goals })
  const subjects = useQuery({ queryKey: ['subjects', goalId], queryFn: () => api.subjects(goalId), enabled: !!goalId })
  const assessments = useQuery({ queryKey: ['assessments', goalId], queryFn: () => api.assessments(goalId || undefined) })
  useEffect(() => { if (!goalId && goals.data?.length) setGoalId(goals.data[0].id) }, [goalId, goals.data])
  const refresh = () => { qc.invalidateQueries({ queryKey: ['assessments'] }); qc.invalidateQueries({ queryKey: ['dashboard'] }); qc.invalidateQueries({ queryKey: ['analytics'] }) }
  const save = useMutation({ mutationFn: (body: Partial<Assessment>) => editing ? api.updateAssessment(editing.id, body) : api.createAssessment(body), onSuccess: () => { refresh(); closeModal() } })
  const remove = useMutation({ mutationFn: api.deleteAssessment, onSuccess: refresh })
  const trend = useMemo(() => (assessments.data || []).slice().reverse().map(a => ({ date: a.exam_date.slice(5), value: a.full_score ? Math.round(a.score / a.full_score * 100) : a.score })), [assessments.data])

  function openCreate() { setEditing(null); setAssessmentType('practice'); setShow(true) }
  function openEdit(item: Assessment) { setEditing(item); setAssessmentType(item.assessment_type); setGoalId(item.goal_id); setShow(true) }
  function closeModal() { setShow(false); setEditing(null) }
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault(); const f = new FormData(e.currentTarget)
    const details: Record<string, unknown> = {}
    if (assessmentType === 'ielts') {
      details.listening = Number(f.get('listening') || 0); details.reading = Number(f.get('reading') || 0)
      details.writing = Number(f.get('writing') || 0); details.speaking = Number(f.get('speaking') || 0)
    } else {
      const note = String(f.get('note') || ''); if (note) details.note = note
      const sectionScores = String(f.get('section_scores') || ''); if (sectionScores) details.section_scores = sectionScores
    }
    save.mutate({
      goal_id: String(f.get('goal_id')), subject_id: String(f.get('subject_id') || '') || null, name: String(f.get('name')),
      assessment_type: assessmentType, exam_date: String(f.get('exam_date')), score: Number(f.get('score')),
      full_score: Number(f.get('full_score')), duration_minutes: Number(f.get('duration_minutes') || 0), details,
    })
  }
  const label = (type: Assessment['assessment_type']) => type === 'mock' ? '模考' : type === 'ielts' ? '雅思' : type === 'exam' ? '正式考试' : '练习'

  return <>
    <PageHeader title="模考与成绩" subtitle="记录练习、模考和正式考试，用趋势检验学习效果。" action={<button className="primary-button" onClick={openCreate} disabled={!goals.data?.length}>＋ 记录成绩</button>} />
    <Card className="filter-card"><label>学习目标<select value={goalId} onChange={e => setGoalId(e.target.value)}><option value="">全部目标</option>{goals.data?.map(g => <option value={g.id} key={g.id}>{g.name}</option>)}</select></label><div className="filter-summary"><b>{assessments.data?.length || 0}</b><span>次成绩记录</span></div></Card>
    <div className="two-column assessment-layout">
      <Card><div className="card-title"><div><h2>成绩趋势</h2><p>统一折算为百分比便于比较</p></div></div>{trend.length ? <MiniBars data={trend.slice(-12)} /> : <Empty />}</Card>
      <Card><div className="card-title"><div><h2>关键指标</h2><p>基于当前筛选结果</p></div></div>{(assessments.data || []).length ? <div className="metric-stack"><div><span>最近成绩</span><strong>{assessments.data![0].score} / {assessments.data![0].full_score}</strong></div><div><span>最高得分率</span><strong>{Math.max(...assessments.data!.map(a => a.full_score ? a.score / a.full_score * 100 : 0)).toFixed(1)}%</strong></div><div><span>平均用时</span><strong>{Math.round(assessments.data!.reduce((s, a) => s + a.duration_minutes, 0) / assessments.data!.length)} 分钟</strong></div></div> : <Empty />}</Card>
    </div>
    {assessments.isLoading ? <div className="center-screen"><div className="spinner" /></div> : assessments.error ? <ErrorBox error={assessments.error} /> : (assessments.data || []).length === 0 ? <Card><Empty text="还没有成绩记录" /></Card> : <Card className="table-card"><div className="data-table assessment-table head"><span>考试</span><span>日期</span><span>类型</span><span>成绩</span><span>得分率</span><span>用时</span><span /></div>{assessments.data?.map(a => <div className="data-table assessment-table row" key={a.id}><div><b>{a.name}</b><small>{a.subject_id ? subjects.data?.find(s => s.id === a.subject_id)?.name : '综合'}{a.assessment_type === 'ielts' ? ` · 听${a.details.listening ?? '-'} 读${a.details.reading ?? '-'} 写${a.details.writing ?? '-'} 说${a.details.speaking ?? '-'}` : a.details.section_scores ? ` · ${String(a.details.section_scores)}` : ''}</small></div><span>{a.exam_date}</span><span className="badge">{label(a.assessment_type)}</span><strong>{a.score} / {a.full_score}</strong><span>{a.full_score ? (a.score / a.full_score * 100).toFixed(1) : 0}%</span><span>{a.duration_minutes} 分钟</span><div className="row-actions"><button className="ghost-button small" onClick={() => openEdit(a)}>编辑</button><button aria-label="删除成绩" className="icon-button danger" onClick={() => confirm('确定删除此记录？') && remove.mutate(a.id)}>×</button></div></div>)}</Card>}

    {show && <Modal title={editing ? '编辑考试成绩' : '记录考试成绩'} onClose={closeModal}><form className="form-stack" onSubmit={submit}>
      <label>学习目标<select name="goal_id" value={goalId} onChange={e => setGoalId(e.target.value)} required><option value="">请选择</option>{goals.data?.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}</select></label>
      <div className="form-grid"><label>考试名称<input name="name" defaultValue={editing?.name} required placeholder="例如：数学模拟卷 3" /></label><label>科目<select name="subject_id" defaultValue={editing?.subject_id || ''}><option value="">综合考试</option>{subjects.data?.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}</select></label></div>
      <div className="form-grid"><label>类型<select name="assessment_type" value={assessmentType} onChange={e => setAssessmentType(e.target.value as Assessment['assessment_type'])}><option value="practice">练习</option><option value="mock">模考</option><option value="exam">正式考试</option><option value="ielts">雅思</option></select></label><label>日期<input name="exam_date" type="date" defaultValue={editing?.exam_date || today()} required /></label></div>
      <div className="form-grid three"><label>{assessmentType === 'ielts' ? '总分' : '得分'}<input name="score" type="number" step="0.5" defaultValue={editing?.score} required /></label><label>满分<input name="full_score" type="number" step="0.5" defaultValue={editing?.full_score || (assessmentType === 'ielts' ? 9 : 100)} required /></label><label>用时（分钟）<input name="duration_minutes" type="number" min="0" defaultValue={editing?.duration_minutes || 120} /></label></div>
      {assessmentType === 'ielts' ? <div className="form-grid"><label>听力<input name="listening" type="number" min="0" max="9" step="0.5" defaultValue={detailNumber(editing, 'listening')} /></label><label>阅读<input name="reading" type="number" min="0" max="9" step="0.5" defaultValue={detailNumber(editing, 'reading')} /></label><label>写作<input name="writing" type="number" min="0" max="9" step="0.5" defaultValue={detailNumber(editing, 'writing')} /></label><label>口语<input name="speaking" type="number" min="0" max="9" step="0.5" defaultValue={detailNumber(editing, 'speaking')} /></label></div> : <><label>分项成绩<input name="section_scores" defaultValue={String(editing?.details?.section_scores || '')} placeholder="例如：选择 42/50，填空 24/30，解答 42/70" /></label><label>备注<textarea name="note" rows={3} defaultValue={String(editing?.details?.note || '')} placeholder="错题分布、状态或复盘结论" /></label></>}
      {save.error && <ErrorBox error={save.error} />}
      <div className="modal-actions"><button type="button" className="ghost-button" onClick={closeModal}>取消</button><button className="primary-button" disabled={save.isPending}>{editing ? '保存修改' : '保存成绩'}</button></div>
    </form></Modal>}
  </>
}
