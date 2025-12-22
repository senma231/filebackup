@echo off
chcp 65001 >nul
setlocal

echo ========================================
echo   文档扫描Agent - 服务卸载脚本
echo ========================================
echo.

:: 检查管理员权限
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 此脚本需要管理员权限！
    echo.
    echo 请右键点击此文件，选择"以管理员身份运行"
    echo.
    pause
    exit /b 1
)

echo [警告] 即将卸载Windows服务
echo.
set /p confirm="确认卸载吗？(Y/N): "

if /i not "%confirm%"=="Y" (
    echo.
    echo 已取消卸载
    pause
    exit /b 0
)

echo.
echo [提示] 正在卸载服务...
echo.

agent.exe uninstall

if %errorlevel% equ 0 (
    echo.
    echo [成功] 服务已卸载
    echo.
) else (
    echo.
    echo [错误] 卸载失败，请检查错误信息
    echo.
)

pause
