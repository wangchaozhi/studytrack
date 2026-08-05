import { FormEvent, useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { api } from '../api/client'
import type { KnowledgePoint } from '../types'
import { Card, Empty, ErrorBox, Loading, Modal, PageHeader, Progress } from '../components/UI'

function flatten(points: KnowledgePoint[]) {
  const children = new Map<string | null, KnowledgePoint[]>()
  points.forEach(point => { const key = point.parent_id || null; children.set(key, [...(children.get(key) || []), point]) })
  const result: Array<{ point: KnowledgePoint; depth: number }> = []
  const visit = (parent: string | null, depth: number) => (children.get(parent) || []).forEach(point => { result.push({ point, depth }); visit(point.id, depth + 1) })
  visit(null, 0)
  return result.length === points.length ? result : points.map(point => ({ point, depth: 0 }))
}
function dateTimeValue(value?: string | null) { return value ? new Date(value).toISOString().slice(0, 16) : '' }

export function KnowledgePage() {
  const [params] = useSearchParams()
  const [goalId, setGoalId] = useState(params.get('goal_id') || '')
  const [subjectId, setSubjectId] = useState('')
  const [show, setShow] = useState(false)
  const [editing, setEditing] = useState<KnowledgePoint | null>(null)
  const [reviewId, setReviewId] = useState<string | null>(null)
  const qc = useQueryClient()
  const goals = useQuery({ queryKey: ['goals'], queryFn: api.goals })
  const subjects = useQuery({ queryKey: ['subjects', goalId], queryFn: () => api.subjects(goalId), enabled: !!goalId })
  const knowledge = useQuery({ queryKey: ['knowledge', subjectId], queryFn: () => api.knowledge(subjectId), enabled: !!subjectId })

  useEffect(() => { setSubjectId('') }, [goalId])
  useEffect(() => { if (!goalId && goals.data?.length) setGoalId(goals.data[0].id) }, [goalId, goals.data])
  useEffect(() => { if (!subjectId && subjects.data?.length) setSubjectId(subjects.data[0].id) }, [subjectId, subjects.data])
  const refresh = () => { qc.invalidateQueries({ queryKey: ['knowledge', subjectId] }); qc.invalidateQueries({ queryKey: ['dashboard'] }); qc.invalidateQueries({ queryKey: ['analytics'] }) }
  const save = useMutation({ mutationFn: (body: Partial<KnowledgePoint>) => editing ? api.updateKnowledge(editing.id, body) : api.createKnowledge(body), onSuccess: () => { refresh(); setShow(false); setEditing(null) } })
  const review = useMutation({ mutationFn: ({ id, mastery, wrong }: { id: string; mastery: number; wrong: number }) => api.reviewKnowledge(id, mastery, wrong), onSuccess: () => { refresh(); setReviewId(null) } })

  const points = knowledge.data || []
  const ordered = useMemo(() => flatten(points), [points])
  const selected = points.find(x => x.id === reviewId)
  function openCreate() { setEditing(null); setShow(true) }
  function openEdit(point: KnowledgePoint) { setEditing(point); setShow(true) }
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault(); const f = new FormData(e.currentTarget); const next = String(f.get('next_review_at') || '')
    save.mutate({
      subject_id: subjectId, parent_id: String(f.get('parent_id') || '') || null, name: String(f.get('name')),
      importance: String(f.get('importance')) as KnowledgePoint['importance'], mastery: Number(f.get('mastery') || 0),
      review_status: String(f.get('review_status')) as KnowledgePoint['review_status'], next_review_at: next ? new Date(next).toISOString() : null,
      review_count: editing?.review_count || 0, wrong_count: Number(f.get('wrong_count') || 0),
    })
  }
  function submitReview(e: FormEvent<HTMLFormElement>) {
    e.preventDefault(); const f = new FormData(e.currentTarget)
    if (reviewId) review.mutate({ id: reviewId, mastery: Number(f.get('mastery')), wrong: Number(f.get('wrong_delta') || 0) })
  }

  if (goals.isLoading) return <Loading />
  return <>
    <PageHeader title="知识点管理" subtitle="用掌握度、错题数和复习次数定位薄弱环节。" action={<button className="primary-button" onClick={openCreate} disabled={!subjectId}>＋ 添加知识点</button>} />
    <Card className="filter-card"><label>学习目标<select value={goalId} onChange={e => setGoalId(e.target.value)}><option value="">请选择</option>{goals.data?.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}</select></label><label>科目<select value={subjectId} onChange={e => setSubjectId(e.target.value)}><option value="">请选择</option>{subjects.data?.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}</select></label><div className="filter-summary"><b>{points.length}</b><span>个知识点</span></div></Card>
    {!subjectId ? <Card><Empty text="先选择学习目标和科目" /></Card> : knowledge.isLoading ? <Loading /> : knowledge.error ? <ErrorBox error={knowledge.error} /> : points.length === 0 ? <Card><Empty text="这个科目还没有知识点" /><div className="center-actions"><button className="primary-button" onClick={openCreate}>添加第一个知识点</button></div></Card> : <Card>
      <div className="knowledge-table head"><span>知识点</span><span>重要度</span><span>错题</span><span>复习</span><span>掌握度</span><span /></div>
      {ordered.map(({ point, depth }) => <div className="knowledge-table row" key={point.id}>
        <div style={{ paddingLeft: `${depth * 20}px` }}><span className="knowledge-icon">{depth ? '└' : '◇'}</span><b>{point.name}</b><small>{point.review_status === 'mastered' ? '已掌握' : point.review_status === 'learning' ? '学习中' : point.review_status === 'review' ? '需复习' : '未学习'}{point.next_review_at ? ` · ${new Date(point.next_review_at).toLocaleDateString('zh-CN')}复习` : ''}</small></div>
        <span className={`badge ${point.importance}`}>{point.importance === 'core' ? '核心' : point.importance === 'important' ? '重点' : '普通'}</span><span>{point.wrong_count}</span><span>{point.review_count}</span>
        <div><div className="progress-head"><b>{Math.round(point.mastery)}%</b></div><Progress value={point.mastery} /></div>
        <div className="row-actions"><button className="ghost-button small" onClick={() => setReviewId(point.id)}>复习</button><button className="ghost-button small" onClick={() => openEdit(point)}>编辑</button></div>
      </div>)}
    </Card>}

    {show && <Modal title={editing ? `编辑：${editing.name}` : '添加知识点'} onClose={() => { setShow(false); setEditing(null) }}><form className="form-stack" onSubmit={submit}>
      <label>知识点名称<input name="name" defaultValue={editing?.name} required placeholder="例如：函数极限" /></label>
      <label>上级知识点<select name="parent_id" defaultValue={editing?.parent_id || ''}><option value="">作为一级知识点</option>{points.filter(x => x.id !== editing?.id).map(x => <option key={x.id} value={x.id}>{x.name}</option>)}</select></label>
      <div className="form-grid"><label>重要程度<select name="importance" defaultValue={editing?.importance || 'normal'}><option value="normal">普通</option><option value="important">重点</option><option value="core">核心</option></select></label><label>掌握度<input name="mastery" type="number" min="0" max="100" defaultValue={editing?.mastery || 0} /></label></div>
      <div className="form-grid"><label>复习状态<select name="review_status" defaultValue={editing?.review_status || 'unlearned'}><option value="unlearned">未学习</option><option value="learning">学习中</option><option value="mastered">已掌握</option><option value="review">需复习</option></select></label><label>错题数<input name="wrong_count" type="number" min="0" defaultValue={editing?.wrong_count || 0} /></label></div>
      <label>下次复习时间<input name="next_review_at" type="datetime-local" defaultValue={dateTimeValue(editing?.next_review_at)} /></label>
      {save.error && <ErrorBox error={save.error} />}
      <div className="modal-actions"><button type="button" className="ghost-button" onClick={() => { setShow(false); setEditing(null) }}>取消</button><button className="primary-button">{editing ? '保存修改' : '保存'}</button></div>
    </form></Modal>}

    {selected && <Modal title={`复习记录：${selected.name}`} onClose={() => setReviewId(null)}><form className="form-stack" onSubmit={submitReview}>
      <div className="finish-summary"><span>当前掌握度</span><strong>{Math.round(selected.mastery)}%</strong></div>
      <label>本次评估掌握度<input name="mastery" type="range" min="0" max="100" defaultValue={Math.round(selected.mastery)} onInput={e => { const output = e.currentTarget.nextElementSibling; if (output) output.textContent = `${e.currentTarget.value}%` }} /><output>{Math.round(selected.mastery)}%</output></label>
      <label>新增错题数<input name="wrong_delta" type="number" min={-selected.wrong_count} defaultValue="0" /><small>订正错题可填写负数；下次复习时间会根据掌握度自动安排，也可随后编辑。</small></label>
      {review.error && <ErrorBox error={review.error} />}
      <div className="modal-actions"><button type="button" className="ghost-button" onClick={() => setReviewId(null)}>取消</button><button className="primary-button">保存复习结果</button></div>
    </form></Modal>}
  </>
}
