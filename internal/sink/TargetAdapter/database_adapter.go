package TargetAdapter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/edge-stream/internal/flowfile"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// DatabaseOutputAdapter 数据库输出适配器
type DatabaseOutputAdapter struct {
	*AbstractTargetAdapter

	// 数据库相关
	db          *sql.DB
	driverName  string
	dsn         string
	tableName   string
	batchSize   int
	enableBatch bool

	// 字段映射
	fieldMapping map[string]string

	// 事务支持
	enableTransaction bool
	transaction       *sql.Tx

	// 连接池配置
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	DriverName        string            `json:"driver_name"`
	DSN               string            `json:"dsn"`
	TableName         string            `json:"table_name"`
	BatchSize         int               `json:"batch_size"`
	EnableBatch       bool              `json:"enable_batch"`
	FieldMapping      map[string]string `json:"field_mapping"`
	EnableTransaction bool              `json:"enable_transaction"`
	MaxOpenConns      int               `json:"max_open_conns"`
	MaxIdleConns      int               `json:"max_idle_conns"`
	ConnMaxLifetime   time.Duration     `json:"conn_max_lifetime"`
}

// NewDatabaseOutputAdapter 创建数据库输出适配器
func NewDatabaseOutputAdapter(config map[string]string) *DatabaseOutputAdapter {
	adapter := &DatabaseOutputAdapter{
		AbstractTargetAdapter: NewAbstractTargetAdapter(
			config["id"],
			config["name"],
			"Database",
		),
		batchSize:         100,
		enableBatch:       false,
		fieldMapping:      make(map[string]string),
		enableTransaction: false,
		maxOpenConns:      10,
		maxIdleConns:      5,
		connMaxLifetime:   1 * time.Hour,
	}

	// 应用配置
	adapter.ApplyConfig(config)

	return adapter
}

// ApplyConfig 应用配置
func (a *DatabaseOutputAdapter) ApplyConfig(config map[string]string) {
	if driver, exists := config["driver_name"]; exists {
		a.driverName = driver
	}

	if dsn, exists := config["dsn"]; exists {
		a.dsn = dsn
	}

	if table, exists := config["table_name"]; exists {
		a.tableName = table
	}

	if batchSize, exists := config["batch_size"]; exists {
		if size, err := strconv.Atoi(batchSize); err == nil {
			a.batchSize = size
		}
	}

	if enableBatch, exists := config["enable_batch"]; exists {
		a.enableBatch = strings.ToLower(enableBatch) == "true"
	}

	if enableTx, exists := config["enable_transaction"]; exists {
		a.enableTransaction = strings.ToLower(enableTx) == "true"
	}

	if maxOpen, exists := config["max_open_conns"]; exists {
		if size, err := strconv.Atoi(maxOpen); err == nil {
			a.maxOpenConns = size
		}
	}

	if maxIdle, exists := config["max_idle_conns"]; exists {
		if size, err := strconv.Atoi(maxIdle); err == nil {
			a.maxIdleConns = size
		}
	}

	if lifetime, exists := config["conn_max_lifetime"]; exists {
		if duration, err := time.ParseDuration(lifetime); err == nil {
			a.connMaxLifetime = duration
		}
	}

	// 解析字段映射
	if fieldMapping, exists := config["field_mapping"]; exists {
		if err := json.Unmarshal([]byte(fieldMapping), &a.fieldMapping); err != nil {
			log.Printf("Failed to parse field mapping: %v", err)
		}
	}
}

// GetRequiredConfig 获取必需的配置项
func (a *DatabaseOutputAdapter) GetRequiredConfig() []string {
	return []string{"driver_name", "dsn", "table_name"}
}

// Initialize 初始化数据库适配器
func (a *DatabaseOutputAdapter) Initialize(config map[string]string) error {
	// 调用父类初始化
	if err := a.AbstractTargetAdapter.Initialize(config); err != nil {
		return err
	}

	// 验证配置
	if err := a.ValidateConfig(); err != nil {
		return err
	}

	// 连接数据库
	if err := a.connectDatabase(); err != nil {
		return err
	}

	// 验证表是否存在
	if err := a.validateTable(); err != nil {
		return err
	}

	return nil
}

