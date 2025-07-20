package MetricCollector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// MetricsExporter 指标导出器接口
type MetricsExporter interface {
	// ExportToPrometheus 导出到Prometheus格式
	ExportToPrometheus(snapshot *MetricsSnapshot) (string, error)

	// ExportToOpenTSDB 导出到OpenTSDB格式
	ExportToOpenTSDB(snapshot *MetricsSnapshot) ([]byte, error)

	// ExportToGraphite 导出到Graphite格式
	ExportToGraphite(snapshot *MetricsSnapshot) ([]byte, error)

	// ExportToJMX 导出到JMX格式
	ExportToJMX(snapshot *MetricsSnapshot) ([]byte, error)

	// ExportToCSV 导出到CSV格式
	ExportToCSV(snapshot *MetricsSnapshot) ([]byte, error)

	// StartHTTPEndpoint 启动HTTP端点
	StartHTTPEndpoint(port int) error

	// StopHTTPEndpoint 停止HTTP端点
	StopHTTPEndpoint() error
}

// PrometheusExporter Prometheus导出器
type PrometheusExporter struct {
	mutex sync.RWMutex
}

// NewPrometheusExporter 创建新的Prometheus导出器
func NewPrometheusExporter() *PrometheusExporter {
	return &PrometheusExporter{}
}

// ExportToPrometheus 导出到Prometheus格式
func (pe *PrometheusExporter) ExportToPrometheus(snapshot *MetricsSnapshot) (string, error) {
	pe.mutex.RLock()
	defer pe.mutex.RUnlock()

	var result string

	// 添加时间戳注释
	result += fmt.Sprintf("# HELP nifi_metrics_timestamp The timestamp of the metrics snapshot\n")
	result += fmt.Sprintf("# TYPE nifi_metrics_timestamp gauge\n")
	result += fmt.Sprintf("nifi_metrics_timestamp %d\n\n", snapshot.Timestamp.Unix())

	// 导出Gauge指标
	for name, value := range snapshot.Gauges {
		result += fmt.Sprintf("# HELP nifi_%s NiFi Gauge Metric\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s gauge\n", name)
		result += fmt.Sprintf("nifi_%s %f\n", name, value)
	}

	// 导出Counter指标
	for name, value := range snapshot.Counters {
		result += fmt.Sprintf("# HELP nifi_%s NiFi Counter Metric\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s counter\n", name)
		result += fmt.Sprintf("nifi_%s %d\n", name, value)
	}

	// 导出Histogram指标
	for name, data := range snapshot.Histograms {
		result += fmt.Sprintf("# HELP nifi_%s_count NiFi Histogram Count\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_count counter\n", name)
		result += fmt.Sprintf("nifi_%s_count %d\n", name, data.Count)

		result += fmt.Sprintf("# HELP nifi_%s_sum NiFi Histogram Sum\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_sum counter\n", name)
		result += fmt.Sprintf("nifi_%s_sum %f\n", name, data.Sum)

		result += fmt.Sprintf("# HELP nifi_%s_min NiFi Histogram Min\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_min gauge\n", name)
		result += fmt.Sprintf("nifi_%s_min %f\n", name, data.Min)

		result += fmt.Sprintf("# HELP nifi_%s_max NiFi Histogram Max\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_max gauge\n", name)
		result += fmt.Sprintf("nifi_%s_max %f\n", name, data.Max)

		result += fmt.Sprintf("# HELP nifi_%s_mean NiFi Histogram Mean\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_mean gauge\n", name)
		result += fmt.Sprintf("nifi_%s_mean %f\n", name, data.Mean)
	}

	// 导出Timer指标
	for name, data := range snapshot.Timers {
		result += fmt.Sprintf("# HELP nifi_%s_count NiFi Timer Count\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_count counter\n", name)
		result += fmt.Sprintf("nifi_%s_count %d\n", name, data.Count)

		result += fmt.Sprintf("# HELP nifi_%s_sum NiFi Timer Sum\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_sum counter\n", name)
		result += fmt.Sprintf("nifi_%s_sum %d\n", name, data.Sum.Nanoseconds())

		result += fmt.Sprintf("# HELP nifi_%s_min NiFi Timer Min\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_min gauge\n", name)
		result += fmt.Sprintf("nifi_%s_min %d\n", name, data.Min.Nanoseconds())

		result += fmt.Sprintf("# HELP nifi_%s_max NiFi Timer Max\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_max gauge\n", name)
		result += fmt.Sprintf("nifi_%s_max %d\n", name, data.Max.Nanoseconds())

		result += fmt.Sprintf("# HELP nifi_%s_mean NiFi Timer Mean\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_mean gauge\n", name)
		result += fmt.Sprintf("nifi_%s_mean %d\n", name, data.Mean.Nanoseconds())
	}

	// 导出Meter指标
	for name, data := range snapshot.Meters {
		result += fmt.Sprintf("# HELP nifi_%s_count NiFi Meter Count\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_count counter\n", name)
		result += fmt.Sprintf("nifi_%s_count %d\n", name, data.Count)

		result += fmt.Sprintf("# HELP nifi_%s_mean_rate NiFi Meter Mean Rate\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_mean_rate gauge\n", name)
		result += fmt.Sprintf("nifi_%s_mean_rate %f\n", name, data.MeanRate)

		result += fmt.Sprintf("# HELP nifi_%s_one_minute_rate NiFi Meter 1m Rate\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_one_minute_rate gauge\n", name)
		result += fmt.Sprintf("nifi_%s_one_minute_rate %f\n", name, data.OneMinuteRate)

		result += fmt.Sprintf("# HELP nifi_%s_five_minute_rate NiFi Meter 5m Rate\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_five_minute_rate gauge\n", name)
		result += fmt.Sprintf("nifi_%s_five_minute_rate %f\n", name, data.FiveMinuteRate)

		result += fmt.Sprintf("# HELP nifi_%s_fifteen_minute_rate NiFi Meter 15m Rate\n", name)
		result += fmt.Sprintf("# TYPE nifi_%s_fifteen_minute_rate gauge\n", name)
		result += fmt.Sprintf("nifi_%s_fifteen_minute_rate %f\n", name, data.FifteenMinuteRate)
	}

	return result, nil
}

