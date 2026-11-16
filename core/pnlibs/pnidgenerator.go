package pnlibs

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	sf "github.com/bwmarrin/snowflake"
)

// 错误定义
var (
	ErrGeneratorStopped    = errors.New("id generator is stopped")
	ErrClockBackward       = errors.New("clock moved backwards")
	ErrMaxRetriesExceeded  = errors.New("max retries exceeded")
	ErrMaxWaitTimeExceeded = errors.New("max wait time exceeded")
	ErrInvalidNodeID       = errors.New("node ID must be between 0 and 1023")
	ErrBatchSizeTooLarge   = errors.New("batch size too large, max is 100000")
	ErrIDGenerationFailed  = errors.New("id generation failed")
)

// ClockBackwardDetail 时钟回拨错误详情
type ClockBackwardDetail struct {
	RetryCount    int
	WaitTime      time.Duration
	MaxRetries    int
	MaxWaitTime   time.Duration
	LastAttempt   time.Time
	OriginalError error
}

func (d *ClockBackwardDetail) Error() string {
	return fmt.Sprintf("clock backward error: attempted %d/%d retries, waited %v of max %v: %v",
		d.RetryCount, d.MaxRetries, d.WaitTime, d.MaxWaitTime, d.OriginalError)
}

func (d *ClockBackwardDetail) Unwrap() error {
	return d.OriginalError
}

// Metrics 监控指标
type Metrics struct {
	GeneratedCount     int64     `json:"generated_count"`
	ErrorCount         int64     `json:"error_count"`
	ClockBackwardCount int64     `json:"clock_backward_count"`
	LastGeneratedTime  time.Time `json:"last_generated_time"`
}

// PnIDGenerator 简化的ID生成器
type PnIDGenerator struct {
	node     *sf.Node
	config   Config
	metrics  Metrics
	stopChan chan struct{}
	stopped  int32
	mu       sync.RWMutex
}

// Config 生成器配置
type Config struct {
	NodeID        int64         `json:"node_id"`        // 节点ID，范围0-1023
	Epoch         int64         `json:"epoch"`          // 起始时间戳（毫秒）
	MaxRetries    int           `json:"max_retries"`    // 最大重试次数
	RetryDelay    time.Duration `json:"retry_delay"`    // 重试延迟时间
	MaxWaitTime   time.Duration `json:"max_wait_time"`  // 最大等待时间
	EnableMetrics bool          `json:"enable_metrics"` // 是否启用监控指标
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	// 设置默认起始时间为2020-01-01 00:00:00 UTC
	defaultEpoch := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	return Config{
		NodeID:        1,                // 默认节点ID为1
		Epoch:         defaultEpoch,     // 使用2020-01-01作为起始时间
		MaxRetries:    50,               // 最大重试50次
		RetryDelay:    time.Millisecond, // 重试延迟1毫秒
		MaxWaitTime:   30 * time.Second, // 最大等待30秒
		EnableMetrics: true,             // 默认启用监控指标
	}
}

// NewPnIDGenerator 创建新的ID生成器
func NewPnIDGenerator(config Config) (*PnIDGenerator, error) {
	// 验证节点ID是否在有效范围内
	if config.NodeID < 0 || config.NodeID > 1023 {
		return nil, fmt.Errorf("%w: node ID %d is invalid", ErrInvalidNodeID, config.NodeID)
	}

	// 设置自定义epoch
	if config.Epoch > 0 {
		sf.Epoch = config.Epoch
	}

	// 创建snowflake节点
	node, err := sf.NewNode(config.NodeID)
	if err != nil {
		return nil, fmt.Errorf("create snowflake node failed: %w", err)
	}

	// 创建并返回ID生成器实例
	generator := &PnIDGenerator{
		node:     node,
		config:   config,
		stopChan: make(chan struct{}),
	}

	return generator, nil
}

// Generate 生成一个唯一ID
func (g *PnIDGenerator) Generate() (int64, error) {
	// 检查生成器是否已停止
	if atomic.LoadInt32(&g.stopped) == 1 {
		return 0, ErrGeneratorStopped
	}

	// 使用重试机制生成ID
	return g.generateWithRetry(0, time.Now())
}

