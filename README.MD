# claude-code-go

> 基于 [Claude Code](https://www.npmjs.com/package/@anthropic-ai/claude-code) v2.1.88 Source Map 还原并翻译为 Go 语言的非官方实现

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Research%20Use%20Only-yellow.svg)](LICENSE)

> ⚠️ **警告**：本仓库为非官方整理版，基于公开 npm 发布包与 Source Map 分析还原，仅供研究使用。不代表官方原始内部开发仓库结构。

---

## 📖 项目说明

### 什么是 Claude Code？

Claude Code 是 Anthropic 官方推出的 AI 编程助手命令行工具，可以帮助开发者：
- 理解和导航代码库
- 编写和编辑代码
- 执行终端命令
- 调试和修复问题
- 与 MCP (Model Context Protocol) 服务器集成

### 本项目目的

本项目通过 `@anthropic-ai/claude-code` npm 发布包内附带的 Source Map（`cli.js.map`）还原了 TypeScript 源码（版本 2.1.88），并将其翻译为 Go 语言实现。

**还原规模：**
- 还原文件数：**4756** 个（含 1884 个 `.ts` / `.tsx` 源文件）
- 还原方式：提取 `cli.js.map` 中的 `sourcesContent` 字段

### 为什么选择 Go 版本？

- **单二进制分发**：无需 Node.js 运行时，编译后单个可执行文件即可运行
- **跨平台支持**：支持 Linux、macOS、Windows 等多平台
- **性能优势**：Go 语言的并发模型更适合处理 I/O 密集型任务
- **易于部署**：静态编译，无外部依赖

项目学习文档：https://nangongwentian-fe.github.io/learn-claude-source/

---

## 🔄 翻译进度

### 已完成模块

| 模块 | TypeScript 文件数 | Go 文件数 | 状态 |
|------|------------------|----------|------|
| **认证系统 (OAuth)** | ~15 | 3 | ✅ 完成 |
| **API 客户端** | ~20 | 8 | ✅ 完成 |
| **重试机制** | ~5 | 1 | ✅ 完成 |
| **MCP 协议** | ~10 | 6 | ✅ 完成 |
| **Bash 解析器** | ~16 | 6 | ✅ 完成 |
| **命令前缀提取** | ~5 | 1 | ✅ 完成 |
| **Shell 安全检查** | ~10 | 3 | ✅ 完成 |
| **类型定义** | ~8 | 9 | ✅ 完成 |
| **常量定义** | ~21 | 22 | ✅ 完成 |
| **CLI 框架** | ~10 | 2 | ✅ 完成 |

### 进行中模块

| 模块 | 说明 | 状态 |
|------|------|------|
| **工具系统 (Tools)** | BashTool, FileReadTool, GrepTool 等 30+ 个工具 | 🔄 部分完成 |
| **命令系统 (Commands)** | commit, review, config 等 40+ 个命令 | 🔄 部分完成 |
| **UI 组件** | React 组件的 Go 替代实现 | 🔄 部分完成 |

### 待翻译模块

| 模块 | 文件数 | 说明 |
|------|--------|------|
| **代理系统 (Agent)** | ~16 | 多代理协调、子代理生成 |
| **Hooks 系统** | ~85 | 事件钩子和回调 |
| **权限系统** | ~24 | 细粒度权限控制 |
| **UI 组件库** | ~144 | React 组件 |
| **Vim 模式** | ~5 | Vim 键绑定支持 |
| **语音交互** | ~1 | 语音输入/输出 |

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

### 🔧 Bash 安全解析

- **命令解析**：完整的 Bash 命令词法分析
- **安全检查**：检测命令注入、进程替换、危险内置命令
- **前缀提取**：智能提取命令前缀用于权限控制
- **只读模式验证**：检测破坏性命令（rm、chmod 等）

---

## 🚀 安装部署

### 前置要求

- **Go 版本**：Go 1.26 或更高版本
- **操作系统**：Linux、macOS、Windows
- **API 访问**：Anthropic API 密钥或 OAuth 认证

### 方式一：从源码构建

