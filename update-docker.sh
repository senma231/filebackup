#!/bin/bash

echo "=== 更新 Docker 容器 ==="

# 1. 停止并删除旧容器
echo "1. 停止旧容器..."
docker stop doc-scanner-server 2>/dev/null || true
docker rm doc-scanner-server 2>/dev/null || true

# 2. 拉取最新镜像
echo "2. 拉取最新镜像..."
docker pull ghcr.io/senma231/doc-scanner-server:latest

# 3. 启动新容器
echo "3. 启动新容器..."
docker run -d \
  --name doc-scanner-server \
  -p 8889:8889 \
  -v doc-scanner-data:/app/data \
  --restart unless-stopped \
  ghcr.io/senma231/doc-scanner-server:latest

# 4. 检查容器状态
echo "4. 检查容器状态..."
docker ps | grep doc-scanner-server

# 5. 查看日志
echo ""
echo "5. 最近的日志："
docker logs --tail 20 doc-scanner-server

echo ""
echo "✅ 更新完成！"
echo "访问: http://localhost:8889"
echo ""
echo "查看完整日志: docker logs -f doc-scanner-server"