// connectDatabase 连接数据库
func (a *DatabaseOutputAdapter) connectDatabase() error {
	var err error

	// 打开数据库连接
	a.db, err = sql.Open(a.driverName, a.dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// 配置连接池
	a.db.SetMaxOpenConns(a.maxOpenConns)
	a.db.SetMaxIdleConns(a.maxIdleConns)
	a.db.SetConnMaxLifetime(a.connMaxLifetime)

	// 测试连接
	if err := a.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	a.LogOutputActivity(fmt.Sprintf("Connected to database: %s", a.driverName))
	return nil
}

// validateTable 验证表是否存在
func (a *DatabaseOutputAdapter) validateTable() error {
	// 构建检查表是否存在的 SQL
	var checkSQL string
	switch a.driverName {
	case "mysql":
		checkSQL = fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = '%s'", a.tableName)
	case "postgres":
		checkSQL = fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = '%s'", a.tableName)
	default:
		// 对于其他数据库，尝试简单的 SELECT 查询
		checkSQL = fmt.Sprintf("SELECT COUNT(*) FROM %s LIMIT 1", a.tableName)
	}

	var count int
	err := a.db.QueryRow(checkSQL).Scan(&count)
	if err != nil {
		return fmt.Errorf("table validation failed: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("table does not exist: %s", a.tableName)
	}

	return nil
}

// doWrite 实现具体的数据库写入逻辑
func (a *DatabaseOutputAdapter) doWrite(flowFile interface{}) error {
	// 验证 FlowFile
	if err := a.ValidateFlowFile(flowFile); err != nil {
		return err
	}

	ff := flowFile.(*flowfile.FlowFile)

	// 解析数据
	data, err := a.parseFlowFileData(ff)
	if err != nil {
		return fmt.Errorf("failed to parse flowfile data: %w", err)
	}

	// 构建插入语句
	query, values, err := a.buildInsertQuery(data)
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	// 执行插入
	if a.enableTransaction {
		return a.executeWithTransaction(query, values)
	} else {
		return a.executeDirect(query, values)
	}
}

// parseFlowFileData 解析 FlowFile 数据
func (a *DatabaseOutputAdapter) parseFlowFileData(ff *flowfile.FlowFile) (map[string]interface{}, error) {
	// 获取内容
	content, err := ff.GetContent()
	if err != nil {
		return nil, err
	}

	// 尝试解析为 JSON
	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		// 如果不是 JSON，尝试其他格式
		return a.parseNonJSONData(content, ff)
	}

	// 添加 FlowFile 属性
	for key, value := range ff.GetAttributes() {
		data[key] = value
	}

	return data, nil
}

// parseNonJSONData 解析非 JSON 数据
func (a *DatabaseOutputAdapter) parseNonJSONData(content []byte, ff *flowfile.FlowFile) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// 添加原始内容
	data["content"] = string(content)
	data["content_length"] = len(content)

	// 添加 FlowFile 属性
	for key, value := range ff.GetAttributes() {
		data[key] = value
	}

	// 添加时间戳
	data["timestamp"] = time.Now()
	data["uuid"] = ff.GetAttribute("uuid")

	return data, nil
}

// buildInsertQuery 构建插入查询
func (a *DatabaseOutputAdapter) buildInsertQuery(data map[string]interface{}) (string, []interface{}, error) {
	if len(data) == 0 {
		return "", nil, fmt.Errorf("no data to insert")
	}

	// 确定要插入的字段
	var fields []string
	var placeholders []string
	var values []interface{}

	for key, value := range data {
		// 应用字段映射
		fieldName := a.mapFieldName(key)
		if fieldName == "" {
			continue // 跳过未映射的字段
		}

		fields = append(fields, fieldName)
		values = append(values, value)

		// 根据数据库类型构建占位符
		switch a.driverName {
		case "mysql":
			placeholders = append(placeholders, "?")
		case "postgres":
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(placeholders)+1))
		default:
			placeholders = append(placeholders, "?")
		}
	}

	if len(fields) == 0 {
		return "", nil, fmt.Errorf("no valid fields to insert")
	}

	// 构建 SQL 语句
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		a.tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)

	return query, values, nil
}

