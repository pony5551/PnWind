package utils

import (
	"fmt"
	"strings"
)

// GenerateUpsertSQL generates an INSERT ... ON CONFLICT ... DO UPDATE statement
// tableName: 表名
// fields: 表中所有字段按顺序排列, fields参数只用于排序，实际使用的字段来自data
// data: 包含要插入/更新的数据的 map，键是字段名，值是字段值
// primaryKeys: 主键字段列表
// excludeFromUpdate: 在 UPDATE 部分排除的字段列表（通常是主键或不应更新的字段）
// totalFields: 需要在 UPDATE 部分进行累加的字段列表（例如计数器字段）
func GenerateUpsertSQL(
	tableName string,
	fields []string,
	data map[string]interface{},
	primaryKeys []string,
	excludeFromUpdate []string,
	totalFields []string,
) string {
	if len(data) == 0 {
		return ""
	}

	// 创建有序的字段集合
	orderedFields := make([]string, 0)

	// 按照fields中的顺序遍历字段，但只使用data中存在的字段
	for _, field := range fields {
		if v, exists := data[field]; exists {
			if strValue, ok := v.(string); ok {
				// 类型断言成功，v 确实是 string 类型
				if strValue != "" {
					// 不是空字符串
					orderedFields = append(orderedFields, field)
				}
			} else {
				// v 不是 string 类型，可以根据需要处理
				orderedFields = append(orderedFields, field)
			}
		}
	}

	// 如果data中有fields中不存在的字段，我们不处理它们
	// 这确保了我们只处理fields中列出的顺序

	// Build the INSERT part
	var fieldNames []string
	var placeholders []string
	for _, field := range orderedFields {
		fieldNames = append(fieldNames, field)
		placeholders = append(placeholders, ":"+field)
	}

	// Build the UPDATE part
	var updateFields []string
	for _, field := range orderedFields {
		// Skip fields that are in the primary key or excluded from update
		shouldSkip := false
		for _, pk := range primaryKeys {
			if field == pk {
				shouldSkip = true
				break
			}
		}
		if !shouldSkip {
			for _, exclude := range excludeFromUpdate {
				if field == exclude {
					shouldSkip = true
					break
				}
			}
		}

		if !shouldSkip {
			totalFieldsStr := "," + strings.Join(totalFields, ",") + ","
			if strings.Contains(totalFieldsStr, ","+field+",") {
				updateFields = append(updateFields, fmt.Sprintf("%s = %s.%s+EXCLUDED.%s", field, tableName, field, field))
			} else {
				updateFields = append(updateFields, fmt.Sprintf("%s = EXCLUDED.%s", field, field))
			}
		}
	}

	// Construct the complete SQL statement
	sql := fmt.Sprintf(`
INSERT INTO %s (
	%s
) VALUES (
	%s
) ON CONFLICT (%s) DO UPDATE SET 
	%s;`,
		tableName,
		strings.Join(fieldNames, ",\n\t"),
		strings.Join(placeholders, ",\n\t"),
		strings.Join(primaryKeys, ", "),
		strings.Join(updateFields, ",\n\t"))

	return sql
}

// GenerateUpsertSQL generates an INSERT ... ON CONFLICT ... DO UPDATE statement
// fields参数只用于排序，实际使用的字段来自data
func GenerateUpsertSQLEx(
	tableName string,
	fields []string,
	data map[string]interface{},
	primaryKeys []string,
	excludeFromUpdate []string,
) string {
	if len(data) == 0 {
		return ""
	}

	// 创建有序的字段集合
	orderedFields := make([]string, 0)

	// 按照fields中的顺序遍历字段，但只使用data中存在的字段
	for _, field := range fields {
		if v, exists := data[field]; exists {
			if strValue, ok := v.(string); ok {
				// 类型断言成功，v 确实是 string 类型
				if strValue != "" {
					// 不是空字符串
					orderedFields = append(orderedFields, field)
				}
			} else {
				// v 不是 string 类型，可以根据需要处理
				orderedFields = append(orderedFields, field)
			}
		}
	}

	// 如果data中有fields中不存在的字段，我们不处理它们
	// 这确保了我们只处理fields中列出的顺序

	// Build the INSERT part
	var fieldNames []string
	var placeholders []string
	for _, field := range orderedFields {
		fieldNames = append(fieldNames, field)
		placeholders = append(placeholders, ":"+field)
	}

	// Build the UPDATE part
	var updateFields []string
	for _, field := range orderedFields {
		// Skip fields that are in the primary key or excluded from update
		shouldSkip := false
		for _, pk := range primaryKeys {
			if field == pk {
				shouldSkip = true
				break
			}
		}
		if !shouldSkip {
			for _, exclude := range excludeFromUpdate {
				if field == exclude {
					shouldSkip = true
					break
				}
			}
		}

		if !shouldSkip {
			updateFields = append(updateFields, fmt.Sprintf("%s = EXCLUDED.%s", field, field))
		}
	}

	// Construct the complete SQL statement
	sql := fmt.Sprintf(`
INSERT INTO %s (
	%s
) VALUES (
	%s
) ON CONFLICT (%s) DO UPDATE SET 
	%s;`,
		tableName,
		strings.Join(fieldNames, ",\n\t"),
		strings.Join(placeholders, ",\n\t"),
		strings.Join(primaryKeys, ", "),
		strings.Join(updateFields, ",\n\t"))

	return sql
}
