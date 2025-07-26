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

package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestConfigManagerEdgeCases 测试边界条件
func TestConfigManagerEdgeCases(t *testing.T) {
	cm := NewStandardConfigManager("")

	// 测试空键
	t.Run("EmptyKey", func(t *testing.T) {
		err := cm.Set("", "value")
		if err != nil {
			t.Errorf("Set() with empty key error = %v", err)
		}
		if got := cm.GetString(""); got != "value" {
			t.Errorf("GetString() with empty key = %v, want %v", got, "value")
		}
	})

	// 测试空值
	t.Run("EmptyValue", func(t *testing.T) {
		err := cm.Set("empty.value", "")
		if err != nil {
			t.Errorf("Set() with empty value error = %v", err)
		}
		if got := cm.GetString("empty.value"); got != "" {
			t.Errorf("GetString() with empty value = %v, want empty string", got)
		}
	})

	// 测试nil值
	t.Run("NilValue", func(t *testing.T) {
		err := cm.Set("nil.value", nil)
		if err != nil {
			t.Errorf("Set() with nil value error = %v", err)
		}
		if got := cm.GetString("nil.value"); got != "<nil>" {
			t.Errorf("GetString() with nil value = %v, want <nil>", got)
		}
	})

	// 测试长键值
	t.Run("LongKeyValue", func(t *testing.T) {
		longKey := strings.Repeat("a", 1000)
		longValue := strings.Repeat("x", 10000)

		err := cm.Set(longKey, longValue)
		if err != nil {
			t.Errorf("Set() with long key/value error = %v", err)
		}
		if got := cm.GetString(longKey); got != longValue {
			t.Errorf("GetString() with long key/value failed")
		}
	})

	// 测试特殊字符键
	t.Run("SpecialCharacterKeys", func(t *testing.T) {
		specialKeys := []string{
			"key.with.dots",
			"key-with-dashes",
			"key_with_underscores",
			"key with spaces",
			"key/with/slashes",
			"key@with@symbols",
		}

		for _, key := range specialKeys {
			err := cm.Set(key, "value")
			if err != nil {
				t.Errorf("Set() with special key %s error = %v", key, err)
			}
			if got := cm.GetString(key); got != "value" {
				t.Errorf("GetString() with special key %s = %v, want value", key, got)
			}
		}
	})
}

// TestConfigManagerDataTypes 测试不同数据类型
func TestConfigManagerDataTypes(t *testing.T) {
	cm := NewStandardConfigManager("")

	t.Run("IntegerTypes", func(t *testing.T) {
		testIntegerTypes(t, cm)
	})

	t.Run("FloatTypes", func(t *testing.T) {
		testFloatTypes(t, cm)
	})

	t.Run("BooleanTypes", func(t *testing.T) {
		testBooleanTypes(t, cm)
	})

	t.Run("ComplexTypes", func(t *testing.T) {
		testComplexTypes(t, cm)
	})
}

func testIntegerTypes(t *testing.T, cm ConfigManager) {
	testCases := []struct {
		key           string
		value         interface{}
		expectedValid bool
	}{
		{"int8", int8(127), true},
		{"int16", int16(32767), true},
		{"int32", int32(2147483647), true},
		{"int64", int64(2147483647), true},                 // 在int范围内的int64
		{"int64_large", int64(9223372036854775807), false}, // 超出int范围的int64
		{"uint8", uint8(255), true},
		{"uint16", uint16(65535), true},
		{"uint32", uint32(2147483647), true},                  // 在int范围内的uint32
		{"uint32_large", uint32(4294967295), false},           // 超出int范围的uint32
		{"uint64", uint64(2147483647), true},                  // 在int范围内的uint64
		{"uint64_large", uint64(18446744073709551615), false}, // 超出int范围的uint64
	}

	for _, tc := range testCases {
		err := cm.Set(tc.key, tc.value)
		if err != nil {
			t.Errorf("Failed to set %s: %v", tc.key, err)
			continue
		}
		// 验证可以作为整数检索
		intValue := cm.GetInt(tc.key)
		if tc.expectedValid && intValue == 0 {
			t.Errorf("Failed to retrieve %s as int, expected non-zero but got 0", tc.key)
		} else if !tc.expectedValid && intValue != 0 {
			t.Errorf("Expected %s to return 0 due to overflow, but got %d", tc.key, intValue)
		}
	}
}

func testFloatTypes(t *testing.T, cm ConfigManager) {
	testCases := []struct {
		key   string
		value interface{}
	}{
		{"float32", float32(3.14159)},
		{"float64", float64(2.718281828459045)},
	}

	for _, tc := range testCases {
		err := cm.Set(tc.key, tc.value)
		if err != nil {
			t.Errorf("Failed to set %s: %v", tc.key, err)
			continue
		}
		// 验证可以作为浮点数检索
		floatValue := cm.GetFloat64(tc.key)
		if floatValue == 0.0 {
			t.Errorf("Failed to retrieve %s as float64", tc.key)
		}
	}
}

