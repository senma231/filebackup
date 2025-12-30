# 构建阶段
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 设置Go代理以加速下载
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
ENV GO111MODULE=on

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 安装编译需要的工具（包括 GCC）
RUN apk add --no-cache git gcc musl-dev

# 下载依赖并整理
RUN go mod download && go mod tidy

# 复制源代码
COPY . .

# 编译 Server（需要 CGO 支持 SQLite）
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -mod=mod -o server ./cmd

# NOTE: `agent.exe` should be produced by CI (GitHub Actions) and copied to `agent_bin/agent.exe`.
# The CI will compile the Windows binary and copy it to server/agent_bin/ directory
# so the COPY command below can pick it up from the build context.

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 安装必要的工具（包括7z用于生成自解压EXE）
RUN apk add --no-cache ca-certificates 7zip

# 创建数据目录
RUN mkdir -p /app/data /app/bin

# 从构建阶段复制编译好的程序
COPY --from=builder /app/server /app/
COPY --from=builder /app/web /app/web/

# 从构建上下文复制预编译的agent.exe
COPY agent_bin/agent.exe /app/bin/agent.exe

# 暴露端口
EXPOSE 8889

# 运行 Server
CMD ["./server"]
