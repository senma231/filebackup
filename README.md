# Windows文档扫描与上传系统

基于Go语言开发的Windows文档自动扫描与上传系统，采用Agent-Server架构。

## 🌟 核心特性

### Windows Agent
- ✅ 自动扫描文档文件（Word、Excel、PPT、PDF、TXT）
- ✅ 智能文件过滤（排除log文件等）
- ✅ SFTP断点续传上传
- ✅ 心跳监控与状态上报
- ✅ 配置热更新
- ✅ Windows服务模式
- ✅ **邮箱用户识别** - 首次启动提示输入邮箱，上传文件按用户隔离

### Server端
- ✅ Agent注册与管理
- ✅ 实时心跳监控
- ✅ 配置管理（全局/专属）
- ✅ 文件上传记录跟踪
- ✅ 统计报告与数据分析
- ✅ **邮箱信息展示** - 管理界面显示用户邮箱和文件夹前缀

## 📋 新增功能：邮箱用户识别

### 工作流程

1. **Agent首次启动**
   - 自动检测配置文件
   - 如果未配置邮箱，提示用户输入邮箱地址
   - 邮箱格式：`user@domain.com`

2. **邮箱处理**
   - 自动提取邮箱前缀（@前面的部分）
   - 例如：`caa-davidxie@parisigs.com` → `caa-davidxie`
   - 将邮箱完整信息和前缀存储到配置中

3. **文件上传**
   - 上传路径：`/uploads/{邮箱前缀}/{日期}/{文件名}`
   - 例如：`/uploads/caa-davidxie/2025-12-15/document.docx`
   - 实现用户数据隔离

4. **Server管理**
   - 记录Agent的邮箱信息
   - 在管理界面显示用户邮箱和文件夹前缀
   - 支持按用户筛选和搜索

### 配置文件变化

**Agent配置** (`config.json`):
```json
{
  "email": "caa-davidxie@parisigs.com",
  "email_prefix": "caa-davidxie",
  "scan_paths": [...],
  ...
}
```

**数据库结构** (`agents` 表):
```sql
CREATE TABLE agents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id VARCHAR(64) UNIQUE NOT NULL,
    email VARCHAR(255),           -- 新增：完整邮箱
    email_prefix VARCHAR(255),    -- 新增：邮箱前缀
    hostname VARCHAR(255) NOT NULL,
    ...
);
```

## 🏗️ 项目结构

```
doc-scanner-system/
├── agent/                          # Windows Agent
│   ├── cmd/main.go                 # 主程序入口
│   ├── internal/
│   │   ├── config/                 # 配置管理
│   │   │   ├── config.go           # 配置结构（含邮箱字段）
│   │   │   └── loader.go           # 配置加载器
│   │   ├── scanner/                # 文件扫描
│   │   ├── uploader/               # 文件上传
│   │   │   └── transfer.go         # 上传逻辑（支持邮箱前缀路径）
│   │   ├── heartbeat/              # 心跳监控
│   │   │   └── service.go          # 心跳服务（含邮箱上报）
│   │   └── logger/                 # 日志
│   └── configs/
│       └── default.json            # 默认配置（含邮箱字段）
│
├── server/                         # Server端
│   ├── cmd/main.go                 # 主程序入口
│   ├── internal/
│   │   ├── model/
│   │   │   └── agent.go            # Agent模型（含邮箱字段）
│   │   ├── repository/
│   │   │   ├── db.go               # 数据库迁移（含邮箱字段）
│   │   │   └── agent_repo.go       # Agent数据访问（含邮箱处理）
│   │   └── api/
│   │       └── handlers/
│   │           └── agent.go        # Agent API（含邮箱处理）
│   └── migrations/
│       └── 001_init.sql            # 初始化脚本（含邮箱字段）
│
└── docs/                           # 文档
    ├── 需求文档.md
    ├── 实施计划.md
    ├── 项目结构.md
    └── 项目总结.md
```

## 🔧 开发状态

### ✅ 已完成

1. **Agent端**
   - [x] 配置结构添加邮箱字段
   - [x] 邮箱前缀提取逻辑
   - [x] 文件上传路径支持邮箱前缀
   - [x] 心跳上报包含邮箱信息

2. **Server端**
   - [x] 数据库结构更新（添加email和email_prefix字段）
   - [x] Agent模型更新
   - [x] 数据访问层更新
   - [x] API接口更新（注册和心跳都处理邮箱）

### 🔄 待完成

1. **Agent首次启动流程**
   - [ ] 添加邮箱输入提示界面
   - [ ] 交互式配置向导
   - [ ] 邮箱格式验证

2. **Web管理界面**
   - [ ] 显示Agent邮箱信息
   - [ ] 按邮箱筛选功能
   - [ ] 邮箱前缀文件夹展示

3. **系统测试**
   - [ ] 端到端测试
   - [ ] 邮箱流程测试
   - [ ] 文件隔离测试

## 🚀 快速开始

### 1. 构建Agent
```bash
cd agent
go mod tidy
go build -o bin/agent.exe ./cmd/main.go
```

### 2. 构建Server
```bash
cd server
go mod tidy
go build -o bin/server ./cmd/main.go
```

### 3. 启动Server
```bash
cd server
mkdir -p data
./bin/server
```

### 4. 运行Agent
```bash
cd agent
./bin/agent.exe
```

**首次运行提示**：
Agent启动时会检查配置，如果未设置邮箱，会提示：
```
请输入您的邮箱地址（例如：user@domain.com）: _
```

输入邮箱后，Agent将：
1. 保存邮箱到配置
2. 自动提取前缀
3. 向Server注册
4. 开始扫描和上传文件

## 📊 数据库更新

运行数据库迁移后，`agents` 表将自动添加：
- `email` 字段：存储完整邮箱地址
- `email_prefix` 字段：存储邮箱前缀

现有数据会自动保持兼容。

## 🎯 使用示例

### 场景1：用户上传文件
- **邮箱**: `caa-davidxie@parisigs.com`
- **文件夹**: `caa-davidxie`
- **上传路径**: `/uploads/caa-davidxie/2025-12-15/document.docx`

### 场景2：多用户隔离
- **用户A**: `user1@domain.com` → `/uploads/user1/...`
- **用户B**: `user2@domain.com` → `/uploads/user2/...`

不同用户的文件完全隔离，便于管理和检索。

## 📝 变更日志

### v1.1.0 (2025-12-15)
- ✨ 新增邮箱用户识别功能
- ✨ Agent首次启动提示输入邮箱
- ✨ 文件上传按用户隔离
- ✨ Server记录和显示邮箱信息
- 🔧 数据库结构更新（email和email_prefix字段）
- 🔧 API接口更新支持邮箱
- 🔧 配置文件更新

## 📄 许可证

MIT License

## 👨‍💻 作者

Claude Code - Anthropic官方CLI工具

---

**注意**: 本项目仍在开发中，部分功能待完善。欢迎提交Issue和Pull Request！
