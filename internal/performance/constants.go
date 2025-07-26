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

package performance

// Benchmark test constants
const (
	// BenchmarkDataSetSize 基准测试数据集大小
	BenchmarkDataSetSize = 1000

	// BenchmarkLargeDataSize 大数据测试的数据大小（字节）
	BenchmarkLargeDataSize = 1024

	// BenchmarkBatchSize 批量操作的批次大小
	BenchmarkBatchSize = 10

	// BenchmarkParallelWorkers 并行测试的工作线程数
	BenchmarkParallelWorkers = 4

	// BenchmarkConcurrentOperations 并发测试的操作数
	BenchmarkConcurrentOperations = 3

	// BenchmarkStateOperations 状态操作的并发数
	BenchmarkStateOperations = 2
)