// GenerateBatch 批量生成多个ID
func (g *PnIDGenerator) GenerateBatch(count int) ([]int64, error) {
	// 验证数量参数
	if count <= 0 {
		return nil, errors.New("count must be positive")
	}
	if count > 100000 {
		return nil, ErrBatchSizeTooLarge
	}

	// 预分配切片空间
	ids := make([]int64, count)

	// 批量生成ID
	for i := 0; i < count; i++ {
		id, err := g.Generate()
		if err != nil {
			// 返回已成功生成的部分ID和错误信息
			return ids[:i], fmt.Errorf("generate batch failed at index %d: %w", i, err)
		}
		ids[i] = id
	}

	return ids, nil
}

// generateWithRetry 带重试的ID生成（内部方法）
func (g *PnIDGenerator) generateWithRetry(retryCount int, startTime time.Time) (int64, error) {
	// 检查是否超过最大重试次数
	if retryCount > g.config.MaxRetries {
		waitTime := time.Since(startTime)
		detail := &ClockBackwardDetail{
			RetryCount:    retryCount,
			WaitTime:      waitTime,
			MaxRetries:    g.config.MaxRetries,
			MaxWaitTime:   g.config.MaxWaitTime,
			LastAttempt:   time.Now(),
			OriginalError: fmt.Errorf("%w: %d", ErrMaxRetriesExceeded, g.config.MaxRetries),
		}
		return 0, fmt.Errorf("%w: %w", ErrClockBackward, detail)
	}

	// 检查是否超过最大等待时间
	if time.Since(startTime) > g.config.MaxWaitTime {
		detail := &ClockBackwardDetail{
			RetryCount:    retryCount,
			WaitTime:      g.config.MaxWaitTime,
			MaxRetries:    g.config.MaxRetries,
			MaxWaitTime:   g.config.MaxWaitTime,
			LastAttempt:   time.Now(),
			OriginalError: fmt.Errorf("%w: %v", ErrMaxWaitTimeExceeded, g.config.MaxWaitTime),
		}
		return 0, fmt.Errorf("%w: %w", ErrClockBackward, detail)
	}

	// 尝试生成ID
	id, err := g.tryGenerate()
	if err == nil {
		// 生成成功，更新监控指标
		if g.config.EnableMetrics {
			atomic.AddInt64(&g.metrics.GeneratedCount, 1)
			g.mu.Lock()
			g.metrics.LastGeneratedTime = time.Now()
			g.mu.Unlock()
		}
		return id, nil
	}

	// 处理时钟回拨错误
	if g.isClockBackwardError(err) {
		return g.handleClockBackward(retryCount, startTime, err)
	}

	// 其他错误直接返回
	return 0, fmt.Errorf("%w: %v", ErrIDGenerationFailed, err)
}

// tryGenerate 尝试生成ID（内部方法）
func (g *PnIDGenerator) tryGenerate() (int64, error) {
	// 调用底层snowflake库生成ID
	id := g.node.Generate()
	if id == 0 {
		return 0, fmt.Errorf("%w: generated zero ID", ErrIDGenerationFailed)
	}
	return int64(id), nil
}

// isClockBackwardError 判断是否为时钟回拨错误（内部方法）
func (g *PnIDGenerator) isClockBackwardError(err error) bool {
	errorMsg := err.Error()
	// 检查错误消息是否匹配时钟回拨相关的错误
	return errorMsg == "clock moved backwards" || errorMsg == "invalid system clock"
}

// handleClockBackward 处理时钟回拨（内部方法）
func (g *PnIDGenerator) handleClockBackward(retryCount int, startTime time.Time, originalErr error) (int64, error) {
	// 记录时钟回拨事件
	if g.config.EnableMetrics {
		atomic.AddInt64(&g.metrics.ClockBackwardCount, 1)
	}

	// 计算指数退避的等待时间
	waitTime := g.calculateWaitTime(retryCount)

	// 等待指定时间或直到收到停止信号
	select {
	case <-time.After(waitTime):
		// 继续重试
	case <-g.stopChan:
		return 0, ErrGeneratorStopped
	}

	// 递归调用进行重试
	return g.generateWithRetry(retryCount+1, startTime)
}

