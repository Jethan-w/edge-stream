# EdgeStreamPro Makefile

# 变量定义
BINARY_NAME=edgestream-example
WINDOW_BINARY_NAME=window-example
BUILD_DIR=build
EXAMPLES_DIR=examples
CONFIG_DIR=config

# Go相关变量
GO=go
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

# 默认目标
.PHONY: all
all: clean build

# 清理构建目录
.PHONY: clean
clean:
	@echo "清理构建目录..."
	@rm -rf $(BUILD_DIR)
	@mkdir -p $(BUILD_DIR)

# 构建完整示例
.PHONY: build
build: clean
	@echo "构建完整示例..."
	@$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) $(EXAMPLES_DIR)/complete_example.go

# 构建窗口管理示例
.PHONY: build-window
build-window: clean
	@echo "构建窗口管理示例..."
	@$(GO) build -o $(BUILD_DIR)/$(WINDOW_BINARY_NAME) $(EXAMPLES_DIR)/window_example.go

# 构建所有示例
.PHONY: build-all
build-all: build build-window
	@echo "所有示例构建完成"

# 运行完整示例
.PHONY: run
run: build
	@echo "运行完整示例..."
	@mkdir -p config bundles state/local metrics logs output temp input
	@cp $(CONFIG_DIR)/application.properties ./config/ 2>/dev/null || true
	@./$(BUILD_DIR)/$(BINARY_NAME)

# 运行窗口管理示例
.PHONY: run-window
run-window: build-window
	@echo "运行窗口管理示例..."
	@mkdir -p window-state
	@./$(BUILD_DIR)/$(WINDOW_BINARY_NAME)

# 运行所有示例
.PHONY: run-all
run-all: run run-window

# 测试
.PHONY: test
test:
	@echo "运行测试..."
	@$(GO) test ./...

# 测试覆盖率
.PHONY: test-coverage
test-coverage:
	@echo "运行测试覆盖率..."
	@$(GO) test -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 代码格式化
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	@$(GO) fmt ./...

# 代码检查
.PHONY: lint
lint:
	@echo "代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint 未安装，跳过代码检查"; \
	fi

# 依赖管理
.PHONY: deps
deps:
	@echo "下载依赖..."
	@$(GO) mod download
	@$(GO) mod tidy

# 生成文档
.PHONY: docs
docs:
	@echo "生成文档..."
	@if command -v godoc >/dev/null 2>&1; then \
		godoc -http=:6060 & \
		echo "文档服务器已启动: http://localhost:6060"; \
	else \
		echo "godoc 未安装，无法生成文档"; \
	fi

# 安装
.PHONY: install
install: build
	@echo "安装到系统..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "安装完成"

# 卸载
.PHONY: uninstall
uninstall:
	@echo "从系统卸载..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "卸载完成"

# 创建发布包
.PHONY: release
release: build-all
	@echo "创建发布包..."
	@mkdir -p release
	@cp $(BUILD_DIR)/* release/
	@cp -r $(CONFIG_DIR) release/
	@cp $(EXAMPLES_DIR)/README.md release/
	@tar -czf edgestream-examples-$(shell date +%Y%m%d).tar.gz release/
	@echo "发布包已创建: edgestream-examples-$(shell date +%Y%m%d).tar.gz"

# 开发模式
.PHONY: dev
dev:
	@echo "开发模式 - 监听文件变化并自动重新构建..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air 未安装，请先安装: go install github.com/cosmtrek/air@latest"; \
	fi

# 性能分析
.PHONY: profile
profile: build
	@echo "运行性能分析..."
	@./$(BUILD_DIR)/$(BINARY_NAME) -cpuprofile=cpu.prof -memprofile=mem.prof
	@echo "性能分析文件已生成: cpu.prof, mem.prof"

# 查看性能分析结果
.PHONY: profile-view
profile-view:
	@if command -v go tool pprof >/dev/null 2>&1; then \
		echo "查看CPU性能分析..."; \
		go tool pprof cpu.prof; \
	else \
		echo "go tool pprof 不可用"; \
	fi

# 创建必要的目录
.PHONY: setup
setup:
	@echo "创建必要的目录..."
	@mkdir -p config bundles state/local metrics logs output temp input window-state
	@echo "目录创建完成"

# 检查环境
.PHONY: check-env
check-env:
	@echo "检查环境..."
	@echo "Go版本: $(shell go version)"
	@echo "操作系统: $(GOOS)"
	@echo "架构: $(GOARCH)"
	@echo "GOPATH: $(shell go env GOPATH)"
	@echo "GOROOT: $(shell go env GOROOT)"

# 帮助信息
.PHONY: help
help:
	@echo "EdgeStreamPro Makefile 帮助"
	@echo ""
	@echo "可用目标:"
	@echo "  all          - 清理并构建所有示例"
	@echo "  build        - 构建完整示例"
	@echo "  build-window - 构建窗口管理示例"
	@echo "  build-all    - 构建所有示例"
	@echo "  run          - 运行完整示例"
	@echo "  run-window   - 运行窗口管理示例"
	@echo "  run-all      - 运行所有示例"
	@echo "  test         - 运行测试"
	@echo "  test-coverage- 运行测试覆盖率"
	@echo "  fmt          - 格式化代码"
	@echo "  lint         - 代码检查"
	@echo "  deps         - 下载依赖"
	@echo "  docs         - 生成文档"
	@echo "  install      - 安装到系统"
	@echo "  uninstall    - 从系统卸载"
	@echo "  release      - 创建发布包"
	@echo "  dev          - 开发模式"
	@echo "  profile      - 性能分析"
	@echo "  profile-view - 查看性能分析结果"
	@echo "  setup        - 创建必要的目录"
	@echo "  check-env    - 检查环境"
	@echo "  help         - 显示此帮助信息"
	@echo "  clean        - 清理构建目录"

# 默认目标
.DEFAULT_GOAL := help 