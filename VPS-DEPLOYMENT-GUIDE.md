# VPS Docker 部署指南

本指南说明如何在VPS上拉取GHCR镜像并部署运行Doc Scanner Server。

## 📋 前置要求

### 系统要求
- **操作系统**: Ubuntu 20.04+ / Debian 11+ / CentOS 8+
- **内存**: 最低1GB，推荐2GB+
- **磁盘**: 最低10GB可用空间
- **网络**: 公网IP或域名（用于Agent连接）

### 软件依赖
- Docker 20.10+
- Docker Compose 2.0+ (可选)
- Git (用于克隆配置)

---

## 🚀 快速部署（5分钟启动）

### 1. 安装Docker

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# 启动Docker服务
sudo systemctl start docker
sudo systemctl enable docker

# 验证安装
docker --version
```

### 2. 登录GHCR (GitHub Container Registry)

```bash
# 使用GitHub Personal Access Token登录
echo $GITHUB_TOKEN | docker login ghcr.io -u <YOUR_GITHUB_USERNAME> --password-stdin

# 或者使用交互式登录
docker login ghcr.io
# Username: <YOUR_GITHUB_USERNAME>
# Password: <YOUR_GITHUB_TOKEN>
```

> **注意**: 需要在GitHub创建Personal Access Token，权限需要包含 `read:packages`
> 创建地址: https://github.com/settings/tokens

### 3. 拉取镜像

```bash
# 拉取最新版本
docker pull ghcr.io/<ORG_OR_USERNAME>/doc-scanner-server:latest

# 或拉取特定版本（通过Git SHA）
docker pull ghcr.io/<ORG_OR_USERNAME>/doc-scanner-server:sha-abc1234
```

### 4. 准备目录结构

```bash
# 创建数据目录
sudo mkdir -p /opt/doc-scanner/{data,logs,uploads,config}

# 设置权限
sudo chown -R 1000:1000 /opt/doc-scanner
```

### 5. 创建配置文件

创建 `/opt/doc-scanner/config/server.yaml`:

```yaml
server:
  port: 8080
  host: 0.0.0.0

database:
  path: /app/data/doc-scanner.db

sftp:
  host: your-sftp-server.com
  port: 22
  username: sftpuser
  password: your-sftp-password  # 建议使用环境变量
  base_path: /uploads

logging:
  level: info
  file: /app/logs/server.log
```

### 6. 运行容器

```bash
docker run -d \
  --name doc-scanner-server \
  --restart unless-stopped \
  -p 8080:8080 \
  -v /opt/doc-scanner/data:/app/data \
  -v /opt/doc-scanner/logs:/app/logs \
  -v /opt/doc-scanner/uploads:/app/uploads \
  -v /opt/doc-scanner/config:/app/config \
  -e SERVER_PORT=8080 \
  -e DATABASE_PATH=/app/data/doc-scanner.db \
  -e SFTP_HOST=your-sftp-server.com \
  -e SFTP_PORT=22 \
  -e SFTP_USERNAME=sftpuser \
  -e SFTP_PASSWORD=your-sftp-password \
  ghcr.io/<ORG_OR_USERNAME>/doc-scanner-server:latest
```

### 7. 验证运行

```bash
# 查看容器状态
docker ps | grep doc-scanner-server

# 查看日志
docker logs -f doc-scanner-server

# 测试API
curl http://localhost:8080/api/v1/health
```

---

## 🔧 Docker Compose 部署（推荐）

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  doc-scanner-server:
    image: ghcr.io/<ORG_OR_USERNAME>/doc-scanner-server:latest
    container_name: doc-scanner-server
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./logs:/app/logs
      - ./uploads:/app/uploads
      - ./config:/app/config
    environment:
      - SERVER_PORT=8080
      - SERVER_HOST=0.0.0.0
      - DATABASE_PATH=/app/data/doc-scanner.db
      - SFTP_HOST=${SFTP_HOST}
      - SFTP_PORT=${SFTP_PORT:-22}
      - SFTP_USERNAME=${SFTP_USERNAME}
      - SFTP_PASSWORD=${SFTP_PASSWORD}
      - LOG_LEVEL=info
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    networks:
      - doc-scanner-network

networks:
  doc-scanner-network:
    driver: bridge
```

创建 `.env` 文件（与docker-compose.yml同目录）:

