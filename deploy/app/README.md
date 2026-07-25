# 一键部署

这个目录负责启动当前仓库的本地 Docker 应用栈：

- `app`：Go 后端，并内嵌 `web/` 构建产物，对外提供主应用页面和 `/api/*`
- `knowledge`：独立知识库服务

Langfuse 没有并进默认栈，仍保持为可选观测增强，入口是 `deploy/langfuse/docker-compose.yml`。

这套 compose 的定位是：

- 本地部署 / 本机演示
- 知识库目录直接和仓库同步
- 不作为线上正式部署方案

## 端口

- 主应用：`http://localhost:8080/`
- 主应用健康检查：`http://localhost:8080/api/health`
- 知识库：`http://localhost:3100/`
- 知识库状态：`http://localhost:3100/api/status`

## 首次启动

```bash
cd deploy/app
cp .env.example .env
docker compose up -d --build
```

说明：

- `app` 镜像会在构建时打包 `web/dist`，所以部署后不再需要单独跑一个前端容器。
- `knowledge/wiki/` 和 `knowledge/raw/` 会直接 bind mount 到容器里的 `/app/wiki` 和 `/app/raw`，因此本地通过知识库服务新增或修改的知识会直接落回仓库文件。
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` 和 `NEXT_PUBLIC_OWNER_HANDLE` 会在知识库镜像构建时注入到客户端 bundle，因此改完这两个值后需要重新 `docker compose up -d --build`。

## 常用命令

启动：

```bash
docker compose up -d --build
```

查看日志：

```bash
docker compose logs -f app knowledge
```

停止：

```bash
docker compose down
```

停止并删除应用侧数据卷：

```bash
docker compose down -v
```

## 环境变量

`deploy/app/.env.example` 里已经按服务分组放了最小变量集，重点是：

- `LLM_API_KEY`：主应用后端调用模型必须配置
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` / `CLERK_SECRET_KEY`：知识库 Clerk 认证
- `DEEPSEEK_API_KEY`：知识库生成与部分处理能力
- `KNOWLEDGE_SERVICE_TOKEN`：知识库写接口的服务令牌

## 同步语义

当前这套 compose 对知识库采用“仓库文件即本地真相源”：

- 容器读的是仓库里的 `knowledge/wiki/`、`knowledge/raw/`
- 容器写回的也是这两个目录
- 不再使用 `knowledge-seed` 或知识库数据 volume

如果将来线上部署不用 Docker Compose，这套本地直挂载方式会更适合持续维护知识内容。

如果只想部署 Langfuse 观测栈，请改用：

```bash
docker compose -f ../langfuse/docker-compose.yml up -d
```
