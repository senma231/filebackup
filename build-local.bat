@echo off
chcp 65001 >nul
setlocal

echo ========================================
echo   本地构建脚本 - Agent + Server
echo ========================================
echo.

REM 检查 Go 是否安装
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 未找到 Go，请先安装 Go: https://golang.org/dl/
    pause
    exit /b 1
)

echo [步骤 1/3] 编译 Windows Agent (amd64)...
cd /d "%~dp0agent"
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0

go mod tidy
if %errorlevel% neq 0 (
    echo [错误] Go mod tidy 失败
    pause
    exit /b 1
)

go build -v -ldflags="-s -w" -trimpath -o agent.exe ./cmd
if %errorlevel% neq 0 (
    echo [错误] Agent 编译失败
    pause
    exit /b 1
)

echo [成功] Agent 编译完成: agent/agent.exe
echo.

cd /d "%~dp0"

echo [步骤 2/3] 创建 server/agent_bin 目录...
if not exist "server\agent_bin" mkdir server\agent_bin

echo [步骤 3/3] 复制 agent.exe 到 server/agent_bin...
copy /Y "agent\agent.exe" "server\agent_bin\agent.exe" >nul
if %errorlevel% neq 0 (
    echo [错误] 复制失败
    pause
    exit /b 1
)

echo.
echo ========================================
echo   构建完成！
echo ========================================
echo.
echo 生成的文件：
echo   - agent\agent.exe
echo   - server\agent_bin\agent.exe
echo.
echo 现在可以：
echo   1. 启动 Server: cd server && go run cmd/main.go
echo   2. 访问 http://localhost:8889/download 下载 Agent
echo   3. 下载的 ZIP 包将包含 agent.exe
echo.
pause
