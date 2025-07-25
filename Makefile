# Edge Stream Makefile

.PHONY: build test test-unit test-coverage test-race test-bench test-profile test-quick test-ci test-standards clean run-basic run-dataprocessing run-mockdemo run-windowmanager

# Build all packages
build:
	go build ./...

# Run complete test suite
test: test-unit test-coverage test-race test-bench
	@echo "所有测试完成"

# Run unit tests
test-unit:
	@echo "运行单元测试..."
	@mkdir -p test-results
	go test -v ./... -timeout=30s | tee test-results/unit-tests.log

# Run coverage tests
test-coverage:
	@echo "运行覆盖率测试..."
	@mkdir -p test-results
	go test -v ./... -coverprofile=test-results/coverage.out -covermode=atomic
	go tool cover -html=test-results/coverage.out -o test-results/coverage.html
	@echo "覆盖率报告: test-results/coverage.html"
	go tool cover -func=test-results/coverage.out | tail -1

# Run race condition tests
test-race:
	@echo "运行竞态条件检测..."
	@mkdir -p test-results
	go test -race -v ./... -timeout=60s | tee test-results/race-tests.log

# Run benchmark tests
test-bench:
	@echo "运行性能基准测试..."
	@mkdir -p test-results
	go test -bench=. -benchmem -v ./... -timeout=120s | tee test-results/benchmark.log

# Run performance profiling
test-profile:
	@echo "运行性能分析..."
	@mkdir -p test-results
	@echo "内存分析..."
	go test -memprofile=test-results/mem.prof -v ./internal/... -timeout=60s
	@echo "CPU分析..."
	go test -cpuprofile=test-results/cpu.prof -v ./internal/metrics -run=^$$ -bench=BenchmarkMetricCollector
	@echo "性能分析文件:"
	@echo "  内存: test-results/mem.prof"
	@echo "  CPU:  test-results/cpu.prof"
	@echo "查看分析: go tool pprof test-results/mem.prof"

# Quick tests (unit + coverage)
test-quick: test-unit test-coverage
	@echo "快速测试完成"

# CI/CD tests (all checks)
test-ci: fmt vet test
	@echo "CI/CD 测试完成"

# Performance standards check
test-standards:
	@echo "检查性能标准..."
	@go test -bench=BenchmarkConfigManager ./internal/config -benchtime=1s
	@go test -bench=BenchmarkMetricCollector ./internal/metrics -benchtime=1s
	@go test -bench=BenchmarkStreamEngine ./internal/stream -benchtime=1s
	@go test -bench=BenchmarkStateManager ./internal/state -benchtime=1s
	@echo "性能标准检查完成"



# Run complete example
run-example:
	go run ./examples/complete/main.go

# Run basic framework example
run-basic:
	go run ./examples/basic_framework/main.go

# Run mock demo
run-mockdemo:
	go run ./examples/mockdemo/main.go

# Run window manager example
run-windowmanager:
	go run ./examples/windowmanager/main.go

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Clean build artifacts and test results
clean:
	go clean ./...
	rm -f *.exe
	rm -rf test-results/
	go clean -cache
	go clean -testcache

# Show help
help:
	@echo "Edge Stream 可用命令:"
	@echo "构建相关:"
	@echo "  build            - 构建所有包"
	@echo "  clean            - 清理构建文件和测试结果"
	@echo "  fmt              - 格式化代码"
	@echo "  vet              - 代码检查"
	@echo "  deps             - 安装依赖"
	@echo ""
	@echo "测试相关:"
	@echo "  test             - 运行完整测试套件"
	@echo "  test-unit        - 运行单元测试"
	@echo "  test-coverage    - 运行覆盖率测试"
	@echo "  test-race        - 运行竞态条件检测"
	@echo "  test-bench       - 运行性能基准测试"
	@echo "  test-profile     - 运行性能分析"
	@echo "  test-quick       - 快速测试 (单元+覆盖率)"
	@echo "  test-ci          - CI/CD 测试 (所有检查)"
	@echo "  test-standards   - 性能标准检查"
	@echo ""
	@echo "运行示例:"
	@echo "  run-example      - 运行完整示例"
	@echo "  run-basic        - 运行基础框架示例"
	@echo "  run-mockdemo     - 运行模拟演示"
	@echo "  run-windowmanager - 运行窗口管理器示例"
	@echo ""
	@echo "  help             - 显示此帮助信息"