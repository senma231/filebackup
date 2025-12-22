@echo off
chcp 65001 >nul
setlocal

echo ========================================
echo   文档扫描Agent - 服务安装脚本
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

echo [提示] 正在安装Windows服务...
echo.

:: 安装服务
agent.exe install

if %errorlevel% equ 0 (
    echo.
    echo ========================================
    echo   安装成功！
    echo ========================================
    echo.
    echo 后续操作：
    echo   1. 启动服务：双击 start-service.bat
    echo   2. 停止服务：双击 stop-service.bat
    echo   3. 卸载服务：双击 uninstall-service.bat
    echo.
    echo 或者使用Windows服务管理器（services.msc）管理
    echo.
) else (
    echo.
    echo [错误] 安装失败，请检查错误信息
    echo.
)

pause