```bash
# 1. 克隆仓库
git clone <repository-url>
cd claude-code-go

# 2. 下载依赖
go mod download

# 3. 构建
go build -o claude-code ./cmd/cli

# 4. (可选) 安装到系统路径
sudo mv claude-code /usr/local/bin/
```

### 方式二：直接运行

```bash
# 克隆仓库
git clone <repository-url>
cd claude-code-go

# 直接运行
go run ./cmd/cli [命令] [参数]
```

### 方式三：交叉编译

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o claude-code-linux-amd64 ./cmd/cli

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o claude-code-darwin-arm64 ./cmd/cli

# macOS AMD64 (Intel)
GOOS=darwin GOARCH=amd64 go build -o claude-code-darwin-amd64 ./cmd/cli

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o claude-code-windows-amd64.exe ./cmd/cli
```

### 验证安装

```bash
# 检查版本
claude-code version

# 运行诊断
claude-code doctor
```

---

## 📚 如何使用

### 基本用法

```bash
# 交互式对话
claude-code

# 带初始提示启动
claude-code "帮我分析这个项目的结构"

# 指定模型
claude-code --model claude-sonnet-4-20250514 "优化这段代码"

# 调试模式
claude-code --debug "为什么这个测试失败了？"
```

### 命令行参数

| 参数 | 简写 | 说明 |
|------|------|------|
| `--debug` | `-d` | 启用调试模式，输出详细日志 |
| `--verbose` | `-v` | 启用详细输出 |
| `--print` | `-p` | 打印模式（非交互式） |
| `--model` | `-m` | 指定使用的模型（如 claude-sonnet-4-20250514） |
| `--permission-mode` | | 权限模式：default, acceptEdits, bypassPermissions |
| `--max-turns` | | 最大对话轮数（默认 100） |

### 子命令

```bash
# 显示版本信息
claude-code version

# 配置管理
claude-code config

# MCP 服务器管理
claude-code mcp

# 运行系统诊断
claude-code doctor
```

### 认证配置

#### 使用 API Key

```bash
# 设置环境变量
export ANTHROPIC_API_KEY="your-api-key"

# 或者使用 OAuth token
export ANTHROPIC_AUTH_TOKEN="your-oauth-token"
```

#### 使用 AWS Bedrock

```bash
export CLAUDE_CODE_USE_BEDROCK=true
export AWS_REGION="us-east-1"
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
```

#### 使用 Google Vertex AI

```bash
export CLAUDE_CODE_USE_VERTEX=true
# 使用 ADC 认证，需先运行：gcloud auth application-default login
```

#### 使用 Azure Foundry

```bash
export CLAUDE_CODE_USE_FOUNDRY=true
export ANTHROPIC_API_KEY="your-azure-api-key"
```

---

## 🎯 场景用例

### 1. 代码审查

```bash
# 审查最近的代码变更
claude-code "审查 git diff 的变更，指出潜在问题"

# 审查特定文件
claude-code "审查 src/api/client.go，检查错误处理是否完善"
```

### 2. 代码重构

```bash
# 重构建议
claude-code "重构 internal/services/api/retry.go，使其更易测试"

# 代码优化
claude-code "优化这个函数的性能，减少内存分配"
```

### 3. 调试问题

```bash
# 启用调试模式分析问题
claude-code --debug "为什么我的程序在处理并发请求时出现死锁？"

# 分析错误日志
claude-code "分析这个错误日志，找出根本原因：$(cat error.log)"
```

### 4. 文档生成

```bash
# 生成函数文档
claude-code "为 internal/utils/auth.go 中的所有公开函数生成 Go 文档注释"

# 生成 README
claude-code "为我的新项目生成 README.md，包括安装和使用说明"
```

### 5. 测试辅助

```bash
# 生成测试用例
claude-code "为 internal/services/api/client.go 生成单元测试"

# 分析测试覆盖率
claude-code "分析测试覆盖率报告，找出未覆盖的关键路径"
```

### 6. 架构理解

```bash
# 理解项目结构
claude-code "解释这个项目的架构设计，包括各模块的职责"

# 代码导航
claude-code "找到所有实现了 io.Reader 接口的类型"
```

### 7. 安全审计

```bash
# 安全检查
claude-code "检查代码中是否存在 SQL 注入、XSS 等安全漏洞"

