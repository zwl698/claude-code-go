# claude-code-go

> 基于 [Claude Code](https://www.npmjs.com/package/@anthropic-ai/claude-code) v2.1.88 Source Map 还原并翻译为 Go 语言的非官方实现

[!WARNING]
> ⚠️ 本仓库为非官方整理版，基于公开 npm 发布包与 Source Map 分析还原，仅供研究使用。不代表官方原始内部开发仓库结构。一切基于 L 站「飘然与我同」的情报提供。

---

## 📖 概述

本项目通过 `@anthropic-ai/claude-code` npm 发布包内附带的 Source Map（`cli.js.map`）还原了 TypeScript 源码（版本 2.1.88），并将其翻译为 Go 语言实现。

**还原规模：**

- 还原文件数：**4756** 个（含 1884 个 `.ts` / `.tsx` 源文件）
- 还原方式：提取 `cli.js.map` 中的 `sourcesContent` 字段

---

## 📂 目录结构

```
restored-src/src/
├── main.tsx              # CLI 入口
├── tools/                # 工具实现（Bash、FileEdit、Grep、MCP 等 30+ 个）
├── commands/             # 命令实现（commit、review、config 等 40+ 个）
├── services/             # API、MCP、分析等服务
├── utils/                # 工具函数（git、model、auth、env 等）
├── context/              # React Context
├── coordinator/          # 多 Agent 协调模式
├── assistant/            # 助手模式（KAIROS）
├── buddy/                # AI 伴侣 UI
├── remote/               # 远程会话
├── plugins/              # 插件系统
├── skills/               # 技能系统
├── voice/                # 语音交互
└── vim/                  # Vim 模式
```

---

## 🔗 来源信息

| 项目 | 说明 |
|------|------|
| npm 包 | `@anthropic-ai/claude-code` |
| 还原版本 | 2.1.88 |

---

## ⚖️ 声明

- 源码版权归 **Anthropic** 所有
- 本仓库仅用于技术研究与学习，请勿用于商业用途
- 如有侵权，请联系删除
