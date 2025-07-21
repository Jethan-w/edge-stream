@echo off
chcp 65001 >nul

echo === EdgeStream 综合示例启动脚本 ===

REM 检查 Go 环境
where go >nul 2>nul
if %errorlevel% neq 0 (
    echo 错误: 未找到 Go 环境，请先安装 Go
    pause
    exit /b 1
)

echo Go 版本:
go version

REM 创建必要目录
echo 创建必要目录...
if not exist "data\input" mkdir "data\input"
if not exist "data\output" mkdir "data\output"
if not exist "config" mkdir "config"
if not exist "logs" mkdir "logs"
if not exist "state" mkdir "state"
if not exist "bundles" mkdir "bundles"

echo 目录创建完成

REM 检查配置文件
if not exist "config\edge-stream.properties" (
    echo 警告: 配置文件不存在，将使用默认配置
) else (
    echo 配置文件检查通过
)

REM 检查 Go 模块
if not exist "go.mod" (
    echo 初始化 Go 模块...
    go mod init edge-stream
)

REM 更新依赖
echo 更新 Go 依赖...
go mod tidy

REM 运行示例
echo 启动综合示例...
echo ==================================

go run examples\comprehensive_example.go

echo ==================================
echo 示例运行完成

REM 显示结果
echo.
echo === 结果检查 ===

REM 检查输出文件
if exist "data\output" (
    dir "data\output" /b >nul 2>nul
    if %errorlevel% equ 0 (
        echo ✓ 输出文件已生成:
        dir "data\output"
    ) else (
        echo ⚠ 未找到输出文件
    )
) else (
    echo ⚠ 未找到输出文件
)

REM 检查日志文件
if exist "logs" (
    dir "logs" /b >nul 2>nul
    if %errorlevel% equ 0 (
        echo ✓ 日志文件已生成:
        dir "logs"
    ) else (
        echo ⚠ 未找到日志文件
    )
) else (
    echo ⚠ 未找到日志文件
)

echo.
echo 示例运行完成！
pause 