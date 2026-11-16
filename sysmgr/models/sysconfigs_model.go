package models

import "database/sql"

// SysConfig DB 扫描结构（处理可空字段）
type SysConfig struct {
	ConfigId  int64          `db:"config_id"`
	Sort      string         `db:"sort"`
	Key       string         `db:"key"`
	Value     sql.NullString `db:"value"`
	Remark    sql.NullString `db:"remark"`
	Patrn     sql.NullString `db:"patrn"`
	Expansion sql.NullString `db:"expansion"`
}

// SysConfigItem JSON 输出结构
type SysConfigItem struct {
	ConfigId  int64  `json:"config_id"`
	Sort      string `json:"sort"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Remark    string `json:"remark"`
	Patrn     string `json:"patrn"`
	Expansion string `json:"expansion"`
}

// SysConfigAddForm 新增表单
type SysConfigAddForm struct {
	Sort      string `json:"sort"      validate:"required" label:"分组"   trim:"true"`
	Key       string `json:"key"       validate:"required" label:"参数名" trim:"true"`
	Value     string `json:"value"     validate:"required" label:"参数值"`
	Remark    string `json:"remark"`
	Patrn     string `json:"patrn"`
	Expansion string `json:"expansion"`
}

// SysConfigEditForm 修改表单（只允许修改 value）
type SysConfigEditForm struct {
	ConfigId int64  `json:"config_id" validate:"required,gt=0" label:"config_id"`
	Value    string `json:"value"     validate:"required"      label:"参数值"`
}

// SysConfigSearchParams 查询参数
type SysConfigSearchParams struct {
	Sort string `json:"sort"`
	PageParams
}