// OpenTSDBExporter OpenTSDB导出器
type OpenTSDBExporter struct {
	mutex sync.RWMutex
}

// NewOpenTSDBExporter 创建新的OpenTSDB导出器
func NewOpenTSDBExporter() *OpenTSDBExporter {
	return &OpenTSDBExporter{}
}

// ExportToOpenTSDB 导出到OpenTSDB格式
func (oe *OpenTSDBExporter) ExportToOpenTSDB(snapshot *MetricsSnapshot) ([]byte, error) {
	oe.mutex.RLock()
	defer oe.mutex.RUnlock()

	var metrics []map[string]interface{}

	// 导出Gauge指标
	for name, value := range snapshot.Gauges {
		metric := map[string]interface{}{
			"metric":    fmt.Sprintf("nifi.%s", name),
			"timestamp": snapshot.Timestamp.Unix(),
			"value":     value,
			"tags": map[string]string{
				"type": "gauge",
			},
		}
		metrics = append(metrics, metric)
	}

	// 导出Counter指标
	for name, value := range snapshot.Counters {
		metric := map[string]interface{}{
			"metric":    fmt.Sprintf("nifi.%s", name),
			"timestamp": snapshot.Timestamp.Unix(),
			"value":     value,
			"tags": map[string]string{
				"type": "counter",
			},
		}
		metrics = append(metrics, metric)
	}

	// 导出Histogram指标
	for name, data := range snapshot.Histograms {
		// Count
		metric := map[string]interface{}{
			"metric":    fmt.Sprintf("nifi.%s.count", name),
			"timestamp": snapshot.Timestamp.Unix(),
			"value":     data.Count,
			"tags": map[string]string{
				"type": "histogram",
			},
		}
		metrics = append(metrics, metric)

		// Sum
		metric = map[string]interface{}{
			"metric":    fmt.Sprintf("nifi.%s.sum", name),
			"timestamp": snapshot.Timestamp.Unix(),
			"value":     data.Sum,
			"tags": map[string]string{
				"type": "histogram",
			},
		}
		metrics = append(metrics, metric)

		// Min
		metric = map[string]interface{}{
			"metric":    fmt.Sprintf("nifi.%s.min", name),
			"timestamp": snapshot.Timestamp.Unix(),
			"value":     data.Min,
			"tags": map[string]string{
				"type": "histogram",
			},
		}
		metrics = append(metrics, metric)

		// Max
		metric = map[string]interface{}{
			"metric":    fmt.Sprintf("nifi.%s.max", name),
			"timestamp": snapshot.Timestamp.Unix(),
			"value":     data.Max,
			"tags": map[string]string{
				"type": "histogram",
			},
		}
		metrics = append(metrics, metric)

		// Mean
		metric = map[string]interface{}{
			"metric":    fmt.Sprintf("nifi.%s.mean", name),
			"timestamp": snapshot.Timestamp.Unix(),
			"value":     data.Mean,
			"tags": map[string]string{
				"type": "histogram",
			},
		}
		metrics = append(metrics, metric)
	}

	return json.Marshal(metrics)
}

