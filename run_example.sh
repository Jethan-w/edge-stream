#!/bin/bash

# EdgeStream 综合示例运行脚本

echo "=== EdgeStream 综合示例启动脚本 ==="

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo "错误: 未找到 Go 环境，请先安装 Go"
    exit 1
fi

echo "Go 版本: $(go version)"

# 创建必要目录
echo "创建必要目录..."
mkdir -p data/input data/output
mkdir -p config
mkdir -p logs
mkdir -p state
mkdir -p bundles

echo "目录创建完成"

# 检查配置文件
if [ ! -f "config/edge-stream.properties" ]; then
    echo "警告: 配置文件不存在，将使用默认配置"
else
    echo "配置文件检查通过"
fi

# 检查 Go 模块
if [ ! -f "go.mod" ]; then
    echo "初始化 Go 模块..."
    go mod init edge-stream
fi

# 更新依赖
echo "更新 Go 依赖..."
go mod tidy

# 运行示例
echo "启动综合示例..."
echo "=================================="

go run examples/comprehensive_example.go

echo "=================================="
echo "示例运行完成"

# 显示结果
echo ""
echo "=== 结果检查 ==="

# 检查输出文件
if [ -d "data/output" ] && [ "$(ls -A data/output)" ]; then
    echo "✓ 输出文件已生成:"
    ls -la data/output/
else
    echo "⚠ 未找到输出文件"
fi

# 检查日志文件
if [ -d "logs" ] && [ "$(ls -A logs)" ]; then
    echo "✓ 日志文件已生成:"
    ls -la logs/
else
    echo "⚠ 未找到日志文件"
fi

echo ""
echo "示例运行完成！" 