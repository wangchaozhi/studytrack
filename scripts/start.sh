#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$project_root"

no_build=false
if [[ "${1:-}" == "--no-build" ]]; then
  no_build=true
elif [[ $# -gt 0 ]]; then
  echo "用法：$0 [--no-build]" >&2
  exit 2
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "未找到 Docker，请先安装并启动 Docker。" >&2
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose 不可用，请确认 Docker 已正常启动。" >&2
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "未找到 curl，无法执行启动健康检查。" >&2
  exit 1
fi

if [[ ! -f .env ]]; then
  if [[ ! -f .env.example ]]; then
    echo "缺少 .env.example，无法生成运行配置。" >&2
    exit 1
  fi

  umask 077
  if command -v openssl >/dev/null 2>&1; then
    jwt_secret="$(openssl rand -hex 32)"
  else
    jwt_secret="$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')"
  fi

  awk -v secret="$jwt_secret" '
    /^JWT_SECRET=/ { print "JWT_SECRET=" secret; next }
    { print }
  ' .env.example > .env
  echo "已根据 .env.example 创建 .env，并生成随机 JWT 密钥。"
fi

env_value() {
  local key="$1"
  local default_value="$2"
  local value
  value="${!key:-}"
  if [[ -n "$value" ]]; then
    printf '%s' "$value"
    return
  fi
  value="$(awk -F= -v key="$key" '$1 == key { sub(/^[^=]*=/, ""); value=$0 } END { print value }' .env | tr -d '\r')"
  printf '%s' "${value:-$default_value}"
}

api_port="$(env_value API_PORT 8080)"
web_port="$(env_value WEB_PORT 3000)"
api_health_url="http://localhost:${api_port}/healthz"
web_url="http://localhost:${web_port}"

echo "正在启动 StudyTrack..."
if [[ "$no_build" == true ]]; then
  if ! docker compose up -d; then
    docker compose ps
    echo "Docker Compose 启动失败。如果端口被占用，请修改 .env 中的 POSTGRES_PORT、API_PORT 或 WEB_PORT 后重试。" >&2
    exit 1
  fi
else
  if ! docker compose up -d --build; then
    docker compose ps
    echo "Docker Compose 启动失败。如果端口被占用，请修改 .env 中的 POSTGRES_PORT、API_PORT 或 WEB_PORT 后重试。" >&2
    exit 1
  fi
fi

echo "正在等待 API 和 Web 就绪..."
ready=false
for ((attempt = 1; attempt <= 60; attempt++)); do
  if curl --fail --silent --max-time 3 "$api_health_url" >/dev/null \
    && curl --fail --silent --max-time 3 "$web_url" >/dev/null; then
    ready=true
    break
  fi
  sleep 2
done

if [[ "$ready" != true ]]; then
  docker compose ps
  echo "服务未在 120 秒内就绪。请运行 'docker compose logs' 查看日志。" >&2
  exit 1
fi

echo
echo "StudyTrack 已启动"
echo "Web：$web_url"
echo "API：http://localhost:${api_port}/api/v1"
echo "健康检查：$api_health_url"
echo "停止服务：docker compose down"
