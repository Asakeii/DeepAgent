# syntax=docker/dockerfile:1
#
# deepAgent 容器镜像
# 运行时需要三类东西：
#   1) Go 编译出的 deep-agent 二进制（build 阶段产物）
#   2) uv + python 运行环境，用于拉起 mcps/python 的 stdio MCP（server.py）
#   3) nodejs + npx，用于拉起 tavily-mcp（stdio MCP，通过 npx 动态下载）
# 因此 runtime 阶段下拉 python:3.12-slim 作为基础，apt 装 nodejs/npm，
# pip 装 uv，再用 uv sync 按仓库现有的 uv.lock 装 python MCP 依赖。
# （_preflight 修正：原 brief 手写 pip install mcp==1.8.1 ... 违背项目 uv 约定，
#  这里对齐 mcps/python/pyproject.toml + uv.lock，保证依赖与本地一致。）

############################
# build stage：编译 Go 二进制
############################
FROM golang:1.25-alpine AS builder

WORKDIR /src
# 缓存依赖：先拷 mod 文件下载，再拷源码
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0：go-sql-driver/mysql 是纯 Go，多平台/静态链接更稳。
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/deep-agent .

############################
# runtime stage
############################
FROM python:3.12-slim

# 安装 nodejs + npm（npx 由 npm 提供），tavily-mcp 通过 npx 拉起。
# ca-certificates 供 npx/uv 走 https。
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        nodejs \
        npm \
    && rm -rf /var/lib/apt/lists/*

# 安装 uv（官方推荐方式之一：pip 装），用于按 lock 文件同步 python MCP 依赖。
RUN pip install --no-cache-dir uv

# 先拷 python MCP 的 lock 文件并 uv sync，利用层缓存。
# --frozen 强制使用 uv.lock，不更新；--no-dev 不装开发依赖。
WORKDIR /app/mcps/python
COPY mcps/python/pyproject.toml mcps/python/uv.lock ./
RUN uv sync --frozen --no-dev

# server.py 在 uv 创建的 venv 之外用 uv run 启动，故单独拷一份。
COPY mcps/python/server.py ./

WORKDIR /app

# 拷 Go 二进制与运行时需要的静态资源。
# 注意：conf/deep-agent.yaml 通过 docker-compose volume 挂载注入（含真实 key，不进镜像）。
COPY --from=builder /out/deep-agent /app/deep-agent
COPY internal/agent/ /app/internal/agent/
COPY internal/tool/ /app/internal/tool/
COPY internal/prompts/ /app/internal/prompts/
COPY internal/store/schema.sql /app/internal/store/schema.sql

# 暴露 server 模式监听端口（DEEPAGENT_MODE=server 时 listen :8080）
EXPOSE 8080

CMD ["/app/deep-agent"]
