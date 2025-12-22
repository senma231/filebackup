@echo off
chcp 65001 >nul
setlocal

echo ========================================
echo   文档扫描Agent - 重启服务
echo ========================================
echo.

agent.exe restart

if %errorlevel% equ 0 (
    echo.
    echo [成功] 服务已重启
    echo.
) else (
    echo.
    echo [错误] 重启失败，请检查错误信息
    echo.
)

pause