// GraphiteExporter Graphite导出器
type GraphiteExporter struct {
	mutex sync.RWMutex
}

// NewGraphiteExporter 创建新的Graphite导出器
func NewGraphiteExporter() *GraphiteExporter {
	return &GraphiteExporter{}
}

// ExportToGraphite 导出到Graphite格式
func (ge *GraphiteExporter) ExportToGraphite(snapshot *MetricsSnapshot) ([]byte, error) {
	ge.mutex.RLock()
	defer ge.mutex.RUnlock()

	var lines []string
	timestamp := snapshot.Timestamp.Unix()

	// 导出Gauge指标
	for name, value := range snapshot.Gauges {
		line := fmt.Sprintf("nifi.%s %f %d", name, value, timestamp)
		lines = append(lines, line)
	}

	// 导出Counter指标
	for name, value := range snapshot.Counters {
		line := fmt.Sprintf("nifi.%s %d %d", name, value, timestamp)
		lines = append(lines, line)
	}

	// 导出Histogram指标
	for name, data := range snapshot.Histograms {
		lines = append(lines, fmt.Sprintf("nifi.%s.count %d %d", name, data.Count, timestamp))
		lines = append(lines, fmt.Sprintf("nifi.%s.sum %f %d", name, data.Sum, timestamp))
		lines = append(lines, fmt.Sprintf("nifi.%s.min %f %d", name, data.Min, timestamp))
		lines = append(lines, fmt.Sprintf("nifi.%s.max %f %d", name, data.Max, timestamp))
		lines = append(lines, fmt.Sprintf("nifi.%s.mean %f %d", name, data.Mean, timestamp))
	}

	// 导出Timer指标
	for name, data := range snapshot.Timers {
		lines = append(lines, fmt.Sprintf("nifi.%s.count %d %d", name, data.Count, timestamp))
		lines = append(lines, fmt.Sprintf("nifi.%s.sum %d %d", name, data.Sum.Nanoseconds(), timestamp))
		lines = append(lines, fmt.Sprintf("nifi.%s.min %d %d", name, data.Min.Nanoseconds(), timestamp))
		lines = append(lines, fmt.Sprintf("nifi.%s.max %d %d", name, data.Max.Nanoseconds(), timestamp))
		lines = append(lines, fmt.Sprintf("nifi.%s.mean %d %d", name, data.Mean.Nanoseconds(), timestamp))
	}

	// 导出Meter指标
	for name, data := range snapshot.Meters {
		lines = append(lines, fmt.Sprintf("nifi.%s.count %d %d", name, data.Count, timestamp))
		lines = append(lines, fmt.Sprintf("nifi.%s.mean_rate %f %d", name, data.MeanRate, timestamp))
		lines = append(lines, fmt.Sprintf("nifi.%s.one_minute_rate %f %d", name, data.OneMinuteRate, timestamp))
		lines = append(lines, fmt.Sprintf("nifi.%s.five_minute_rate %f %d", name, data.FiveMinuteRate, timestamp))
		lines = append(lines, fmt.Sprintf("nifi.%s.fifteen_minute_rate %f %d", name, data.FifteenMinuteRate, timestamp))
	}

	result := ""
	for _, line := range lines {
		result += line + "\n"
	}

	return []byte(result), nil
}

// JMXExporter JMX导出器
type JMXExporter struct {
	mutex sync.RWMutex
}

// NewJMXExporter 创建新的JMX导出器
func NewJMXExporter() *JMXExporter {
	return &JMXExporter{}
}

