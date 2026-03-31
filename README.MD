# claude-code-go

> 基于 [Claude Code](https://www.npmjs.com/package/@anthropic-ai/claude-code) v2.1.88 Source Map 还原并翻译为 Go 语言的非官方实现

[!WARNING]
> ⚠️ 本仓库为非官方整理版，基于公开 npm 发布包与 Source Map 分析还原，仅供研究使用。不代表官方原始内部开发仓库结构。一切基于 L 站「飘然与我同」的情报提供。

---

## 📖 概述

本项目通过 `@anthropic-ai/claude-code` npm 发布包内附带的 Source Map（`cli.js.map`）还原了 TypeScript 源码（版本 2.1.88），并将其翻译为 Go 语言实现。
项目学习文档：https://nangongwentian-fe.github.io/learn-claude-source/
**还原规模：**

- 还原文件数：**4756** 个（含 1884 个 `.ts` / `.tsx` 源文件）
- 还原方式：提取 `cli.js.map` 中的 `sourcesContent` 字段

---

## ✨ Go 版本实现功能

### 🔐 认证系统

#### OAuth 认证
- **Token 管理**：支持 Access Token、Refresh Token 的存储与自动刷新
- **并发安全**：使用 mutex 和 channel 实现线程安全的 token 操作
- **跨进程协调**：文件锁确保多进程环境下 token 刷新的正确性
- **自动刷新**：检测 token 过期或 401 错误时自动刷新
- **环境变量支持**：`CLAUDE_CODE_OAUTH_TOKEN` 环境变量覆盖

#### 多云供应商认证
- **AWS Bedrock**：AWS SigV4 签名、STS 临时凭证、SSO 登录
- **Google Vertex AI**：ADC (Application Default Credentials) 认证
- **Azure Foundry**：API Key 和 Azure AD 认证

### 🔄 重试机制

- **指数退避**：基础延迟 500ms，最大延迟 32 秒
- **抖动**：随机化退避时间避免惊群效应
- **持久重试**：支持长时间运行的会话（最大 6 小时）
- **错误分类处理**：
  - 429 速率限制：尊重 `Retry-After` 和 `Anthropic-Ratelimit-Unified-Reset` 头
  - 529 过载错误：区分前台/后台任务重试策略
  - 401 认证错误：自动刷新 OAuth token
  - 5xx 服务器错误：指数退避重试
- **心跳机制**：长时间等待时定期输出心跳日志

### 🔌 MCP (Model Context Protocol)

- **传输层**：支持 stdio 和 SSE 传输
- **工具过滤**：基于 URL 模式的工具名称过滤
- **配置管理**：支持 `${VAR:-default}` 环境变量扩展语法
- **权限检查**：细粒度的工具权限验证

---

## 📂 目录结构

### TypeScript 原始代码

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

### Go 实现代码

```
internal/
├── constants/            # 常量定义
│   ├── oauth.go          # OAuth 配置（scopes、endpoints）
│   ├── betas.go          # API beta headers
│   └── api_limits.go     # API 限制常量
├── services/
│   ├── api/              # API 客户端
│   │   ├── client.go     # Anthropic API 客户端
│   │   ├── retry.go      # 重试逻辑
│   │   ├── errors.go     # 错误类型定义
│   │   ├── auth_aws.go   # AWS Bedrock 认证
│   │   ├── auth_vertex.go # Google Vertex 认证
│   │   ├── auth_azure.go # Azure Foundry 认证
│   │   └── signer_aws.go # AWS SigV4 签名
│   ├── oauth/            # OAuth 认证
│   │   └── client.go     # OAuth token 管理
│   └── mcp/              # MCP 协议实现
│       ├── client.go     # MCP 客户端
│       ├── config.go     # 配置管理
│       └── manager.go    # MCP 服务器管理
├── utils/                # 工具函数
│   ├── auth.go           # 认证管理器
│   ├── providers.go      # API 提供商检测
│   └── uuid.go           # UUID 生成
├── types/                # 类型定义
└── cli/                  # CLI 入口
```

---

## 🔗 来源信息

| 项目 | 说明 |
|------|------|
| npm 包 | `@anthropic-ai/claude-code` |
| 还原版本 | 2.1.88 |

---

## 🛠️ 环境变量

### OAuth 配置
| 变量 | 说明 |
|------|------|
| `CLAUDE_CODE_OAUTH_TOKEN` | 直接设置 OAuth access token |
| `CLAUDE_CODE_OAUTH_TOKEN_FILE_DESCRIPTOR` | 从文件描述符读取 token |
| `CLAUDE_CODE_OAUTH_CLIENT_ID` | 覆盖 OAuth client ID |
| `CLAUDE_CODE_CUSTOM_OAUTH_URL` | 自定义 OAuth 端点（仅限 FedStart） |

### 多云认证
| 变量 | 说明 |
|------|------|
| `CLAUDE_CODE_USE_BEDROCK` | 使用 AWS Bedrock |
| `CLAUDE_CODE_USE_VERTEX` | 使用 Google Vertex AI |
| `CLAUDE_CODE_USE_FOUNDRY` | 使用 Azure Foundry |
| `ANTHROPIC_API_KEY` | API 密钥 |
| `ANTHROPIC_AUTH_TOKEN` | OAuth bearer token |

### 重试配置
| 变量 | 说明 |
|------|------|
| `API_TIMEOUT_MS` | API 请求超时（毫秒） |
| `CLAUDE_CODE_DISABLE_PERSISTENT_RETRY` | 禁用持久重试 |

---

## ⚖️ 声明

- 源码版权归 **Anthropic** 所有
- 本仓库仅用于技术研究与学习，请勿用于商业用途
- 如有侵权，请联系删除
