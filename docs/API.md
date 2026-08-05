# StudyTrack REST API

API 前缀：`/api/v1`

除注册和登录外，请求需要：

```http
Authorization: Bearer <token>
Content-Type: application/json
```

错误格式：

```json
{ "error": "错误说明" }
```

## 认证与用户

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/auth/register` | 注册 |
| POST | `/auth/login` | 登录 |
| POST | `/auth/refresh` | 刷新令牌 |
| GET | `/users/me` | 当前用户 |
| PUT | `/users/me` | 更新昵称和时区 |

注册示例：

```json
{
  "email": "student@example.com",
  "password": "password123",
  "nickname": "学习者",
  "timezone": "Asia/Shanghai"
}
```

## 首页

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/dashboard` | 全部目标首页数据 |
| GET | `/dashboard?goal_id=<uuid>` | 指定目标首页数据 |
| GET | `/goals/:id/dashboard` | 指定目标首页数据 |

## 学习目标

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/goals` | 目标列表 |
| POST | `/goals` | 创建目标并生成模板 |
| GET | `/goals/:id` | 目标详情 |
| PUT | `/goals/:id` | 更新目标 |
| DELETE | `/goals/:id` | 删除目标和关联数据 |

创建考研目标：

```json
{
  "name": "2027 计算机考研",
  "type": "postgraduate",
  "start_date": "2026-08-05",
  "exam_date": "2026-12-26",
  "status": "active",
  "target_score": 390,
  "progress": 0,
  "settings": {
    "school": "XX大学",
    "major": "计算机科学与技术",
    "daily_minutes": 360,
    "politics_target": 70,
    "english_target": 75,
    "math_target": 120,
    "professional_target": 125
  }
}
```

目标类型：`degree`、`postgraduate`、`ielts`。

`degree` 设置可包含 `semester`、`weekly_minutes` 和 `courses`；`ielts` 设置可包含当前/目标总分、听说读写目标分数和 `daily_vocabulary`。创建目标时，这些设置会用于生成默认科目及其目标分数。

## 阶段与科目

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/goals/:id/phases` | 阶段列表 |
| POST | `/goals/:id/phases` | 新增阶段 |
| GET | `/goals/:id/subjects` | 科目列表 |
| POST | `/goals/:id/subjects` | 新增科目 |
| PUT | `/subjects/:id` | 更新科目 |

## 学习任务

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/tasks` | 任务列表 |
| POST | `/tasks` | 新建任务 |
| PUT | `/tasks/:id` | 更新任务 |
| PATCH | `/tasks/:id/status` | 更新状态 |
| DELETE | `/tasks/:id` | 删除任务 |

查询参数：`date`、`goal_id`、`status`、`from`、`to`。

任务示例：

```json
{
  "goal_id": "goal-uuid",
  "subject_id": "subject-uuid",
  "title": "完成高数极限练习 20 题",
  "description": "订正并关联错题知识点",
  "task_type": "practice",
  "priority": "high",
  "status": "pending",
  "planned_date": "2026-08-05",
  "estimated_minutes": 60,
  "actual_minutes": 0
}
```

任务状态：`pending`、`in_progress`、`completed`、`overdue`、`skipped`。

## 专注学习

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/study-sessions/start` | 开始学习 |
| POST | `/study-sessions/:id/finish` | 完成学习 |
| GET | `/study-sessions` | 学习记录 |

完成示例：

```json
{
  "duration_minutes": 52,
  "focus_score": 4,
  "mastery_score": 3,
  "notes": "极限定义仍需复习",
  "ended_at": "2026-08-05T10:30:00Z"
}
```

## 知识点

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/subjects/:id/knowledge-points` | 科目知识点 |
| POST | `/knowledge-points` | 新增知识点 |
| PUT | `/knowledge-points/:id` | 更新知识点 |
| POST | `/knowledge-points/:id/review` | 记录复习结果 |

复习示例：

```json
{
  "mastery": 75,
  "wrong_delta": -2
}
```

## 模考与成绩

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/assessments` | 成绩列表 |
| GET | `/goals/:id/assessments` | 指定目标成绩 |
| POST | `/assessments` | 新增成绩 |
| PUT | `/assessments/:id` | 更新成绩 |
| DELETE | `/assessments/:id` | 删除成绩 |

## 每周复盘

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/weekly-reviews` | 复盘列表 |
| GET | `/goals/:id/reviews` | 指定目标复盘列表 |
| POST | `/weekly-reviews` | 按目标与周起始日期新增或更新复盘 |

## 数据分析

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/analytics/overview` | 汇总分析 |
| GET | `/analytics/overview?goal_id=<uuid>` | 指定目标分析 |

返回累计及平均单次学习时间、任务完成率、逾期数、专注度、掌握度、高效日期/科目、30 天学习趋势、14 天任务趋势、科目进度、成绩趋势和薄弱知识点。