// calculateWaitTime 计算等待时间（内部方法）
func (g *PnIDGenerator) calculateWaitTime(retryCount int) time.Duration {
	// 使用指数退避策略，但不超过1秒
	waitTime := g.config.RetryDelay * time.Duration(1<<min(retryCount, 10))
	if waitTime > time.Second {
		waitTime = time.Second
	}
	return waitTime
}

// GetNodeID 获取节点ID
func (g *PnIDGenerator) GetNodeID() int64 {
	return g.config.NodeID
}

// GetStats 获取生成器统计信息
func (g *PnIDGenerator) GetStats() map[string]any {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return map[string]any{
		"node_id":              g.config.NodeID,
		"epoch":                g.config.Epoch,
		"epoch_time":           time.UnixMilli(g.config.Epoch).Format(time.RFC3339),
		"max_retries":          g.config.MaxRetries,
		"retry_delay":          g.config.RetryDelay.String(),
		"max_wait_time":        g.config.MaxWaitTime.String(),
		"stopped":              atomic.LoadInt32(&g.stopped) == 1,
		"generated_count":      g.metrics.GeneratedCount,
		"error_count":          g.metrics.ErrorCount,
		"clock_backward_count": g.metrics.ClockBackwardCount,
		"last_generated_time":  g.metrics.LastGeneratedTime.Format(time.RFC3339),
	}
}

// Stop 停止ID生成器
func (g *PnIDGenerator) Stop() {
	// 使用原子操作确保只停止一次
	if atomic.CompareAndSwapInt32(&g.stopped, 0, 1) {
		close(g.stopChan)
	}
}

// IsStopped 检查生成器是否已停止
func (g *PnIDGenerator) IsStopped() bool {
	return atomic.LoadInt32(&g.stopped) == 1
}

// HealthCheck 健康检查
func (g *PnIDGenerator) HealthCheck() error {
	// 检查生成器状态
	if g.IsStopped() {
		return ErrGeneratorStopped
	}

	// 尝试生成一个测试ID来验证功能
	_, err := g.tryGenerate()
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// IsClockBackwardError 检查错误是否为时钟回拨错误
func (g *PnIDGenerator) IsClockBackwardError(err error) bool {
	var clockErr *ClockBackwardDetail
	return errors.As(err, &clockErr)
}

// GetClockBackwardDetail 获取时钟回拨错误的详细信息
func (g *PnIDGenerator) GetClockBackwardDetail(err error) (*ClockBackwardDetail, bool) {
	var detail *ClockBackwardDetail
	if errors.As(err, &detail) {
		return detail, true
	}
	return nil, false
}

// 辅助函数：返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 使用示例
func Example() {
	// 创建生成器
	config := DefaultConfig()
	config.NodeID = 1

	gen, err := NewPnIDGenerator(config)
	if err != nil {
		panic(err)
	}
	defer gen.Stop()

	// 生成单个ID
	id, err := gen.Generate()
	if err != nil {
		fmt.Printf("Error generating ID: %v\n", err)
		return
	}

	fmt.Printf("Generated ID: %d\n", id)

	// 检查是否为时钟回拨错误
	if gen.IsClockBackwardError(err) {
		fmt.Println("This was a clock backward error")
		if detail, ok := gen.GetClockBackwardDetail(err); ok {
			fmt.Printf("Clock backward details: %+v\n", detail)
		}
	}

	// 批量生成ID
	ids, err := gen.GenerateBatch(5)
	if err != nil {
		fmt.Printf("Error generating batch: %v\n", err)
		return
	}

	fmt.Printf("Generated batch: %v\n", ids)

	// 查看统计信息
	stats := gen.GetStats()
	fmt.Printf("Generator stats: %+v\n", stats)
}
