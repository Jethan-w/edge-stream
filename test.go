package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	action := strings.ToLower(os.Args[1])

	switch action {
	case "coverage":
		runCoverage()
	case "benchmark":
		runBenchmark()
	case "build":
		runBuild()
	case "all":
		runAll()
	default:
		fmt.Printf("未知选项: %s\n", action)
		showUsage()
		os.Exit(1)
	}

	fmt.Println("\n测试脚本执行完成。")
}

func showUsage() {
	fmt.Println("使用方法: go run test.go [coverage|benchmark|build|all]")
	fmt.Println("")
	fmt.Println("可用选项:")
	fmt.Println("  coverage   - 运行覆盖率测试")
	fmt.Println("  benchmark  - 运行性能基准测试")
	fmt.Println("  build      - 运行构建检查")
	fmt.Println("  all        - 运行所有测试")
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCoverage() {
	fmt.Println("运行覆盖率测试...")
	err := runCommand("go", "test", "-cover", "-timeout", "10m", "./internal/config", "./internal/metrics", "./internal/state", "./internal/stream")
	if err != nil {
		fmt.Println("覆盖率测试失败!")
		os.Exit(1)
	}
	fmt.Println("覆盖率测试完成!")
}

func runBenchmark() {
	fmt.Println("运行性能基准测试...")

	// 切换到 internal/performance 目录
	originalDir, _ := os.Getwd()
	perfDir := filepath.Join(originalDir, "internal", "performance")
	err := os.Chdir(perfDir)
	if err != nil {
		fmt.Printf("无法切换到目录 %s: %v\n", perfDir, err)
		os.Exit(1)
	}

	// 运行基准测试
	err = runCommand("go", "test", "-bench=BenchmarkConfigManager|BenchmarkMetricsCollector|BenchmarkStateManager", "-benchmem", "-run=^$", "-timeout", "10m", "-v")

	// 切换回原目录
	os.Chdir(originalDir)

	if err != nil {
		fmt.Println("基准测试失败!")
		os.Exit(1)
	}
	fmt.Println("基准测试完成!")
}

func runBuild() {
	fmt.Println("运行构建检查...")
	err := runCommand("go", "build", "./...")
	if err != nil {
		fmt.Println("构建检查失败!")
		os.Exit(1)
	}
	fmt.Println("构建检查完成!")
}

func runAll() {
	fmt.Println("运行所有测试...")
	fmt.Println("")

	fmt.Println("[1/3] 运行覆盖率测试...")
	runCoverage()
	fmt.Println("")

	fmt.Println("[2/3] 运行性能基准测试...")
	runBenchmark()
	fmt.Println("")

	fmt.Println("[3/3] 运行构建检查...")
	runBuild()
	fmt.Println("")

	fmt.Println("所有测试完成!")
}
