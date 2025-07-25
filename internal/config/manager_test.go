package config

import (
	"os"
	"testing"
	"time"
)

func TestStandardConfigManager(t *testing.T) {
	cm := NewStandardConfigManager("")

	// 测试基本的设置和获取
	t.Run("BasicSetGet", func(t *testing.T) {
		cm.Set("test.string", "hello")
		cm.Set("test.int", 42)
		cm.Set("test.bool", true)
		cm.Set("test.duration", "30s")

		if got := cm.GetString("test.string"); got != "hello" {
			t.Errorf("GetString() = %v, want %v", got, "hello")
		}

		if got := cm.GetInt("test.int"); got != 42 {
			t.Errorf("GetInt() = %v, want %v", got, 42)
		}

		if got := cm.GetBool("test.bool"); got != true {
			t.Errorf("GetBool() = %v, want %v", got, true)
		}

		if got := cm.GetDuration("test.duration"); got != 30*time.Second {
			t.Errorf("GetDuration() = %v, want %v", got, 30*time.Second)
		}
	})

	// 测试默认值
	t.Run("DefaultValues", func(t *testing.T) {
		if got := cm.GetString("nonexistent"); got != "" {
			t.Errorf("GetString() for nonexistent key = %v, want empty string", got)
		}

		if got := cm.GetInt("nonexistent"); got != 0 {
			t.Errorf("GetInt() for nonexistent key = %v, want 0", got)
		}

		if got := cm.GetBool("nonexistent"); got != false {
			t.Errorf("GetBool() for nonexistent key = %v, want false", got)
		}
	})

	// 测试配置文件加载
	t.Run("LoadConfig", func(t *testing.T) {
		// 创建临时配置文件
		tempFile, err := os.CreateTemp("", "test_config_*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tempFile.Name())

		configContent := `
test:
  name: "test_app"
  port: 8080
  enabled: true
`
		if _, err := tempFile.WriteString(configContent); err != nil {
			t.Fatal(err)
		}
		tempFile.Close()

		if err := cm.LoadConfig(tempFile.Name()); err != nil {
			t.Errorf("LoadConfig() error = %v", err)
		}

		if got := cm.GetString("test.name"); got != "test_app" {
			t.Errorf("GetString() after LoadConfig = %v, want %v", got, "test_app")
		}
	})
}

func TestConfigManagerConcurrency(t *testing.T) {
	cm := NewStandardConfigManager("")

	// 并发读写测试
	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		done := make(chan bool)
		go func() {
			for i := 0; i < 100; i++ {
				cm.Set("concurrent.test", i)
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 100; i++ {
				cm.GetInt("concurrent.test")
			}
			done <- true
		}()

		<-done
		<-done
	})
}

func BenchmarkConfigManager(b *testing.B) {
	cm := NewStandardConfigManager("")
	cm.Set("benchmark.test", "value")

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cm.Set("benchmark.key", i)
		}
	})

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cm.GetString("benchmark.test")
		}
	})
}
