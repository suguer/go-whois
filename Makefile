# Go-WHOIS Makefile

# 变量
APP_NAME := go-whois
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | cut -d ' ' -f 3)

# 构建标志
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: all build clean test lint fmt vet run help

all: clean lint test build

## build: 构建项目
build:
	@echo "构建 $(APP_NAME)..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o bin/$(APP_NAME).exe .
	@echo "构建完成: bin/$(APP_NAME).exe"

## clean: 清理构建产物
clean:
	@echo "清理构建产物..."
	@rm -rf bin/
	@go clean

## test: 运行测试
test:
	@echo "运行测试..."
	@go test -v ./...

## test-short: 运行单元测试（跳过集成测试）
test-short:
	@echo "运行单元测试..."
	@go test -short -v ./...

## test-integration: 运行集成测试
test-integration:
	@echo "运行集成测试..."
	@go test -tags=integration -v ./test/...

## coverage: 生成覆盖率报告
coverage:
	@echo "生成覆盖率报告..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告: coverage.html"

## lint: 代码检查
lint:
	@echo "运行代码检查..."
	@golangci-lint run

## fmt: 格式化代码
fmt:
	@echo "格式化代码..."
	@gofmt -w .
	@goimports -w .

## vet: 静态分析
vet:
	@echo "运行静态分析..."
	@go vet ./...

## run: 运行项目
run: build
	@./bin/$(APP_NAME).exe $(ARGS)

## serve: 启动 HTTP 服务
serve: build
	@./bin/$(APP_NAME).exe serve

## lookup: 查询域名
lookup: build
	@./bin/$(APP_NAME).exe lookup $(DOMAIN)

## docker-build: 构建 Docker 镜像
docker-build:
	@echo "构建 Docker 镜像..."
	@docker build -t $(APP_NAME):$(VERSION) .

## docker-run: 运行 Docker 容器
docker-run:
	@echo "运行 Docker 容器..."
	@docker run -p 8080:8080 $(APP_NAME):$(VERSION)

## deps: 下载依赖
deps:
	@echo "下载依赖..."
	@go mod download

## deps-update: 更新依赖
deps-update:
	@echo "更新依赖..."
	@go get -u ./...
	@go mod tidy

## deps-check: 检查依赖更新
deps-check:
	@echo "检查依赖更新..."
	@go list -m -u all

## vuln: 安全漏洞检查
vuln:
	@echo "安全漏洞检查..."
	@govulncheck ./...

## help: 显示帮助
help:
	@echo "Go-WHOIS Makefile"
	@echo ""
	@echo "用法: make [target]"
	@echo ""
	@echo "目标:"
	@echo "  all              清理、检查、测试、构建"
	@echo "  build            构建项目"
	@echo "  clean            清理构建产物"
	@echo "  test             运行测试"
	@echo "  test-short       运行单元测试"
	@echo "  test-integration 运行集成测试"
	@echo "  coverage         生成覆盖率报告"
	@echo "  lint             代码检查"
	@echo "  fmt              格式化代码"
	@echo "  vet              静态分析"
	@echo "  run              运行项目"
	@echo "  serve            启动 HTTP 服务"
	@echo "  lookup           查询域名"
	@echo "  docker-build     构建 Docker 镜像"
	@echo "  docker-run       运行 Docker 容器"
	@echo "  deps             下载依赖"
	@echo "  deps-update      更新依赖"
	@echo "  deps-check       检查依赖更新"
	@echo "  vuln             安全漏洞检查"
	@echo "  help             显示帮助"
