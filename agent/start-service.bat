@echo off
chcp 65001 >nul
setlocal

echo ========================================
echo   文档扫描Agent - 启动服务
echo ========================================
echo.

agent.exe start

if %errorlevel% equ 0 (
    echo.
    echo [成功] 服务已启动
    echo.
    echo 可以通过以下方式查看服务状态：
    echo   1. 打开服务管理器（services.msc）
    echo   2. 查找"文档扫描Agent"服务
    echo.
) else (
    echo.
    echo [错误] 启动失败，请检查错误信息
    echo 提示：如果服务未安装，请先运行 install-service.bat
    echo.
)

pause
