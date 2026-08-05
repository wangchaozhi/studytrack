# 学迹 StudyTrack

面向 **学位复习、考研和雅思** 的学习追踪系统。项目包含完整的 React 前端、Go REST API、PostgreSQL 数据库、自动迁移和 Docker Compose 部署配置。

## 已实现功能

- 邮箱注册、登录、JWT 鉴权和个人资料设置
- 学位复习、考研、雅思三类目标模板
- 自动生成阶段与默认科目
- 学习目标、科目、阶段和任务管理
- 今日任务、周计划和逾期任务追踪
- 专注计时、专注度、掌握度与学习笔记
- 知识点树、掌握度、错题数、复习次数和下次复习时间
- 练习、模考、正式考试和雅思成绩记录
- 首页、科目进度、学习趋势、任务完成率和成绩趋势
- 每周复盘和历史复盘
- 响应式桌面端与移动端界面
- GitHub Actions 持续集成与按版本标签发布 Release

## 技术栈

### 前端

- React 19 + TypeScript
- Vite
- React Router
- TanStack Query
- Zustand
- 原生 CSS 响应式组件体系

### 后端

- Go 1.26
- 标准库 `net/http` 路由
- pgx PostgreSQL 驱动与连接池
- bcrypt 密码哈希
- HMAC-SHA256 JWT
- PostgreSQL 17

## 一键启动

需要安装 Docker 与 Docker Compose。

```bash
cp .env.example .env
docker compose up --build
```

启动完成后访问：

- Web：`http://localhost:3000`
- API：`http://localhost:8080/api/v1`
- 健康检查：`http://localhost:8080/healthz`
- PostgreSQL：`localhost:5432`

第一次打开后注册账户即可。创建目标时，系统会自动生成对应的学习阶段和默认科目。

如果默认端口已被占用，可以在 `.env` 中调整：

```dotenv
POSTGRES_PORT=15432
API_PORT=18080
WEB_PORT=13000
VITE_API_URL=http://localhost:18080/api/v1
```

## 本地开发

### 1. 启动 PostgreSQL

可以只启动数据库：

```bash
docker compose up -d db
```

### 2. 启动 Go API

```bash
cd backend
export DATABASE_URL='postgres://studytrack:studytrack@localhost:5432/studytrack?sslmode=disable'
export JWT_SECRET='replace-with-a-long-random-secret'
go mod download
go run ./cmd/server
```

API 启动时会自动执行幂等数据库迁移。

### 3. 启动 React 前端

```bash
cd frontend
npm install
npm run dev
```

开发地址为 `http://localhost:5173`。Vite 会把 `/api` 请求代理到本机 `8080` 端口。

## 构建与检查

```bash
make build
make test
```

也可以分别执行：

```bash
cd backend
go test ./...
go build ./cmd/server

cd ../frontend
npm run lint
npm run build
```

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `DATABASE_URL` | 本地 PostgreSQL 地址 | PostgreSQL 连接串 |
| `JWT_SECRET` | 开发占位值 | 生产环境必须替换为长随机字符串 |
| `HTTP_ADDR` | `:8080` | API 监听地址 |
| `CORS_ORIGINS` | 本地前端地址 | 允许的跨域来源，逗号分隔 |
| `VITE_API_URL` | `http://localhost:8080/api/v1` | 前端 API 根地址 |
| `CORS_ORIGINS` | 本地 Web 与开发地址 | 允许访问 API 的浏览器来源，逗号分隔 |
| `POSTGRES_PORT` | `5432` | PostgreSQL 宿主机端口 |
| `API_PORT` | `8080` | API 宿主机端口 |
| `WEB_PORT` | `3000` | Web 宿主机端口 |

## 持续集成与发布

- `.github/workflows/ci.yml` 在 `main` 推送和 Pull Request 时运行 Go 测试、前端类型检查、依赖安全检查、生产构建和 Docker 镜像构建。
- `.github/workflows/release.yml` 在推送 `v*` 标签时重复完整检查，构建 Linux、macOS、Windows 后端二进制与前端静态包，并创建带 SHA-256 校验文件的 GitHub Release。
- 发布标签必须与 `frontend/package.json` 中的版本一致。例如当前版本使用 `v1.0.0`。

## 目录结构

```text
studytrack/
├── backend/
│   ├── cmd/server/              # API 入口
│   ├── internal/auth/           # 密码与 JWT
│   ├── internal/config/         # 环境配置
│   ├── internal/database/       # 数据库连接与自动迁移
│   ├── internal/httpapi/        # 路由、中间件和处理器
│   ├── internal/models/         # 领域模型
│   ├── internal/store/          # PostgreSQL 查询与统计
│   ├── migrations/              # SQL 迁移
│   └── Dockerfile
├── frontend/
│   ├── src/api/                 # API 客户端
│   ├── src/components/          # 布局与通用组件
│   ├── src/pages/               # 全部业务页面
│   ├── src/store/               # Zustand 状态
│   ├── src/types.ts             # TypeScript 类型
│   ├── src/styles.css           # 完整 UI 样式
│   ├── Dockerfile
│   └── nginx.conf
├── docs/
│   ├── API.md                  # REST API 文档
│   └── PRODUCT_PROTOTYPE.md    # 完整产品原型
├── docker-compose.yml
├── Makefile
└── .env.example
```

## 核心数据关系

```text
User
└── Goal
    ├── Phase
    ├── Subject
    │   └── KnowledgePoint
    ├── Task
    │   └── StudySession
    ├── Assessment
    └── WeeklyReview
```

## 安全与生产部署建议

- 必须替换 `JWT_SECRET`，建议使用至少 32 字节随机值。
- 生产环境应通过 HTTPS 提供前后端服务。
- PostgreSQL 不应直接暴露到公网。
- 建议将 CORS 来源限制为真实前端域名。
- 当前 JWT 有效期为 24 小时；正式产品可增加短期访问令牌、刷新令牌撤销表和设备管理。
- 当前迁移为单一幂等 schema；持续迭代时建议改为按版本递增的迁移工具。

## 产品与 API 文档

- 完整产品原型、页面线框、用户流程和验收用例见 [`docs/PRODUCT_PROTOTYPE.md`](docs/PRODUCT_PROTOTYPE.md)。
- 完整路由和请求示例见 [`docs/API.md`](docs/API.md)。
