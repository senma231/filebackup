@echo off
chcp 65001 >nul
setlocal

echo ========================================
echo   文档扫描Agent - 控制台模式
echo ========================================
echo.
echo [提示] 以控制台模式运行（开发/调试用）
echo [提示] 按 Ctrl+C 可停止运行
echo.
echo 生产环境请使用服务模式：
echo   1. 右键以管理员身份运行 install-service.bat
echo   2. 双击 start-service.bat 启动服务
echo.
echo ========================================
echo.

agent.exe console

pause
