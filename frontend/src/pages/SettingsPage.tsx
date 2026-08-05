import { FormEvent } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api } from '../api/client'
import { Card, ErrorBox, PageHeader } from '../components/UI'
import { useAuthStore } from '../store/auth'

export function SettingsPage() {
  const user = useAuthStore(s => s.user)
  const setUser = useAuthStore(s => s.setUser)
  const update = useMutation({ mutationFn: api.updateMe, onSuccess: setUser })
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault(); const f = new FormData(e.currentTarget)
    update.mutate({ nickname: String(f.get('nickname')), timezone: String(f.get('timezone')) })
  }
  return <>
    <PageHeader title="系统设置" subtitle="管理个人资料、时区和应用偏好。" />
    <div className="two-column settings-layout">
      <Card><div className="card-title"><div><h2>个人资料</h2><p>用于学习记录和时间统计</p></div></div><form className="form-stack" onSubmit={submit}>
        <label>邮箱<input value={user?.email || ''} disabled /></label>
        <label>昵称<input name="nickname" defaultValue={user?.nickname} required /></label>
        <label>时区<select name="timezone" defaultValue={user?.timezone || 'Asia/Shanghai'}><option value="Asia/Shanghai">中国标准时间</option><option value="Asia/Hong_Kong">香港时间</option><option value="Asia/Tokyo">日本时间</option><option value="America/Los_Angeles">美国太平洋时间</option><option value="Europe/London">英国时间</option></select></label>
        {update.error && <ErrorBox error={update.error} />}{update.isSuccess && <div className="success-box">资料已更新</div>}
        <button className="primary-button">保存设置</button>
      </form></Card>
      <Card><div className="card-title"><div><h2>产品说明</h2><p>当前版本能力</p></div></div><div className="about-list"><div><b>版本</b><span>1.0.0 MVP</span></div><div><b>前端</b><span>React + TypeScript + Vite</span></div><div><b>后端</b><span>Go REST API</span></div><div><b>数据库</b><span>PostgreSQL</span></div><div><b>数据范围</b><span>目标、任务、专注、知识点、成绩、复盘</span></div></div></Card>
    </div>
  </>
}
