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

// Package performance provides performance testing utilities and benchmarks
// for the edge-stream project.
package performance

// Version represents the performance package version
const Version = "1.0.0"

// DefaultIterations 默认基准测试迭代次数
const DefaultIterations = 1000000

// BenchmarkConfig holds configuration for performance benchmarks
type BenchmarkConfig struct {
	// Iterations specifies the number of benchmark iterations
	Iterations int
	// Timeout specifies the benchmark timeout
	Timeout string
	// MemProfile enables memory profiling
	MemProfile bool
	// CPUProfile enables CPU profiling
	CPUProfile bool
}

// DefaultBenchmarkConfig returns the default benchmark configuration
func DefaultBenchmarkConfig() *BenchmarkConfig {
	return &BenchmarkConfig{
		Iterations: DefaultIterations,
		Timeout:    "10m",
		MemProfile: true,
		CPUProfile: false,
	}
}