# 敏感信息检查
claude-code "检查代码中是否意外暴露了敏感信息（API key、密码等）"
```

### 8. 依赖管理

```bash
# 分析依赖
claude-code "分析 go.mod 中的依赖，找出可升级的版本"

# 检查漏洞
claude-code "检查项目依赖是否存在已知安全漏洞"
```

### 9. CI/CD 集成

```bash
# 在 CI 中非交互式运行
claude-code --print --permission-mode bypassPermissions "检查代码是否符合项目规范"

# 自动化代码审查
claude-code --max-turns 1 "审查当前 PR 的变更并给出建议"
```

### 10. 多云部署场景

```bash
# 使用 AWS Bedrock
export CLAUDE_CODE_USE_BEDROCK=true
claude-code "生成部署到 AWS Lambda 的配置文件"

# 使用 Google Vertex AI
export CLAUDE_CODE_USE_VERTEX=true
claude-code "生成 Cloud Run 的部署配置"
```

---

## 📂 目录结构

### TypeScript 原始代码

```
restored-src/src/
├── main.tsx              # CLI 入口
├── tools/                # 工具实现（Bash、FileEdit、Grep、MCP 等 170+ 个文件）
│   ├── BashTool/         # Bash 命令执行工具
│   ├── FileReadTool/     # 文件读取工具
│   ├── FileEditTool/     # 文件编辑工具
│   ├── GrepTool/         # 文件内容搜索工具
│   ├── GlobTool/         # 文件名搜索工具
│   ├── MCPTool/          # MCP 协议工具
│   ├── AgentTool/        # 多代理系统
│   └── ...               # 其他 30+ 个工具
├── commands/             # 命令实现（commit、review、config 等 100+ 个文件）
├── services/             # API、MCP、分析等服务（36 个文件）
├── utils/                # 工具函数（git、model、auth、env 等 430+ 个文件）
│   ├── bash/             # Bash 解析器（16 个文件）
│   ├── model/            # 模型管理（17 个文件）
│   ├── permissions/      # 权限系统（24 个文件）
│   └── shell/            # Shell 提供者（10 个文件）
├── components/           # React UI 组件（144 个文件）
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
│   ├── tools.go          # 工具名称常量和权限配置
│   ├── common.go         # 通用常量和日期函数
│   ├── api_limits.go     # API 限制常量
│   ├── keys.go           # GrowthBook 密钥配置
│   ├── files.go          # 文件类型检测
│   ├── prompts.go        # 系统提示构建
│   ├── messages.go       # 消息常量
│   ├── error_ids.go      # 错误ID常量
│   ├── github_app.go     # GitHub App 配置
│   ├── product.go        # 产品常量
│   ├── turn_completion_verbs.go # 轮次完成动词
│   ├── cyber_risk.go     # 安全风险指令
│   ├── tool_limits.go    # 工具限制
│   ├── spinner_verbs.go  # 加载动画动词
│   ├── figures.go        # Unicode 图形字符
│   ├── xml.go            # XML 常量
│   ├── output_styles.go  # 输出样式配置
│   ├── system.go         # 系统提示前缀和归属头
│   └── system_prompt_sections.go # 系统提示段落管理
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
│   ├── mcp/              # MCP 协议实现
│   │   ├── client.go     # MCP 客户端
│   │   ├── config.go     # 配置管理
│   │   ├── manager.go    # MCP 服务器管理
│   │   ├── types.go      # 类型定义
│   │   └── mcpUtils.go   # MCP 工具函数
│   ├── config.go         # 配置服务
│   ├── compact.go        # 压缩服务
│   ├── plugins.go        # 插件服务
│   ├── analytics.go      # 分析服务
│   └── lsp.go            # LSP 服务
├── utils/                # 工具函数
│   ├── auth.go           # 认证管理器
│   ├── providers.go      # API 提供商检测
│   ├── uuid.go           # UUID 生成
│   ├── file.go           # 文件操作
│   ├── http.go           # HTTP 工具
│   ├── git_utils.go      # Git 操作
│   ├── lru_cache.go      # LRU 缓存
│   ├── bash_parser.go    # Bash 命令解析器
│   ├── bash_security.go  # Bash 安全检查
│   ├── shell_ast.go      # Shell AST 解析
│   ├── shell_quote.go    # Shell 引用处理
│   ├── command_spec.go   # 命令规范
│   ├── command_prefix.go # 命令前缀提取
│   └── ...               # 其他工具函数
├── types/                # 类型定义
│   ├── ids.go            # ID 类型
│   ├── logs.go           # 日志类型
│   ├── task.go           # 任务类型
│   ├── tool.go           # 工具类型
│   ├── hooks.go          # 钩子类型
│   ├── plugin.go         # 插件类型
│   ├── command.go        # 命令类型
│   ├── appstate.go       # 应用状态
│   └── permissions.go    # 权限类型
├── tools/                # 工具实现
│   ├── tools.go          # 工具注册
│   ├── advanced_tools.go # 高级工具
│   └── additional_tools.go # 其他工具
├── commands/             # 命令实现
│   └── commands.go       # 命令注册
├── cli/                  # CLI 应用
│   └── app.go            # 应用逻辑
├── query/                # 查询引擎
│   └── engine.go         # 查询引擎实现
├── state/                # 状态管理
└── ui/                   # UI 组件
    ├── ui.go             # UI 主入口
    ├── hooks/            # UI Hooks
    └── components/       # UI 组件