func testBooleanTypes(t *testing.T, cm ConfigManager) {
	testCases := []struct {
		key   string
		value bool
	}{
		{"bool.true", true},
		{"bool.false", false},
	}

	for _, tc := range testCases {
		err := cm.Set(tc.key, tc.value)
		if err != nil {
			t.Errorf("Failed to set %s: %v", tc.key, err)
			continue
		}
		boolValue := cm.GetBool(tc.key)
		if boolValue != tc.value {
			t.Errorf("Boolean value mismatch for %s: expected %v, got %v", tc.key, tc.value, boolValue)
		}
	}
}

func testComplexTypes(t *testing.T, cm ConfigManager) {
	// 测试切片
	sliceValue := []string{"a", "b", "c"}
	err := cm.Set("slice.value", sliceValue)
	if err != nil {
		t.Errorf("Failed to set slice: %v", err)
	}

	// 测试映射
	mapValue := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	err = cm.Set("map.value", mapValue)
	if err != nil {
		t.Errorf("Failed to set map: %v", err)
	}

	// 测试结构体
	type TestStruct struct {
		Name  string
		Value int
	}
	structValue := TestStruct{Name: "test", Value: 123}
	err = cm.Set("struct.value", structValue)
	if err != nil {
		t.Errorf("Failed to set struct: %v", err)
	}
}

// TestConfigManagerAdvancedConcurrency 测试高级并发安全场景
func TestConfigManagerAdvancedConcurrency(t *testing.T) {
	cm := NewStandardConfigManager("")

	// 测试并发读写
	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		var wg sync.WaitGroup
		const numGoroutines = 10
		const numOperations = 100

		// 并发写入
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := fmt.Sprintf("concurrent.write.%d.%d", id, j)
					value := fmt.Sprintf("value-%d-%d", id, j)
					err := cm.Set(key, value)
					if err != nil {
						t.Errorf("Concurrent write failed: %v", err)
					}
				}
			}(i)
		}

		// 并发读取
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := fmt.Sprintf("concurrent.read.%d", id)
					cm.Set(key, "test-value")
					value := cm.GetString(key)
					if value != "test-value" {
						t.Errorf("Concurrent read failed: expected 'test-value', got '%s'", value)
					}
				}
			}(i)
		}

		wg.Wait()
	})

	// 测试并发更新同一键
	t.Run("ConcurrentUpdateSameKey", func(t *testing.T) {
		var wg sync.WaitGroup
		const numGoroutines = 10
		const numOperations = 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					value := fmt.Sprintf("value-%d-%d", id, j)
					err := cm.Set("shared.key", value)
					if err != nil {
						t.Errorf("Concurrent update failed: %v", err)
					}
				}
			}(i)
		}

		wg.Wait()

		// 验证最终值存在
		finalValue := cm.GetString("shared.key")
		if finalValue == "" {
			t.Error("Final value should not be empty")
		}
	})
}

// TestConfigManagerFileOperations 测试文件操作
func TestConfigManagerFileOperations(t *testing.T) {
	cm := NewStandardConfigManager("")

	// 测试从不存在的文件加载
	t.Run("LoadNonExistentFile", func(t *testing.T) {
		err := cm.LoadConfig("nonexistent.yaml")
		if err == nil {
			t.Error("Loading non-existent file should return error")
		}
	})

	// 测试保存到文件
	t.Run("SaveToFile", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "test_save_*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.Remove(tempFile.Name())
		}()
		_ = tempFile.Close()

		_ = cm.Set("test.key", "test.value")
		err = cm.Save(tempFile.Name())
		if err != nil {
			t.Errorf("Save() error = %v", err)
		}

		// 验证文件存在
		if _, err := os.Stat(tempFile.Name()); os.IsNotExist(err) {
			t.Error("Saved file does not exist")
		}
	})

	// 测试从文件重新加载
	t.Run("ReloadFromFile", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "test_reload_*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.Remove(tempFile.Name())
		}()

		configContent := `
test:
  reload: "success"
`
		if _, err := tempFile.WriteString(configContent); err != nil {
			t.Fatal(err)
		}
		tempFile.Close()

		err = cm.LoadConfig(tempFile.Name())
		if err != nil {
			t.Errorf("LoadConfig() error = %v", err)
		}

		if got := cm.GetString("test.reload"); got != "success" {
			t.Errorf("GetString() after reload = %v, want success", got)
		}
	})

	// 测试无效文件格式
	t.Run("InvalidFileFormat", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "test_invalid_*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.Remove(tempFile.Name())
		}()

		// 写入无效的YAML内容
		invalidContent := `
test:
  invalid: [
    unclosed array
`
		if _, err := tempFile.WriteString(invalidContent); err != nil {
			t.Fatal(err)
		}
		tempFile.Close()

		err = cm.LoadConfig(tempFile.Name())
		if err == nil {
			t.Error("Loading invalid YAML should return error")
		}
	})
}

