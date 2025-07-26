// Copyright 2025 EdgeStream Team
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	packages := []string{"./internal/config", "./internal/metrics", "./internal/state", "./internal/stream"}
	args := append([]string{"test", "-cover", "-timeout", "10m"}, packages...)
	err := runCommand("go", args...)
	if err != nil {
		fmt.Println("覆盖率测试失败!")
		os.Exit(1)
	}
	fmt.Println("覆盖率测试完成!")
}

func runBenchmark() {
	fmt.Println("运行性能基准测试...")

	// 切换到 internal/performance 目录
	originalDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("无法获取当前目录: %v\n", err)
		os.Exit(1)
	}
	perfDir := filepath.Join(originalDir, "internal", "performance")
	err = os.Chdir(perfDir)
	if err != nil {
		fmt.Printf("无法切换到目录 %s: %v\n", perfDir, err)
		os.Exit(1)
	}

	// 运行基准测试
	benchPattern := "BenchmarkConfigManager|BenchmarkMetricsCollector|BenchmarkStateManager"
	err = runCommand("go", "test", "-bench="+benchPattern, "-benchmem", "-run=^$", "-timeout", "10m", "-v")

	// 切换回原目录
	if chdirErr := os.Chdir(originalDir); chdirErr != nil {
		fmt.Printf("无法切换回原目录: %v\n", chdirErr)
	}

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
