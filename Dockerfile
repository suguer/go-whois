# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/go-whois .

# 运行阶段
FROM alpine:latest

# 安装 CA 证书
RUN apk --no-cache add ca-certificates

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/go-whois .

# 复制配置文件
COPY config/ ./config/

# 暴露端口
EXPOSE 8080

# 运行
ENTRYPOINT ["./go-whois"]
CMD ["serve"]