// mapFieldName 映射字段名
func (a *DatabaseOutputAdapter) mapFieldName(originalName string) string {
	// 检查是否有显式映射
	if mappedName, exists := a.fieldMapping[originalName]; exists {
		return mappedName
	}

	// 如果没有映射，使用原始名称
	return originalName
}

// executeWithTransaction 使用事务执行
func (a *DatabaseOutputAdapter) executeWithTransaction(query string, values []interface{}) error {
	// 开始事务
	tx, err := a.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 执行插入
	_, err = tx.Exec(query, values...)
	if err != nil {
		// 回滚事务
		tx.Rollback()
		return fmt.Errorf("failed to execute insert in transaction: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// executeDirect 直接执行
func (a *DatabaseOutputAdapter) executeDirect(query string, values []interface{}) error {
	_, err := a.db.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("failed to execute insert: %w", err)
	}

	return nil
}

// performHealthCheck 执行健康检查
func (a *DatabaseOutputAdapter) performHealthCheck() bool {
	if a.db == nil {
		return false
	}

	// 执行简单的查询来检查连接
	err := a.db.QueryRow("SELECT 1").Scan(new(int))
	return err == nil
}

// Close 关闭数据库连接
func (a *DatabaseOutputAdapter) Close() error {
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			return fmt.Errorf("failed to close database connection: %w", err)
		}
	}

	return a.AbstractTargetAdapter.Close()
}

// GetTableInfo 获取表信息
func (a *DatabaseOutputAdapter) GetTableInfo() ([]ColumnInfo, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var query string
	switch a.driverName {
	case "mysql":
		query = fmt.Sprintf("DESCRIBE %s", a.tableName)
	case "postgres":
		query = fmt.Sprintf(`
			SELECT column_name, data_type, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_name = '%s'
			ORDER BY ordinal_position
		`, a.tableName)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", a.driverName)
	}

	rows, err := a.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query table info: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var column ColumnInfo
		if err := rows.Scan(&column.Name, &column.Type, &column.Nullable, &column.Default); err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}
		columns = append(columns, column)
	}

	return columns, nil
}

// ColumnInfo 列信息
type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable string `json:"nullable"`
	Default  string `json:"default"`
}

// BatchInsert 批量插入
func (a *DatabaseOutputAdapter) BatchInsert(flowFiles []*flowfile.FlowFile) error {
	if !a.enableBatch {
		return fmt.Errorf("batch mode not enabled")
	}

	if len(flowFiles) == 0 {
		return nil
	}

	// 开始事务
	tx, err := a.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin batch transaction: %w", err)
	}

	// 准备语句
	stmt, err := tx.Prepare(a.buildBatchInsertQuery())
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer stmt.Close()

	// 执行批量插入
	for _, ff := range flowFiles {
		data, err := a.parseFlowFileData(ff)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to parse flowfile data: %w", err)
		}

		_, values, err := a.buildInsertQuery(data)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to build insert query: %w", err)
		}

		_, err = stmt.Exec(values...)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute batch insert: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch transaction: %w", err)
	}

	return nil
}

// buildBatchInsertQuery 构建批量插入查询
func (a *DatabaseOutputAdapter) buildBatchInsertQuery() string {
	// 这里应该根据实际的字段映射构建查询
	// 为了简化，返回一个通用的查询模板
	return fmt.Sprintf("INSERT INTO %s (content, timestamp, uuid) VALUES (?, ?, ?)", a.tableName)
}
