# GitHub Actions 构建错误排查指南

## 可能的错误原因

基于我们新增的代码，GitHub Actions构建可能失败的原因：

### 1. Go模块依赖下载失败

**问题**: `go mod download` 失败
**原因**: 新增的 `github.com/studio-b12/gowebdav v0.9.0` 依赖无法下载

**解决方案**:
在agent目录下需要有go.sum文件记录依赖哈希值。

**修复命令** (需要在有Go环境的机器上执行):
```bash
cd agent
go mod tidy
git add go.sum
git commit -m "chore(agent): add go.sum for WebDAV dependency"
git push
```

### 2. 导入路径问题

**问题**: 编译时提示找不到模块
**原因**: 新文件中的import路径不正确

**已修复**: 所有文件已使用 `doc-scanner-agent/internal/...` 作为导入路径

### 3. 接口实现不完整

**问题**: 编译错误提示某个类型没有实现接口方法
**原因**: Uploader接口的方法在实现类中缺失

**检查项**:
- SFTPUploader 是否实现了所有 Uploader 接口方法 ✅
- WebDAVUploader 是否实现了所有 Uploader 接口方法 ✅
- LocalUploader 是否实现了所有 Uploader 接口方法 ✅

## 如何查看具体错误

### 方法1: 通过GitHub网页查看
1. 打开 https://github.com/senma231/filebackup/actions
2. 点击最新的workflow运行
3. 展开"Build agent (windows/amd64)"步骤查看详细日志

### 方法2: 使用GitHub CLI (需要安装gh)
```bash
gh run list --limit 1
gh run view --log
```

### 方法3: 查看Actions徽章
在README.md中可以添加徽章查看状态：
```markdown
![Build Status](https://github.com/senma231/filebackup/actions/workflows/build-and-push.yml/badge.svg)
```

## 最可能的问题: go.sum缺失

我们修改了go.mod添加了新依赖，但**没有生成go.sum文件**。

### 解决步骤

#### 选项A: 本地修复（推荐）
1. 在有Go环境的Windows机器上：
```bash
cd \\192.168.1.8\CodeBase\Backup\agent
go mod tidy
git add go.sum
git commit -m "chore(agent): add go.sum for dependencies"
git push
```

#### 选项B: 修改GitHub Actions自动生成
修改 `.github/workflows/build-and-push.yml` 第33行：
```yaml
run: |
  go mod tidy
  go mod download
  go build -o agent.exe ./cmd
```

#### 选项C: 使用已有go.sum（如果存在）
检查agent目录是否有go.sum文件：
```bash
ls -la \\192.168.1.8\CodeBase\Backup\agent/go.sum
```

如果存在但未提交：
```bash
cd \\192.168.1.8\CodeBase\Backup
git add agent/go.sum
git commit -m "chore(agent): add go.sum"
git push
```

## 快速诊断

执行以下命令查看状态：
```bash
cd \\192.168.1.8\CodeBase\Backup\agent

# 检查go.sum是否存在
ls -l go.sum

# 检查git状态
cd ..
git status
```

## 编译验证（需要Go环境）

在本地验证编译是否成功：
```bash
cd agent
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go mod tidy
go build -o agent.exe ./cmd
```

## 预期的编译输出

如果编译成功，应该看到：
- ✅ go mod download 成功下载依赖
- ✅ go build 编译成功
- ✅ 生成 agent.exe (约5-10MB)
- ✅ Docker镜像构建成功

## 后续Actions运行

修复后，Actions会：
1. ✅ 编译Windows Agent
2. ✅ 上传agent.exe作为Artifact
3. ✅ 复制到server/agent_bin/
4. ✅ 构建Docker镜像
5. ✅ 推送到ghcr.io

---

**下一步操作**:
1. 检查agent/go.sum是否存在
2. 如果不存在，需要在有Go环境的机器上运行 `go mod tidy`
3. 提交go.sum文件