// ExportToJMX 导出到JMX格式
func (je *JMXExporter) ExportToJMX(snapshot *MetricsSnapshot) ([]byte, error) {
	je.mutex.RLock()
	defer je.mutex.RUnlock()

	jmxData := map[string]interface{}{
		"timestamp": snapshot.Timestamp.Unix(),
		"metrics":   make(map[string]interface{}),
	}

	// 导出Gauge指标
	for name, value := range snapshot.Gauges {
		jmxData["metrics"].(map[string]interface{})[fmt.Sprintf("nifi.%s", name)] = map[string]interface{}{
			"type":  "gauge",
			"value": value,
		}
	}

	// 导出Counter指标
	for name, value := range snapshot.Counters {
		jmxData["metrics"].(map[string]interface{})[fmt.Sprintf("nifi.%s", name)] = map[string]interface{}{
			"type":  "counter",
			"value": value,
		}
	}

	// 导出Histogram指标
	for name, data := range snapshot.Histograms {
		jmxData["metrics"].(map[string]interface{})[fmt.Sprintf("nifi.%s", name)] = map[string]interface{}{
			"type":  "histogram",
			"count": data.Count,
			"sum":   data.Sum,
			"min":   data.Min,
			"max":   data.Max,
			"mean":  data.Mean,
		}
	}

	// 导出Timer指标
	for name, data := range snapshot.Timers {
		jmxData["metrics"].(map[string]interface{})[fmt.Sprintf("nifi.%s", name)] = map[string]interface{}{
			"type":  "timer",
			"count": data.Count,
			"sum":   data.Sum.Nanoseconds(),
			"min":   data.Min.Nanoseconds(),
			"max":   data.Max.Nanoseconds(),
			"mean":  data.Mean.Nanoseconds(),
		}
	}

	// 导出Meter指标
	for name, data := range snapshot.Meters {
		jmxData["metrics"].(map[string]interface{})[fmt.Sprintf("nifi.%s", name)] = map[string]interface{}{
			"type":                "meter",
			"count":               data.Count,
			"mean_rate":           data.MeanRate,
			"one_minute_rate":     data.OneMinuteRate,
			"five_minute_rate":    data.FiveMinuteRate,
			"fifteen_minute_rate": data.FifteenMinuteRate,
		}
	}

	return json.MarshalIndent(jmxData, "", "  ")
}

// CSVExporter CSV导出器
type CSVExporter struct {
	mutex sync.RWMutex
}

// NewCSVExporter 创建新的CSV导出器
func NewCSVExporter() *CSVExporter {
	return &CSVExporter{}
}

// ExportToCSV 导出到CSV格式
func (ce *CSVExporter) ExportToCSV(snapshot *MetricsSnapshot) ([]byte, error) {
	ce.mutex.RLock()
	defer ce.mutex.RUnlock()

	var lines []string

	// CSV头部
	header := "timestamp,metric_name,metric_type,value,count,min,max,mean,sum"
	lines = append(lines, header)

	// 导出Gauge指标
	for name, value := range snapshot.Gauges {
		line := fmt.Sprintf("%d,%s,gauge,%f,,,,,", snapshot.Timestamp.Unix(), name, value)
		lines = append(lines, line)
	}

	// 导出Counter指标
	for name, value := range snapshot.Counters {
		line := fmt.Sprintf("%d,%s,counter,%d,,,,,", snapshot.Timestamp.Unix(), name, value)
		lines = append(lines, line)
	}

	// 导出Histogram指标
	for name, data := range snapshot.Histograms {
		line := fmt.Sprintf("%d,%s,histogram,,%d,%f,%f,%f,%f",
			snapshot.Timestamp.Unix(), name, data.Count, data.Min, data.Max, data.Mean, data.Sum)
		lines = append(lines, line)
	}

	// 导出Timer指标
	for name, data := range snapshot.Timers {
		line := fmt.Sprintf("%d,%s,timer,,%d,%d,%d,%d,%d",
			snapshot.Timestamp.Unix(), name, data.Count,
			data.Min.Nanoseconds(), data.Max.Nanoseconds(),
			data.Mean.Nanoseconds(), data.Sum.Nanoseconds())
		lines = append(lines, line)
	}

	// 导出Meter指标
	for name, data := range snapshot.Meters {
		line := fmt.Sprintf("%d,%s,meter,%f,%d,,,,",
			snapshot.Timestamp.Unix(), name, data.MeanRate, data.Count)
		lines = append(lines, line)
	}

	result := ""
	for _, line := range lines {
		result += line + "\n"
	}

	return []byte(result), nil
}

// HTTPMetricsExporter HTTP指标导出器
type HTTPMetricsExporter struct {
	server *http.Server
	mutex  sync.RWMutex
}

// NewHTTPMetricsExporter 创建新的HTTP指标导出器
func NewHTTPMetricsExporter() *HTTPMetricsExporter {
	return &HTTPMetricsExporter{}
}

// StartHTTPEndpoint 启动HTTP端点
func (hme *HTTPMetricsExporter) StartHTTPEndpoint(port int) error {
	hme.mutex.Lock()
	defer hme.mutex.Unlock()

	mux := http.NewServeMux()

	// Prometheus指标端点
	mux.HandleFunc("/metrics", hme.handlePrometheusMetrics)

	// JSON指标端点
	mux.HandleFunc("/metrics/json", hme.handleJSONMetrics)

	// 健康检查端点
	mux.HandleFunc("/health", hme.handleHealth)

	hme.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := hme.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP服务器启动失败: %v\n", err)
		}
	}()

	fmt.Printf("HTTP指标端点已启动，端口: %d\n", port)
	return nil
}

