// Package pnpartmgr 分区表管理（只适用于 PostgreSQL）：给按月分区的表提前建好后面几个月的分区，
// 查询分区的大小、行数、最近一次清理和分析的时间。
//
// 主表要先建好（PARTITION BY RANGE，分区键是日期列或时间列；演示表 db_demo_partition 是单独一列日期 created_date），
// 分区的名字是 <表名>_<年月>，例如 db_demo_partition_202611。
// 怎么装配见 main.go：启动时 CheckPartitionsAll 检查一次，再往任务管理器登记一个定时任务每天检查。
// 表名是直接拼进 SQL 的，只能传代码里写死的名字，不要把请求里的值当表名传进来。
package pnpartmgr

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/pony5551/PnWind/core/pnlogrus"

	"github.com/jmoiron/sqlx"
)

// PnPartitionManager 分区管理器
type PnPartitionManager struct {
	db *sqlx.DB
}

// NewPnPartitionManager 创建新的分区管理器
func NewPnPartitionManager(db *sqlx.DB) *PnPartitionManager {
	// 在结构初始化时创建系统分区日志表
	createLogSQL := `
		CREATE TABLE IF NOT EXISTS db_sys_partition_creation_log (
			id BIGSERIAL PRIMARY KEY,
			partition_name VARCHAR(100) NOT NULL,
			table_name VARCHAR(100) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := db.Exec(createLogSQL)
	if err != nil {
		log.Warnf("创建系统分区日志表失败: %v", err)
	} else {
		log.Printf("系统分区日志表初始化完成: db_sys_partition_creation_log")
	}

	return &PnPartitionManager{
		db: db,
	}
}

// PartitionInfo 分区信息结构体（增加统计字段）
type PartitionInfo struct {
	ParentSchema    string     `json:"parent_schema"`
	ParentName      string     `json:"parent_name"`
	ChildSchema     string     `json:"child_schema"`
	ChildName       string     `json:"child_name"`
	PartitionBound  string     `json:"partition_bound"`
	RelationType    string     `json:"relation_type"`
	Level           int        `json:"level"`
	TotalSize       int64      `json:"total_size"`       // 总大小（字节）
	TableSize       int64      `json:"table_size"`       // 表数据大小（字节）
	IndexSize       int64      `json:"index_size"`       // 索引大小（字节）
	LiveTuples      int64      `json:"live_tuples"`      // 活元组数
	DeadTuples      int64      `json:"dead_tuples"`      // 死元组数
	LastAnalyze     *time.Time `json:"last_analyze"`     // 最后分析时间
	LastAutoanalyze *time.Time `json:"last_autoanalyze"` // 最后自动分析时间
	LastVacuum      *time.Time `json:"last_vacuum"`      // 最后清理时间
	LastAutovacuum  *time.Time `json:"last_autovacuum"`  // 最后自动清理时间
}

// PartitionFilter 分区查询过滤器（增加统计过滤条件）
type PartitionFilter struct {
	TableName    string // 表名，支持 "schema.table" 格式
	SchemaName   string // 模式名：只看主表在这个模式里的（指定了 TableName 时不看它）
	ParentOnly   bool   // 只返回第一层的分区（主表的直接下级）
	LeafOnly     bool   // 只返回叶子分区
	MaxLevel     int    // 最大层级限制
	MinSizeBytes int64  // 最小表大小（字节）
	MaxSizeBytes int64  // 最大表大小（字节）
	MinRows      int64  // 最小行数
	MaxRows      int64  // 最大行数
}

// tableExists 表在不在。按连接的 search_path 找，名字也可以写成 "schema.table"
func (pm *PnPartitionManager) tableExists(name string) (bool, error) {
	var exists bool
	err := pm.db.Get(&exists, `SELECT EXISTS (SELECT 1 FROM pg_class WHERE oid = to_regclass($1) AND relkind IN ('r', 'p'))`, name)
	return exists, err
}

// checkPartitionExists 检查分区是否存在
func (pm *PnPartitionManager) checkPartitionExists(tableName string, partitionDate time.Time) (bool, error) {
	partitionName := fmt.Sprintf("%s_%s", tableName, partitionDate.Format("200601"))

	exists, err := pm.tableExists(partitionName)
	if err != nil {
		return false, fmt.Errorf("检查分区存在性失败: %v", err)
	}

	return exists, nil
}

// createPartition 创建分区表：已经有了就跳过；建好之后往系统表里记一条分区日志
func (pm *PnPartitionManager) createPartition(tableName string, partitionDate time.Time) error {
	partitionName := fmt.Sprintf("%s_%s", tableName, partitionDate.Format("200601"))
	// 分区的范围写的是不带时区的日期（'2026-11-01' 到 '2026-12-01'）。
	// 分区键是日期列（date，演示表的 created_date 就是）时，范围就是这两个日期本身，和时区没有关系。
	// 分区键是带时区的时间（timestamptz）时，数据库按会话的时区解释它：哪一刻换月跟着数据库的时区走，不跟程序所在机器的时区。
	// 两种情况下，连同一个库的几个实例不管各自在什么系统、什么时区，建出来的分区都一样
	startDate := time.Date(partitionDate.Year(), partitionDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	// 检查分区是否已存在
	exists, err := pm.checkPartitionExists(tableName, partitionDate)
	if err != nil {
		return err
	}

	if exists {
		log.Printf("分区已存在: %s", partitionName)
		return nil
	}

	// 创建分区SQL
	createSQL := fmt.Sprintf(`
		CREATE TABLE %s PARTITION OF %s
		FOR VALUES FROM ('%s') TO ('%s')
	`, partitionName, tableName, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	if err = pm.execCreate(createSQL); err != nil {
		return fmt.Errorf("创建分区失败: %s - %v", partitionName, err)
	}

	// 记录分区创建日志到系统表。分区这时已经建好了，日志没记上只是少一条记录
	logSQL := `
		INSERT INTO db_sys_partition_creation_log
		(partition_name, table_name, created_at)
		VALUES ($1, $2, $3)
	`
	_, err = pm.db.Exec(logSQL, partitionName, tableName, time.Now())
	if err != nil {
		log.Warnf("记录分区日志失败: %v", err)
	}

	log.Printf("分区创建成功: %s", partitionName)
	return nil
}

// execCreate 执行建分区的语句。单独一个事务：这个月份建不出来，不连累别的月份
func (pm *PnPartitionManager) execCreate(createSQL string) error {
	tx, err := pm.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback() // 已经提交的话，它什么都不做

	// 建分区要拿主表上最强的锁。主表正被一个没结束的事务占着（例如一个跑了很久的查询）时，这条语句排队等，
	// 它排队期间主表上新来的读写也都排在它后面。所以最多等 5 秒，等不到就放弃这一次：
	// 分区是提前几个月建的，下一轮再建来得及
	if _, err = tx.Exec(`SET LOCAL lock_timeout = '5s'`); err != nil {
		return err
	}
	if _, err = tx.Exec(createSQL); err != nil {
		return err
	}
	return tx.Commit()
}

// CheckAndCreatePartitions 检查并创建分区：过去 checkPastMonths 个月到未来 monthsAhead 个月，一个月一个分区。
// 每个月份各自建：有月份建不出来，记下来接着建后面的。这时结果和错误一起返回，错误里是每个失败月份的原因
func (pm *PnPartitionManager) CheckAndCreatePartitions(tableName string, monthsAhead int, checkPastMonths int) (*PartitionOperationResult, error) {
	currentTime := time.Now()
	currentMonth := time.Date(currentTime.Year(), currentTime.Month(), 1, 0, 0, 0, 0, time.UTC)

	// 参数验证
	if monthsAhead < 1 || monthsAhead > 12 {
		return nil, fmt.Errorf("monthsAhead 参数必须在 1-12 之间")
	}

	if checkPastMonths < 0 || checkPastMonths > 6 {
		return nil, fmt.Errorf("checkPastMonths 参数必须在 0-6 之间")
	}

	result := &PartitionOperationResult{
		TableName:       tableName,
		MonthsAhead:     monthsAhead,
		CheckPastMonths: checkPastMonths,
		StartTime:       time.Now(),
	}

	log.Printf("开始检查分区，表名: %s, 未来月数: %d, 检查过去月数: %d",
		tableName, monthsAhead, checkPastMonths)

	// 检查过去和未来的月份
	startMonth := -checkPastMonths
	endMonth := monthsAhead

	var errs []error
	for i := startMonth; i <= endMonth; i++ {
		targetMonth := currentMonth.AddDate(0, i, 0)
		result.TotalOperations++

		err := pm.createPartition(tableName, targetMonth)
		if err != nil {
			result.FailedOperations++
			log.Errorf("处理月份 %s 失败: %v", targetMonth.Format("2006-01"), err)
			errs = append(errs, err)
		} else {
			result.SuccessfulOperations++
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	log.Printf("分区检查完成: 成功 %d/%d, 耗时: %v",
		result.SuccessfulOperations, result.TotalOperations, result.Duration)

	return result, errors.Join(errs...)
}

// PartitionOperationResult 分区操作结果
type PartitionOperationResult struct {
	TableName            string
	MonthsAhead          int
	CheckPastMonths      int
	TotalOperations      int
	SuccessfulOperations int
	FailedOperations     int
	StartTime            time.Time
	EndTime              time.Time
	Duration             time.Duration
}

// EnsureBaseTableExists 确保基础表存在
func (pm *PnPartitionManager) EnsureBaseTableExists(tableName string) error {
	// 检查基础表是否存在
	exists, err := pm.tableExists(tableName)
	if err != nil {
		return fmt.Errorf("检查基础表存在性失败: %v", err)
	}

	if !exists {
		return fmt.Errorf("基础表不存在: %s，请先创建基础表", tableName)
	}

	return nil
}

// parseTableName 解析表名字符串，支持 "schema.table" 格式
func (pm *PnPartitionManager) parseTableName(tableName string) (schema, table string) {
	parts := strings.Split(tableName, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", tableName // 如果没有点，认为只有表名
}

// GetPartitions 使用过滤器查询分区信息，包含表数据统计（兼容版本）
func (pm *PnPartitionManager) GetPartitions(filter *PartitionFilter) ([]PartitionInfo, error) {
	log.Printf("开始查询分区信息，filter: %+v", filter)

	var whereConditions []string
	var args []any
	var argIndex = 1

	baseQuery := `
    WITH RECURSIVE partition_info AS (
        SELECT
            p.oid AS parent_oid,
            c.oid AS child_oid,
            p.relname AS parent_name,
            c.relname AS child_name,
            p_ns.nspname AS parent_schema,
            c_ns.nspname AS child_schema,
            COALESCE(pg_get_expr(c.relpartbound, c.oid), '') AS partition_bound,
            CASE
                WHEN c.relkind = 'p' THEN '分区表'
                ELSE '子分区'
            END AS relation_type,
            0 AS level,
            -- 表统计信息（使用兼容的函数）
            pg_total_relation_size(c.oid) AS total_size,
            pg_relation_size(c.oid) AS table_size,
            pg_indexes_size(c.oid) AS index_size,
            -- 行数统计：直接调统计函数（pg_stat_all_tables 这个视图就是用它们定义的，逐行去查视图要慢几倍）
            pg_stat_get_live_tuples(c.oid) AS live_tuples,
            pg_stat_get_dead_tuples(c.oid) AS dead_tuples,
            -- 维护时间：同上
            pg_stat_get_last_analyze_time(c.oid) AS last_analyze,
            pg_stat_get_last_autoanalyze_time(c.oid) AS last_autoanalyze,
            pg_stat_get_last_vacuum_time(c.oid) AS last_vacuum,
            pg_stat_get_last_autovacuum_time(c.oid) AS last_autovacuum
        FROM pg_inherits i
        JOIN pg_class p ON i.inhparent = p.oid
        JOIN pg_class c ON i.inhrelid = c.oid
        JOIN pg_namespace p_ns ON p.relnamespace = p_ns.oid
        JOIN pg_namespace c_ns ON c.relnamespace = c_ns.oid
        WHERE p.relkind IN ('p', 'r')
    `

	// 构建过滤条件
	if filter != nil {
		if filter.TableName != "" {
			schema, table := pm.parseTableName(filter.TableName)
			if schema != "" {
				whereConditions = append(whereConditions, fmt.Sprintf("p_ns.nspname = $%d", argIndex))
				args = append(args, schema)
				argIndex++
			}
			if table != "" {
				whereConditions = append(whereConditions, fmt.Sprintf("p.relname = $%d", argIndex))
				args = append(args, table)
				argIndex++
			}
		} else if filter.SchemaName != "" {
			whereConditions = append(whereConditions, fmt.Sprintf("p_ns.nspname = $%d", argIndex))
			args = append(args, filter.SchemaName)
			argIndex++
		}
	}

	// 没有指定表名时，只从最顶层的表开始往下找。
	// 多级分区里的子分区表既是别人的下级、又有自己的下级，把它也当起点的话，它的下级会出现两次
	if filter == nil || filter.TableName == "" {
		whereConditions = append(whereConditions, "NOT EXISTS (SELECT 1 FROM pg_inherits up WHERE up.inhrelid = p.oid)")
	}

	// 添加WHERE条件
	if len(whereConditions) > 0 {
		baseQuery += " AND " + strings.Join(whereConditions, " AND ")
	}

	// 递归部分
	recursiveWhere := "pi.level < 10"
	if filter != nil && filter.MaxLevel > 0 {
		recursiveWhere = fmt.Sprintf("pi.level < %d", filter.MaxLevel)
	}

	recursivePart := fmt.Sprintf(`
        UNION ALL

        SELECT
            pi.child_oid AS parent_oid,
            c.oid AS child_oid,
            pi.child_name AS parent_name,
            c.relname AS child_name,
            pi.child_schema AS parent_schema,
            c_ns.nspname AS child_schema,
            COALESCE(pg_get_expr(c.relpartbound, c.oid), '') AS partition_bound,
            CASE
                WHEN c.relkind = 'p' THEN '子分区表'
                ELSE '叶子分区'
            END AS relation_type,
            pi.level + 1 AS level,
            -- 表统计信息（递归部分使用相同方法）
            pg_total_relation_size(c.oid) AS total_size,
            pg_relation_size(c.oid) AS table_size,
            pg_indexes_size(c.oid) AS index_size,
            pg_stat_get_live_tuples(c.oid) AS live_tuples,
            pg_stat_get_dead_tuples(c.oid) AS dead_tuples,
            pg_stat_get_last_analyze_time(c.oid) AS last_analyze,
            pg_stat_get_last_autoanalyze_time(c.oid) AS last_autoanalyze,
            pg_stat_get_last_vacuum_time(c.oid) AS last_vacuum,
            pg_stat_get_last_autovacuum_time(c.oid) AS last_autovacuum
        FROM partition_info pi
        JOIN pg_inherits i ON i.inhparent = pi.child_oid
        JOIN pg_class c ON i.inhrelid = c.oid
        JOIN pg_namespace c_ns ON c.relnamespace = c_ns.oid
        WHERE %s
    )`, recursiveWhere)

	// 最终SELECT部分
	selectPart := `
    SELECT
        parent_schema,
        parent_name,
        child_schema,
        child_name,
        partition_bound,
        relation_type,
        level,
        total_size,
        table_size,
        index_size,
        COALESCE(live_tuples, 0) as live_tuples,
        COALESCE(dead_tuples, 0) as dead_tuples,
        last_analyze,
        last_autoanalyze,
        last_vacuum,
        last_autovacuum
    FROM partition_info
    `

	// 最终过滤条件
	var finalConditions []string
	if filter != nil {
		if filter.ParentOnly {
			finalConditions = append(finalConditions, "level = 0")
		}
		if filter.LeafOnly {
			finalConditions = append(finalConditions, `
                relation_type != '分区表' AND relation_type != '子分区表'
            `)
		}
		if filter.MinSizeBytes > 0 {
			finalConditions = append(finalConditions, fmt.Sprintf("total_size >= %d", filter.MinSizeBytes))
		}
		if filter.MaxSizeBytes > 0 {
			finalConditions = append(finalConditions, fmt.Sprintf("total_size <= %d", filter.MaxSizeBytes))
		}
		if filter.MinRows > 0 {
			finalConditions = append(finalConditions, fmt.Sprintf("COALESCE(live_tuples, 0) >= %d", filter.MinRows))
		}
		if filter.MaxRows > 0 {
			finalConditions = append(finalConditions, fmt.Sprintf("COALESCE(live_tuples, 0) <= %d", filter.MaxRows))
		}
	}

	if len(finalConditions) > 0 {
		selectPart += " WHERE " + strings.Join(finalConditions, " AND ")
	}

	selectPart += " ORDER BY parent_schema, parent_name, level, child_name;"

	query := baseQuery + recursivePart + selectPart

	//log.Printf("执行的SQL查询:\n%s", query)
	//log.Printf("查询参数: %v", args)

	// 检查数据库连接
	if pm.db == nil {
		return nil, fmt.Errorf("数据库连接为空")
	}

	rows, err := pm.db.Query(query, args...)
	if err != nil {
		log.Errorf("SQL查询失败: %v", err)
		return nil, fmt.Errorf("查询分区信息失败: %w", err)
	}
	defer rows.Close()

	var partitions []PartitionInfo
	rowCount := 0

	for rows.Next() {
		rowCount++
		var pi PartitionInfo
		var (
			lastAnalyze, lastAutoanalyze sql.NullTime
			lastVacuum, lastAutovacuum   sql.NullTime
			liveTuples, deadTuples       sql.NullInt64
		)

		err := rows.Scan(
			&pi.ParentSchema,
			&pi.ParentName,
			&pi.ChildSchema,
			&pi.ChildName,
			&pi.PartitionBound,
			&pi.RelationType,
			&pi.Level,
			&pi.TotalSize,
			&pi.TableSize,
			&pi.IndexSize,
			&liveTuples,
			&deadTuples,
			&lastAnalyze,
			&lastAutoanalyze,
			&lastVacuum,
			&lastAutovacuum,
		)
		if err != nil {
			log.Errorf("扫描行数据失败: %v", err)
			return nil, fmt.Errorf("扫描分区信息失败: %w", err)
		}

		// 处理可能为NULL的字段
		if liveTuples.Valid {
			pi.LiveTuples = liveTuples.Int64
		}
		if deadTuples.Valid {
			pi.DeadTuples = deadTuples.Int64
		}
		if lastAnalyze.Valid {
			pi.LastAnalyze = &lastAnalyze.Time
		}
		if lastAutoanalyze.Valid {
			pi.LastAutoanalyze = &lastAutoanalyze.Time
		}
		if lastVacuum.Valid {
			pi.LastVacuum = &lastVacuum.Time
		}
		if lastAutovacuum.Valid {
			pi.LastAutovacuum = &lastAutovacuum.Time
		}

		partitions = append(partitions, pi)
	}

	if err := rows.Err(); err != nil {
		log.Errorf("遍历行时出错: %v", err)
		return nil, fmt.Errorf("遍历分区信息失败: %w", err)
	}

	log.Printf("成功查询到 %d 个分区", len(partitions))
	return partitions, nil
}

// FormatSize 格式化字节大小为可读格式
func (pm *PnPartitionManager) FormatSize(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(bytes)
	unitIndex := 0

	for size >= 1024 && unitIndex < len(units)-1 {
		size /= 1024
		unitIndex++
	}

	return fmt.Sprintf("%.2f %s", size, units[unitIndex])
}

// FormatTime 格式化时间显示
func (pm *PnPartitionManager) FormatTime(t *time.Time) string {
	if t == nil {
		return "从未"
	}
	return t.Format("2006-01-02 15:04:05")
}

// PrintPartitionTreeWithStats 带统计信息的分区树打印
func (pm *PnPartitionManager) PrintPartitionTreeWithStats(partitions []PartitionInfo) {
	currentParent := ""

	for _, part := range partitions {
		parentKey := part.ParentSchema + "." + part.ParentName
		if parentKey != currentParent {
			fmt.Printf("\n📁 主表: %s.%s\n", part.ParentSchema, part.ParentName)
			currentParent = parentKey
		}

		indent := ""
		for i := 0; i < part.Level; i++ {
			indent += "  "
		}

		icon := "📄"
		if part.RelationType == "分区表" || part.RelationType == "子分区表" {
			icon = "📂"
		}

		fmt.Printf("%s%s %s.%s (%s, 层级: %d)\n",
			indent, icon, part.ChildSchema, part.ChildName,
			part.RelationType, part.Level)

		// 显示统计信息
		if part.LiveTuples > 0 || part.TotalSize > 0 {
			fmt.Printf("%s  数据: %s 行, 总大小: %s (表: %s, 索引: %s)\n",
				indent,
				pm.formatNumber(part.LiveTuples),
				pm.FormatSize(part.TotalSize),
				pm.FormatSize(part.TableSize),
				pm.FormatSize(part.IndexSize))
		}

		if part.DeadTuples > 0 {
			fmt.Printf("%s  死元组: %s, 需要清理\n",
				indent, pm.formatNumber(part.DeadTuples))
		}

		if part.PartitionBound != "" {
			fmt.Printf("%s  约束: %s\n", indent, part.PartitionBound)
		}

		// 显示维护信息
		if part.LastAnalyze != nil || part.LastVacuum != nil {
			fmt.Printf("%s  维护: 分析(%s) 清理(%s)\n",
				indent,
				pm.FormatTime(part.LastAnalyze),
				pm.FormatTime(part.LastVacuum))
		}
	}
}

// formatNumber 格式化数字显示
func (pm *PnPartitionManager) formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	} else if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	} else if n < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	} else {
		return fmt.Sprintf("%.1fB", float64(n)/1000000000)
	}
}

// GetPartitionStats 获取分区统计信息。表名按连接的 search_path 找，也可以写成 "schema.table"
func (pm *PnPartitionManager) GetPartitionStats(tableName string) (map[string]any, error) {
	stats := make(map[string]any)

	// 获取分区数量
	var partitionCount int
	err := pm.db.Get(&partitionCount, `
		SELECT COUNT(*) FROM pg_inherits i
		WHERE i.inhparent = to_regclass($1)
	`, tableName)
	if err != nil {
		return nil, err
	}
	stats["partition_count"] = partitionCount

	// 获取最近创建的分区
	var latestPartition string
	err = pm.db.Get(&latestPartition, `
		SELECT c.relname
		FROM pg_inherits i
		JOIN pg_class c ON i.inhrelid = c.oid
		WHERE i.inhparent = to_regclass($1)
		ORDER BY c.relname DESC
		LIMIT 1
	`, tableName)
	if err == nil {
		stats["latest_partition"] = latestPartition
	}

	// 获取总数据量估算
	var totalSize string
	err = pm.db.Get(&totalSize, `
		SELECT pg_size_pretty(SUM(pg_total_relation_size(c.oid)))
		FROM pg_inherits i
		JOIN pg_class c ON i.inhrelid = c.oid
		WHERE i.inhparent = to_regclass($1)
	`, tableName)
	if err == nil {
		stats["total_size"] = totalSize
	}

	return stats, nil
}

// CheckPartitions 检查一张表：基础表在不在，补建分区，再把分区的统计信息记进日志
func (pm *PnPartitionManager) CheckPartitions(tableName string, monthsAhead int, checkPastMonths int) error {
	log.Printf("执行分区检查: %s, %s", tableName, time.Now().Format("2006-01-02 15:04:05"))

	// 确保基础表存在
	err := pm.EnsureBaseTableExists(tableName)
	if err != nil {
		log.Errorf("确保基础表存在失败: %v", err)
		return err
	}

	// 执行分区检查
	result, err := pm.CheckAndCreatePartitions(tableName, monthsAhead, checkPastMonths)
	if err != nil {
		log.Errorf("分区检查失败: %v", err)
		return err
	} else {
		log.Printf("分区检查完成: %+v", result)
	}

	// 获取并记录统计信息
	stats, err := pm.GetPartitionStats(tableName)
	if err != nil {
		log.Errorf("分区统计失败: %v", err)
		return err
	}

	log.Printf("分区统计: %+v", stats)
	return nil
}

// CheckPartitionsAll 逐张表检查分区。一张表出错不影响后面的表（原因 CheckPartitions 已经记进日志）；
// 有表出错时返回错误，里面是每张出错的表和原因
func (pm *PnPartitionManager) CheckPartitionsAll(tableNames []string, monthsAhead int, checkPastMonths int) error {
	var errs []error
	for _, tableName := range tableNames {
		err := pm.CheckPartitions(tableName, monthsAhead, checkPastMonths)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", tableName, err))
		}
	}
	return errors.Join(errs...)
}

// RunScheduledTask 运行定时任务：每隔 interval 检查一次，quit 关闭时返回。
// 它是一个循环，要交给任务管理器当常驻协程运行，进程退出时才会等它做完手上这一次：
//
//	tasks := core.GConfig.Tasks
//	tasks.Go("分区检查", func(ctx context.Context) error {
//		pm.RunScheduledTask(tasks.Quit(), 24*time.Hour, tableNames, 3, 1)
//		return nil
//	})
//
// 想按钟点跑（例如每天 3 点），用任务管理器的定时任务，写法见 main.go
func (pm *PnPartitionManager) RunScheduledTask(quit <-chan struct{}, interval time.Duration, tableNames []string, monthsAhead int, checkPastMonths int) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("启动定时分区检查任务，间隔: %v, 监控的表: %s", interval, strings.Join(tableNames, ","))

	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			// 出错的原因 CheckPartitions 已经记进日志，这里不用再记
			_ = pm.CheckPartitionsAll(tableNames, monthsAhead, checkPastMonths)
		}
	}
}