cmd/
└── cli/
    └── main.go           # 程序入口

pkg/
└── api/
    └── client.go         # 公开 API 客户端
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

## 🔨 开发指南

### 构建

```bash
# 编译所有包
go build ./...

# 运行测试
go test ./...

# 静态分析
go vet ./...

# 构建二进制
go build -o claude-code ./cmd/cli
```

### 代码规范

- 遵循 [Go 代码规范](https://golang.org/doc/effective_go)
- 使用 `gofmt` 格式化代码
- 添加必要的注释和文档

---

## 🤝 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

---

## 📋 开发路线

### 已完成 ✅
- [x] OAuth 认证流程实现
- [x] 多云供应商认证（AWS Bedrock、Google Vertex、Azure Foundry）
- [x] API 客户端与重试机制
- [x] MCP 协议基础支持
- [x] Bash 命令解析器
- [x] Shell 安全检查
- [x] 命令前缀提取
- [x] CLI 框架（基于 Cobra）
- [x] 完整的常量模块（22个文件，涵盖所有核心常量）
- [x] 系统提示构建系统
- [x] 工具权限配置系统

### 进行中 🔄
- [ ] 工具系统完整实现
- [ ] 命令系统完整实现
- [ ] UI 组件（Bubble Tea）

### 计划中 📝
- [ ] 完整的 MCP 协议支持
- [ ] 多代理协调系统
- [ ] Hooks 系统
- [ ] Vim 模式支持
- [ ] 语音交互支持
- [ ] 插件系统
- [ ] LSP 集成

---

## ❓ 常见问题

### Q: 如何更新到最新版本？

```bash
git pull origin main
go build -o claude-code ./cmd/cli
```

### Q: 遇到认证错误怎么办？

1. 检查 `ANTHROPIC_API_KEY` 或 `ANTHROPIC_AUTH_TOKEN` 是否正确设置
2. 运行 `claude-code doctor` 检查系统状态
3. 尝试使用 `--debug` 参数查看详细日志

### Q: 如何查看支持的模型列表？

```bash
claude-code --help
# 或查看 Anthropic 官方文档
```

### Q: 在企业环境中如何使用？

配置 `CLAUDE_CODE_CUSTOM_OAUTH_URL` 使用企业 OAuth 端点，或使用 AWS Bedrock / Azure Foundry 等云服务。

### Q: 编译时报错怎么办？

确保使用 Go 1.26 或更高版本：
```bash
go version
go build ./...
```

---

## ⚖️ 声明

- 源码版权归 **Anthropic** 所有
- 本仓库仅用于技术研究与学习，请勿用于商业用途
- 如有侵权，请联系删除

---

## 📄 License

本项目仅供学习和研究使用。原始 TypeScript 源码版权归 Anthropic 所有。