// StopHTTPEndpoint 停止HTTP端点
func (hme *HTTPMetricsExporter) StopHTTPEndpoint() error {
	hme.mutex.Lock()
	defer hme.mutex.Unlock()

	if hme.server != nil {
		return hme.server.Close()
	}

	return nil
}

// handlePrometheusMetrics 处理Prometheus指标请求
func (hme *HTTPMetricsExporter) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	// 这里应该从实际的指标收集器获取快照
	// 简化实现，返回示例数据
	snapshot := &MetricsSnapshot{
		Timestamp: time.Now(),
		Gauges: map[string]float64{
			"jvm.memory.used": 1024.0,
			"jvm.cpu.usage":   0.5,
		},
		Counters: map[string]int64{
			"processed.records": 1000,
			"error.count":       5,
		},
	}

	exporter := NewPrometheusExporter()
	content, err := exporter.ExportToPrometheus(snapshot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Write([]byte(content))
}

// handleJSONMetrics 处理JSON指标请求
func (hme *HTTPMetricsExporter) handleJSONMetrics(w http.ResponseWriter, r *http.Request) {
	// 这里应该从实际的指标收集器获取快照
	// 简化实现，返回示例数据
	snapshot := &MetricsSnapshot{
		Timestamp: time.Now(),
		Gauges: map[string]float64{
			"jvm.memory.used": 1024.0,
			"jvm.cpu.usage":   0.5,
		},
		Counters: map[string]int64{
			"processed.records": 1000,
			"error.count":       5,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

// handleHealth 处理健康检查请求
func (hme *HTTPMetricsExporter) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "UP",
		"timestamp": time.Now().Unix(),
		"service":   "nifi-metrics-exporter",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// StandardMetricsExporter 标准指标导出器
type StandardMetricsExporter struct {
	prometheusExporter *PrometheusExporter
	opentsdbExporter   *OpenTSDBExporter
	graphiteExporter   *GraphiteExporter
	jmxExporter        *JMXExporter
	csvExporter        *CSVExporter
	httpExporter       *HTTPMetricsExporter
	mutex              sync.RWMutex
}

// NewStandardMetricsExporter 创建新的标准指标导出器
func NewStandardMetricsExporter() *StandardMetricsExporter {
	return &StandardMetricsExporter{
		prometheusExporter: NewPrometheusExporter(),
		opentsdbExporter:   NewOpenTSDBExporter(),
		graphiteExporter:   NewGraphiteExporter(),
		jmxExporter:        NewJMXExporter(),
		csvExporter:        NewCSVExporter(),
		httpExporter:       NewHTTPMetricsExporter(),
	}
}

// ExportToPrometheus 导出到Prometheus格式
func (sme *StandardMetricsExporter) ExportToPrometheus(snapshot *MetricsSnapshot) (string, error) {
	return sme.prometheusExporter.ExportToPrometheus(snapshot)
}

// ExportToOpenTSDB 导出到OpenTSDB格式
func (sme *StandardMetricsExporter) ExportToOpenTSDB(snapshot *MetricsSnapshot) ([]byte, error) {
	return sme.opentsdbExporter.ExportToOpenTSDB(snapshot)
}

// ExportToGraphite 导出到Graphite格式
func (sme *StandardMetricsExporter) ExportToGraphite(snapshot *MetricsSnapshot) ([]byte, error) {
	return sme.graphiteExporter.ExportToGraphite(snapshot)
}

// ExportToJMX 导出到JMX格式
func (sme *StandardMetricsExporter) ExportToJMX(snapshot *MetricsSnapshot) ([]byte, error) {
	return sme.jmxExporter.ExportToJMX(snapshot)
}

// ExportToCSV 导出到CSV格式
func (sme *StandardMetricsExporter) ExportToCSV(snapshot *MetricsSnapshot) ([]byte, error) {
	return sme.csvExporter.ExportToCSV(snapshot)
}

// StartHTTPEndpoint 启动HTTP端点
func (sme *StandardMetricsExporter) StartHTTPEndpoint(port int) error {
	return sme.httpExporter.StartHTTPEndpoint(port)
}

// StopHTTPEndpoint 停止HTTP端点
func (sme *StandardMetricsExporter) StopHTTPEndpoint() error {
	return sme.httpExporter.StopHTTPEndpoint()
}
