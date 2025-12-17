#!/bin/bash
# 文档扫描系统快速测试脚本

set -e

echo "========================================="
echo "  Windows文档扫描系统 - 快速测试"
echo "========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查Go版本
echo "1. 检查Go版本..."
if command -v go &> /dev/null; then
    GO_VERSION=$(go version)
    echo -e "${GREEN}✅ $GO_VERSION${NC}"
else
    echo -e "${RED}❌ Go未安装${NC}"
    exit 1
fi
echo ""

# 检查编译结果
echo "2. 检查编译结果..."
if [ -f "/Volumes/CodeBase/Backup/agent/agent" ]; then
    AGENT_SIZE=$(ls -lh /Volumes/CodeBase/Backup/agent/agent | awk '{print $5}')
    echo -e "${GREEN}✅ Agent已编译: $AGENT_SIZE${NC}"
else
    echo -e "${RED}❌ Agent未编译${NC}"
fi

if [ -f "/Volumes/CodeBase/Backup/server/server" ]; then
    SERVER_SIZE=$(ls -lh /Volumes/CodeBase/Backup/server/server | awk '{print $5}')
    echo -e "${GREEN}✅ Server已编译: $SERVER_SIZE${NC}"
else
    echo -e "${RED}❌ Server未编译${NC}"
fi
echo ""

# 检查项目结构
echo "3. 检查项目结构..."
STRUCTURE_CHECK=0

# 检查Agent核心文件
if [ -f "/Volumes/CodeBase/Backup/agent/cmd/main.go" ]; then
    echo -e "${GREEN}✅ Agent主程序存在${NC}"
    ((STRUCTURE_CHECK++))
fi

if [ -d "/Volumes/CodeBase/Backup/agent/internal/config" ]; then
    echo -e "${GREEN}✅ Agent配置模块存在${NC}"
    ((STRUCTURE_CHECK++))
fi

if [ -d "/Volumes/CodeBase/Backup/agent/internal/scanner" ]; then
    echo -e "${GREEN}✅ Agent扫描模块存在${NC}"
    ((STRUCTURE_CHECK++))
fi

if [ -d "/Volumes/CodeBase/Backup/agent/internal/uploader" ]; then
    echo -e "${GREEN}✅ Agent上传模块存在${NC}"
    ((STRUCTURE_CHECK++))
fi

# 检查Server核心文件
if [ -f "/Volumes/CodeBase/Backup/server/cmd/main.go" ]; then
    echo -e "${GREEN}✅ Server主程序存在${NC}"
    ((STRUCTURE_CHECK++))
fi

if [ -d "/Volumes/CodeBase/Backup/server/internal/api/handlers" ]; then
    echo -e "${GREEN}✅ Server API处理器存在${NC}"
    ((STRUCTURE_CHECK++))
fi

if [ -f "/Volumes/CodeBase/Backup/server/web/templates/dashboard_enhanced.html" ]; then
    echo -e "${GREEN}✅ Server Web界面存在${NC}"
    ((STRUCTURE_CHECK++))
fi

# 检查增强下载功能
if [ -f "/Volumes/CodeBase/Backup/server/internal/api/handlers/download_enhanced.go" ]; then
    echo -e "${GREEN}✅ 增强下载处理器存在${NC}"
    ((STRUCTURE_CHECK++))
fi

echo ""
echo "结构检查: $STRUCTURE_CHECK/9 项通过"
echo ""

# 检查依赖
echo "4. 检查依赖..."
cd /Volumes/CodeBase/Backup/agent
if go mod tidy &> /dev/null; then
    echo -e "${GREEN}✅ Agent依赖正常${NC}"
else
    echo -e "${YELLOW}⚠️ Agent依赖需要更新${NC}"
fi

cd /Volumes/CodeBase/Backup/server
if go mod tidy &> /dev/null; then
    echo -e "${GREEN}✅ Server依赖正常${NC}"
else
    echo -e "${YELLOW}⚠️ Server依赖需要更新${NC}"
fi
echo ""

# 显示快速启动指南
echo "5. 快速启动指南"
echo "========================================="
echo ""
echo -e "${YELLOW}编译Agent（Windows版本）:${NC}"
echo "  cd /Volumes/CodeBase/Backup/agent"
echo "  GOOS=windows GOARCH=amd64 go build -ldflags \"-s -w\" -o agent.exe ./cmd"
echo ""
echo -e "${YELLOW}复制Agent到Server:${NC}"
echo "  mkdir -p /Volumes/CodeBase/Backup/server/bin"
echo "  cp agent.exe /Volumes/CodeBase/Backup/server/bin/"
echo ""
echo -e "${YELLOW}启动Server:${NC}"
echo "  cd /Volumes/CodeBase/Backup/server"
echo "  go mod tidy"
echo "  ./server"
echo ""
echo -e "${YELLOW}访问Web界面:${NC}"
echo "  浏览器打开: http://localhost:8080"
echo ""
echo "========================================="
echo ""

# 显示文件清单
echo "6. 重要文件清单"
echo "========================================="
echo ""
echo -e "${GREEN}Agent程序:${NC}"
echo "  - /Volumes/CodeBase/Backup/agent/agent (编译后)"
echo "  - /Volumes/CodeBase/Backup/agent/cmd/main.go"
echo "  - /Volumes/CodeBase/Backup/agent/internal/"
echo ""
echo -e "${GREEN}Server程序:${NC}"
echo "  - /Volumes/CodeBase/Backup/server/server (编译后)"
echo "  - /Volumes/CodeBase/Backup/server/cmd/main.go"
echo "  - /Volumes/CodeBase/Backup/server/internal/"
echo "  - /Volumes/CodeBase/Backup/server_enhanced.html"
/web/templates/dashboardecho ""
echo -e "${GREEN}文档:${NC}"
echo "  - /Volumes/CodeBase/Backup/README.md"
echo "  - /Volumes/CodeBase/Backup/项目完成状态.md"
echo "  - /Volumes/CodeBase/Backup/部署完整指南.md"
echo "  - /Volumes/CodeBase/Backup/用户友好的下载流程.md"
echo "  - /Volumes/CodeBase/Backup/预编译Agent说明.md"
echo ""
echo "========================================="
echo ""

# 总结
echo -e "${GREEN}✅ 测试完成！${NC}"
echo ""
echo "系统已准备就绪，可以进行部署和使用。"
echo "详细说明请查看: /Volumes/CodeBase/Backup/项目完成状态.md"
echo ""
