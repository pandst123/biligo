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

## 特别鸣谢
项目 [biliTickerBuy](https://github.com/mikumifa/biliTickerBuy) 提供抢票相关逻辑
项目 [BHYG](https://github.com/ZianTT/BHYG) 提供风控监测相关逻辑