```bash
# SFTP配置
SFTP_HOST=your-sftp-server.com
SFTP_PORT=22
SFTP_USERNAME=sftpuser
SFTP_PASSWORD=your-sftp-password

# Server配置（可选）
SERVER_PORT=8080
LOG_LEVEL=info
```

启动服务:

```bash
# 启动
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止
docker-compose down

# 重启
docker-compose restart
```

---

## 🔒 使用systemd管理Docker容器

创建 `/etc/systemd/system/doc-scanner.service`:

```ini
[Unit]
Description=Doc Scanner Server (Docker)
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/doc-scanner
ExecStartPre=-/usr/bin/docker stop doc-scanner-server
ExecStartPre=-/usr/bin/docker rm doc-scanner-server
ExecStart=/usr/bin/docker run \
  --name doc-scanner-server \
  -p 8080:8080 \
  -v /opt/doc-scanner/data:/app/data \
  -v /opt/doc-scanner/logs:/app/logs \
  -v /opt/doc-scanner/uploads:/app/uploads \
  -v /opt/doc-scanner/config:/app/config \
  -e SERVER_PORT=8080 \
  -e DATABASE_PATH=/app/data/doc-scanner.db \
  -e SFTP_HOST=your-sftp-server.com \
  -e SFTP_PORT=22 \
  -e SFTP_USERNAME=sftpuser \
  -e SFTP_PASSWORD=your-sftp-password \
  ghcr.io/<ORG_OR_USERNAME>/doc-scanner-server:latest

ExecStop=/usr/bin/docker stop doc-scanner-server
ExecStopPost=/usr/bin/docker rm doc-scanner-server

[Install]
WantedBy=multi-user.target
```

管理服务:

```bash
# 重载systemd配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start doc-scanner

# 开机自启
sudo systemctl enable doc-scanner

# 查看状态
sudo systemctl status doc-scanner

# 查看日志
sudo journalctl -u doc-scanner -f
```

---

## 📊 环境变量说明

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `SERVER_PORT` | 否 | 8080 | Server监听端口 |
| `SERVER_HOST` | 否 | 0.0.0.0 | Server监听地址 |
| `DATABASE_PATH` | 否 | /app/data/doc-scanner.db | SQLite数据库路径 |
| `SFTP_HOST` | 是 | - | SFTP服务器地址 |
| `SFTP_PORT` | 否 | 22 | SFTP端口 |
| `SFTP_USERNAME` | 是 | - | SFTP用户名 |
| `SFTP_PASSWORD` | 是 | - | SFTP密码 |
| `SFTP_BASE_PATH` | 否 | /uploads | SFTP上传基础路径 |
| `LOG_LEVEL` | 否 | info | 日志级别 (debug/info/warn/error) |
| `LOG_FILE` | 否 | /app/logs/server.log | 日志文件路径 |

---

## 💾 数据卷说明

| 宿主机路径 | 容器内路径 | 用途 | 建议大小 |
|-----------|-----------|------|----------|
| `/opt/doc-scanner/data` | `/app/data` | SQLite数据库 | 1GB+ |
| `/opt/doc-scanner/logs` | `/app/logs` | 日志文件 | 5GB+ |
| `/opt/doc-scanner/uploads` | `/app/uploads` | 临时上传文件 | 10GB+ |
| `/opt/doc-scanner/config` | `/app/config` | 配置文件 | 100MB |

---

## 🌐 配置Nginx反向代理（可选）

安装Nginx:

```bash
sudo apt-get install nginx
```

创建 `/etc/nginx/sites-available/doc-scanner`:

```nginx
server {
    listen 80;
    server_name doc-scanner.yourdomain.com;

    # 重定向到HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name doc-scanner.yourdomain.com;

    # SSL证书配置（使用Let's Encrypt）
    ssl_certificate /etc/letsencrypt/live/doc-scanner.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/doc-scanner.yourdomain.com/privkey.pem;

    # 安全配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # 日志
    access_log /var/log/nginx/doc-scanner-access.log;
    error_log /var/log/nginx/doc-scanner-error.log;

    # 反向代理到Docker容器
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket支持
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # 静态文件缓存
    location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
        proxy_pass http://localhost:8080;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
```

启用配置:

