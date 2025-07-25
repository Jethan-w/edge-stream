// Package performance provides performance testing utilities and benchmarks
// for the edge-stream project.
package performance

// Version represents the performance package version
const Version = "1.0.0"

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
		Iterations: 1000000,
		Timeout:    "10m",
		MemProfile: true,
		CPUProfile: false,
	}
}
