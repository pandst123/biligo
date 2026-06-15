# Biligo

Biligo 是一个 哔哩哔哩会员购 抢票辅助工具

## 温馨提示
1. Biligo 仅用于个人学习、研究抢票操作。使用者应自行确认使用行为符合平台服务条款。因使用本项目产生的账号限制、订单失败或其他后果，由使用者自行承担。
2. ⚠️ 本项目采用AI代码辅助构建，项目演进流程请参见`docs/completion.md`

## 声明
若本项目有侵权内容，请联系 535128725@qq.com，我会第一时间下架该项目 

## 技术栈

| 模块 | 技术 |
| --- | --- |
| 后端服务 | Go, Gin |
| 前端控制台 | Vue 3, Vite, TypeScript |
| 前端包管理 | pnpm |
| 数据存储 | SQLite |
| 实时状态 | SSE |
| 部署形态 | 本地单用户运行 |

## 架构概览

```text
Vue Web Console
      |
      | HTTP / SSE
      v
Gin API Server
      |
      +-- Auth Session       登录态与 Cookie 状态
      +-- Task Scheduler     本地任务调度、启停、恢复
      +-- Ticket Service     活动、场次、票档监控
      +-- Order Service      下订单
      +-- Log / Notify       任务日志、运行事件、用户提示
      |
      v
SQLite
```

## 接口信息

详见 `docs/api.md`

## 直接运行 (开发模式)

开发时可以前后端分别启动。后端默认监听 `:8080`，前端 Vite 开发服务默认监听 `5173`，并将 `/api` 代理到后端。

```bash
# 启动后端 API
go run ./cmd/server -config config.yaml
```

```bash
# 启动前端开发服务
cd web
pnpm install --frozen-lockfile
pnpm dev
```

访问 `http://127.0.0.1:5173/` 使用前端控制台。

如果已经使用 `embed_web` 构建了嵌入前端的二进制，则只需要运行后端程序，访问 `server.addr` 对应地址即可，例如 `http://127.0.0.1:8080/`。
未使用 `embed_web` 构建时，程序会仅提供 API 服务。

## 编译打包

开发模式下，前端和后端可以分别启动。若需要打包为一个可执行文件，并由同一个端口同时提供前端页面和 `/api`，可以使用 `embed_web` 构建标签。

```bash
cd web
pnpm install --frozen-lockfile
pnpm build
cd ..

rm -rf internal/webui/dist
mkdir -p internal/webui/dist
cp -R web/dist/. internal/webui/dist/

GOOS=windows GOARCH=amd64 go build \
  -tags embed_web \
  -trimpath \
  -ldflags="-s -w" \
  -o release/biligo.exe \
  ./cmd/server
```

嵌入前端会自动启用，无需额外配置。端口仍由 `server.addr` 控制：

```yaml
server:
  addr: ":8080"
```

运行后访问 `http://127.0.0.1:8080/`，API 仍位于同端口的 `/api` 下。

## 特别鸣谢
项目 [biliTickerBuy](https://github.com/mikumifa/biliTickerBuy) 提供抢票相关逻辑
项目 [BHYG](https://github.com/ZianTT/BHYG) 提供风控监测相关逻辑
