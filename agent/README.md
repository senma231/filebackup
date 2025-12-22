# Windows Agent 服务管理脚本使用说明

## 📦 文件说明

本目录包含以下文件：
- `agent.exe` - Agent主程序
- `config.json` - 配置文件
- `install-service.bat` - 安装Windows服务（需要管理员权限）
- `start-service.bat` - 启动服务
- `stop-service.bat` - 停止服务
- `restart-service.bat` - 重启服务
- `uninstall-service.bat` - 卸载服务（需要管理员权限）
- `run-console.bat` - 控制台模式运行（开发/调试用）

## 🚀 快速开始

### 方式一：服务模式（推荐）

**适用于生产环境，开机自动启动**

1. **右键以管理员身份运行** `install-service.bat`
   - 安装Windows服务
   - 服务名称：`DocScannerAgent`
   - 显示名称：`文档扫描Agent`

2. 双击 `start-service.bat` 启动服务

3. 服务管理：
   - 启动服务：双击 `start-service.bat`
   - 停止服务：双击 `stop-service.bat`
   - 重启服务：双击 `restart-service.bat`
   - 卸载服务：**右键以管理员身份运行** `uninstall-service.bat`

### 方式二：控制台模式

**适用于开发调试，窗口关闭后程序停止**

直接双击 `run-console.bat`，按 Ctrl+C 停止

## ⚙️ 配置说明

### config.json 配置文件

```json
{
  "agent_id": "your-agent-id",
  "server_url": "http://your-server:8080",
  "full_disk_scan": true,
  "scan_paths": ["C:\\Users\\%USERNAME%\\Documents"],
  "file_types": [".doc", ".docx", ".pdf", ".txt"],
  "exclude_patterns": ["*.log", "*.tmp", "~$*"],
  "max_file_size": 104857600
}
```

### 重要配置项说明

#### full_disk_scan（全盘扫描）**【重要】**
- **作用**：是否扫描所有可用的磁盘驱动器
- **默认值**：`true`（启用全盘扫描）
- **说明**：
  - `true`：**自动检测并扫描所有可用驱动器**（C:, D:, E:等），忽略 scan_paths 配置
  - `false`：仅扫描 scan_paths 中配置的特定目录

#### scan_paths（扫描路径）
- **作用**：仅当 `full_disk_scan=false` 时使用
- **说明**：指定要监控并上传文件的目录列表
- **示例**：
  - `"C:\\Users\\%USERNAME%\\Documents"` - 监控文档文件夹
  - `"C:\\Users\\%USERNAME%\\Desktop"` - 监控桌面
  - `"D:\\重要文件"` - 监控自定义目录

#### file_types（文件类型）
- **作用**：指定要备份的文件扩展名
- **默认值**：`.doc`, `.docx`, `.xls`, `.xlsx`, `.ppt`, `.pptx`, `.pdf`, `.txt`
- **说明**：只有这些类型的文件会被上传

#### exclude_patterns（排除模式）
- **作用**：排除不需要备份的文件
- **默认值**：`*.log`, `*.tmp`, `~$*`（临时文件、日志文件等）

#### 工作流程

**全盘扫描模式（默认，推荐）**：
```
所有可用驱动器(C:,D:,E:等) → Agent自动检测并扫描 → 匹配file_types → 上传到远程存储
```

**指定路径模式**：
```
scan_paths (指定目录) → Agent扫描 → 匹配file_types → 上传到远程存储
```

**重要**：上传后，原文件仍保留在原位置，不会被删除或移动

#### 其他配置
- `max_file_size`: 最大文件大小（字节），默认 100MB

## 🔧 命令行使用

也可以直接使用命令行：

```cmd
# 安装服务（需要管理员权限）
agent.exe install

# 启动服务
agent.exe start

# 停止服务
agent.exe stop

# 重启服务
agent.exe restart

# 卸载服务（需要管理员权限）
agent.exe uninstall

# 控制台模式
agent.exe console

# 查看帮助
agent.exe help
```

## 📝 服务管理（Windows服务管理器）

1. 按 `Win + R` 打开运行
2. 输入 `services.msc` 回车
3. 找到 `文档扫描Agent` 服务
4. 右键可以：
   - 启动/停止/重启服务
   - 设置启动类型（自动/手动/禁用）
   - 查看服务属性和日志

## ❓ 常见问题

### Q: 是全盘扫描还是指定目录扫描？
A: **默认是全盘扫描**。Agent会自动检测并扫描所有可用的驱动器（C:, D:, E:等），并上传符合 file_types 的文件。

如果您只想扫描特定目录，可以在 `config.json` 中设置：
```json
{
  "full_disk_scan": false,
  "scan_paths": ["C:\\Users\\YourName\\Documents", "D:\\重要文件"]
}
```

### Q: Agent会扫描所有文件吗？
A: **不会**。Agent只扫描 `file_types` 中配置的文件类型（默认：文档、表格、PDF等），并且会排除 `exclude_patterns` 中的文件（如临时文件、日志文件）。

### Q: 为什么双击 install-service.bat 提示需要管理员权限？
A: Windows服务的安装和卸载需要管理员权限。请**右键点击文件**，选择**"以管理员身份运行"**

### Q: 服务安装后如何查看运行状态？
A:
1. 打开服务管理器（`services.msc`）
2. 或者查看日志文件（通常在程序同目录下的 `logs` 文件夹）

### Q: 如何修改配置后生效？
A: 修改 `config.json` 后，需要重启服务：
   - 服务模式：双击 `restart-service.bat`
   - 控制台模式：按 Ctrl+C 停止后重新运行

### Q: 原文件会被删除吗？
A: **不会**。Agent只是读取并上传文件副本，原文件始终保留在原位置。

### Q: 如何限制扫描的文件大小？
A: 在 `config.json` 中设置 `max_file_size`（单位：字节）：
```json
{
  "max_file_size": 104857600  // 100MB
}
```

### Q: 如何查看日志？
A: 日志文件通常保存在 `%USERPROFILE%\AppData\Local\DocScannerAgent\logs` 目录，或者使用 Windows 事件查看器查看服务日志

## 📞 技术支持

如有问题，请联系系统管理员或查看项目文档。
