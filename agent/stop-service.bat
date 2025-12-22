@echo off
chcp 65001 >nul
setlocal

echo ========================================
echo   文档扫描Agent - 停止服务
echo ========================================
echo.

agent.exe stop

if %errorlevel% equ 0 (
    echo.
    echo [成功] 服务已停止
    echo.
) else (
    echo.
    echo [错误] 停止失败，请检查错误信息
    echo.
)

pause