```bash
sudo ln -s /etc/nginx/sites-available/doc-scanner /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

## 🔐 配置SSL证书（Let's Encrypt）

```bash
# 安装Certbot
sudo apt-get install certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d doc-scanner.yourdomain.com

# 自动续期
sudo certbot renew --dry-run
```

---

## 🔍 故障排查

### 容器无法启动

```bash
# 查看详细日志
docker logs doc-scanner-server

# 检查端口占用
sudo netstat -tlnp | grep 8080

# 检查数据卷权限
ls -la /opt/doc-scanner/
```

### 数据库初始化失败

```bash
# 进入容器检查
docker exec -it doc-scanner-server sh

# 查看数据库文件
ls -la /app/data/

# 手动运行迁移（如果容器支持）
docker exec -it doc-scanner-server /app/server migrate
```

### Agent无法连接Server

```bash
# 检查防火墙
sudo ufw status
sudo ufw allow 8080/tcp

# 检查Docker网络
docker network inspect bridge

# 测试端口
curl http://<VPS_IP>:8080/api/v1/health
```

### SFTP连接失败

```bash
# 测试SFTP连接
sftp -P 22 sftpuser@your-sftp-server.com

# 查看容器内网络
docker exec -it doc-scanner-server ping your-sftp-server.com

# 检查环境变量
docker exec -it doc-scanner-server env | grep SFTP
```

---

## 📈 监控与维护

### 查看资源使用

```bash
# 容器资源占用
docker stats doc-scanner-server

# 磁盘使用
du -sh /opt/doc-scanner/*

# 数据库大小
ls -lh /opt/doc-scanner/data/doc-scanner.db
```

### 备份数据

```bash
#!/bin/bash
# backup.sh - 每日备份脚本

BACKUP_DIR="/opt/backups/doc-scanner"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份数据库
docker exec doc-scanner-server sqlite3 /app/data/doc-scanner.db ".backup /app/data/backup_$DATE.db"
cp /opt/doc-scanner/data/backup_$DATE.db $BACKUP_DIR/

# 压缩备份
tar -czf $BACKUP_DIR/doc-scanner-backup-$DATE.tar.gz /opt/doc-scanner/data

# 删除7天前的备份
find $BACKUP_DIR -name "*.tar.gz" -mtime +7 -delete

echo "备份完成: $BACKUP_DIR/doc-scanner-backup-$DATE.tar.gz"
```

配置定时备份（crontab）:

```bash
# 编辑crontab
crontab -e

# 添加每日凌晨2点备份
0 2 * * * /opt/doc-scanner/backup.sh >> /var/log/doc-scanner-backup.log 2>&1
```

### 更新镜像

```bash
# 拉取最新镜像
docker pull ghcr.io/<ORG_OR_USERNAME>/doc-scanner-server:latest

# 停止旧容器
docker stop doc-scanner-server
docker rm doc-scanner-server

# 启动新容器（使用相同配置）
docker run -d \
  --name doc-scanner-server \
  --restart unless-stopped \
  -p 8080:8080 \
  -v /opt/doc-scanner/data:/app/data \
  -v /opt/doc-scanner/logs:/app/logs \
  -v /opt/doc-scanner/uploads:/app/uploads \
  -v /opt/doc-scanner/config:/app/config \
  ghcr.io/<ORG_OR_USERNAME>/doc-scanner-server:latest

# 或使用Docker Compose
docker-compose pull
docker-compose up -d
```

---

## 📝 访问管理界面

部署完成后，访问:

- **HTTP**: http://\<VPS_IP\>:8080
- **HTTPS** (配置Nginx后): https://doc-scanner.yourdomain.com

默认登录信息:
- **用户名**: admin
- **密码**: admin123 (首次登录后请修改)

---

## 🎯 下一步

1. ✅ 配置SFTP服务器信息
2. ✅ 测试Agent连接
3. ✅ 配置域名和SSL
4. ✅ 设置定时备份
5. ✅ 监控系统运行状态

---

## 📞 问题反馈

如遇到问题，请提供以下信息:

1. 容器日志: `docker logs doc-scanner-server`
2. 系统信息: `docker info`
3. 环境变量: `docker exec doc-scanner-server env`
4. 错误截图或描述

---

**文档版本**: v1.0
**更新日期**: 2025-12-17
**维护者**: DevOps Team