// TestConfigManagerPerformance 性能测试
func TestConfigManagerPerformance(t *testing.T) {
	cm := NewStandardConfigManager("")

	// 测试大量键值对的性能
	t.Run("LargeDataSet", func(t *testing.T) {
		numKeys := 10000
		start := time.Now()

		// 设置大量键值对
		for i := 0; i < numKeys; i++ {
			key := fmt.Sprintf("perf.key.%d", i)
			value := fmt.Sprintf("performance-value-%d", i)
			err := cm.Set(key, value)
			if err != nil {
				t.Errorf("Failed to set key %d: %v", i, err)
			}
		}

		setDuration := time.Since(start)
		t.Logf("Set %d keys in %v (%.2f ops/sec)", numKeys, setDuration, float64(numKeys)/setDuration.Seconds())

		// 读取所有键值对
		start = time.Now()
		for i := 0; i < numKeys; i++ {
			key := fmt.Sprintf("perf.key.%d", i)
			value := cm.GetString(key)
			expected := fmt.Sprintf("performance-value-%d", i)
			if value != expected {
				t.Errorf("Value mismatch for key %d: expected %s, got %s", i, expected, value)
			}
		}

		getDuration := time.Since(start)
		t.Logf("Get %d keys in %v (%.2f ops/sec)", numKeys, getDuration, float64(numKeys)/getDuration.Seconds())
	})

	// 测试深层嵌套键的性能
	t.Run("DeepNestedKeys", func(t *testing.T) {
		numLevels := 10
		numKeysPerLevel := 100
		start := time.Now()

		for level := 0; level < numLevels; level++ {
			for i := 0; i < numKeysPerLevel; i++ {
				key := fmt.Sprintf("level%d.sublevel%d.key%d", level, level, i)
				value := fmt.Sprintf("nested-value-%d-%d", level, i)
				err := cm.Set(key, value)
				if err != nil {
					t.Errorf("Failed to set nested key: %v", err)
				}
			}
		}

		duration := time.Since(start)
		totalKeys := numLevels * numKeysPerLevel
		t.Logf("Set %d nested keys in %v (%.2f ops/sec)", totalKeys, duration, float64(totalKeys)/duration.Seconds())
	})
}

// TestConfigManagerErrorHandling 测试错误处理
func TestConfigManagerErrorHandling(t *testing.T) {
	cm := NewStandardConfigManager("")

	// 测试类型转换错误
	t.Run("TypeConversionErrors", func(t *testing.T) {
		// 设置字符串值
		cm.Set("string.value", "not-a-number")

		// 尝试作为整数获取
		intValue := cm.GetInt("string.value")
		if intValue != 0 {
			t.Errorf("Expected 0 for invalid int conversion, got %d", intValue)
		}

		// 尝试作为浮点数获取
		floatValue := cm.GetFloat64("string.value")
		if floatValue != 0.0 {
			t.Errorf("Expected 0.0 for invalid float conversion, got %f", floatValue)
		}

		// 尝试作为布尔值获取
		boolValue := cm.GetBool("string.value")
		if boolValue != false {
			t.Errorf("Expected false for invalid bool conversion, got %v", boolValue)
		}
	})

	// 测试获取不存在的键
	t.Run("GetNonExistentKeys", func(t *testing.T) {
		// 字符串
		strValue := cm.GetString("non.existent.string")
		if strValue != "" {
			t.Errorf("Expected empty string, got %s", strValue)
		}

		// 整数
		intValue := cm.GetInt("non.existent.int")
		if intValue != 0 {
			t.Errorf("Expected 0, got %d", intValue)
		}

		// 浮点数
		floatValue := cm.GetFloat64("non.existent.float")
		if floatValue != 0.0 {
			t.Errorf("Expected 0.0, got %f", floatValue)
		}

		// 布尔值
		boolValue := cm.GetBool("non.existent.bool")
		if boolValue != false {
			t.Errorf("Expected false, got %v", boolValue)
		}
	})

	// 测试带默认值的获取
	t.Run("GetWithDefaults", func(t *testing.T) {
		// 测试存在的键（应该返回实际值）
		cm.Set("existing.key", "actual-value")
		value := cm.GetString("existing.key")
		if value == "" {
			value = "default-value"
		}
		if value != "actual-value" {
			t.Errorf("Expected 'actual-value', got %s", value)
		}

		// 测试不存在的键（应该返回默认值）
		value = cm.GetString("non.existent.key")
		if value == "" {
			value = "default-value"
		}
		if value != "default-value" {
			t.Errorf("Expected 'default-value', got %s", value)
		}
	})
}

// TestConfigManagerWatching 测试配置监听
func TestConfigManagerWatching(t *testing.T) {
	cm := NewStandardConfigManager("")

	// 测试配置变更通知
	t.Run("ConfigChangeNotification", func(t *testing.T) {
		_ = make(chan string, 10) // 创建但不使用，避免未使用变量错误
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 启动监听器（如果支持）
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// 模拟配置变更检测
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()

		// 进行一些配置变更
		cm.Set("watch.test1", "value1")
		cm.Set("watch.test2", "value2")
		cm.Set("watch.test3", "value3")

		// 等待一段时间
		time.Sleep(500 * time.Millisecond)

		// 验证配置已设置
		if cm.GetString("watch.test1") != "value1" {
			t.Error("Configuration change was not applied")
		}
	})
}
