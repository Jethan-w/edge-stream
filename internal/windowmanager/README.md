# 窗口管理器基础框架

这是一个简化的窗口管理器实现，提供了基本的窗口化数据处理功能。

## 功能特性

- **时间窗口**: 基于时间间隔的窗口，当指定时间过去后窗口准备就绪
- **计数窗口**: 基于数据数量的窗口，当达到指定数量后窗口准备就绪
- **线程安全**: 所有操作都是线程安全的
- **简单易用**: 提供简洁的API接口

## 核心组件

### 窗口类型

- `TimeWindow`: 时间窗口，在指定时间间隔后准备就绪
- `CountWindow`: 计数窗口，在达到指定数据量后准备就绪

### 窗口管理器

- `SimpleWindowManager`: 简单窗口管理器，管理多个窗口并处理数据分发

## 使用示例

```go
package main

import (
    "fmt"
    "time"
    "github.com/crazy/edge-stream/internal/windowmanager"
)

func main() {
    // 创建窗口管理器
    manager := windowmanager.NewSimpleWindowManager()
    
    // 创建时间窗口（5秒）
    timeWindow := manager.CreateTimeWindow(5 * time.Second)
    
    // 创建计数窗口（3个数据）
    countWindow := manager.CreateCountWindow(3)
    
    // 处理数据
    manager.ProcessData("数据1")
    manager.ProcessData("数据2")
    manager.ProcessData("数据3")
    
    // 检查准备好的窗口
    readyWindows := manager.GetReadyWindows()
    for _, window := range readyWindows {
        data := window.GetData()
        fmt.Printf("窗口 %s 准备就绪，数据: %v\n", window.GetID(), data)
        
        // 重置窗口
        window.Reset()
    }
}
```

## API 文档

### Window 接口

- `GetID() string`: 获取窗口ID
- `GetWindowType() WindowType`: 获取窗口类型
- `AddData(data interface{})`: 添加数据到窗口
- `GetData() []interface{}`: 获取窗口中的所有数据
- `IsReady() bool`: 检查窗口是否准备就绪
- `Reset()`: 重置窗口状态

### WindowManager 接口

- `CreateTimeWindow(duration time.Duration) Window`: 创建时间窗口
- `CreateCountWindow(count int) Window`: 创建计数窗口
- `ProcessData(data interface{}) error`: 处理数据，将数据分发到所有窗口

### SimpleWindowManager 额外方法

- `GetReadyWindows() []Window`: 获取所有准备就绪的窗口
- `PrintWindowStatus()`: 打印所有窗口的状态信息

## 运行测试

```bash
# 运行单元测试
go test ./internal/windowmanager -v

# 运行示例程序
go run ./examples/windowmanager
```

## 扩展性

这个基础框架可以很容易地扩展：

1. 添加新的窗口类型（如滑动窗口、会话窗口）
2. 添加数据聚合功能
3. 添加触发器和输出处理器
4. 添加状态持久化功能

当前实现专注于核心功能，保持简单和可靠。