package models

import "time"

// SysLogOpt 操作日志（对应 db_sys_logopt 表）。接口里的字段名和列名一样
type SysLogOpt struct {
	Id        uint64    `db:"id"         json:"id"`
	DeviceId  uint64    `db:"device_id"  json:"device_id"`  // 节点号
	UserType  int16     `db:"user_type"  json:"user_type"`  // 1 管理员，2 用户
	UserId    string    `db:"user_id"    json:"user_id"`    // 管理员表或用户表的 user_id
	Type      string    `db:"type"       json:"type"`       // 类别，例如 用户操作
	Events    string    `db:"events"     json:"events"`     // 页面事件，也就是 act
	Details   string    `db:"details"    json:"details"`    // 页面事件详情，例如 部门管理-添加
	PagePath  string    `db:"page_path"  json:"page_path"`  // 页面的接口地址
	PageQuery string    `db:"page_query" json:"page_query"` // 地址参数
	PageForm  string    `db:"page_form"  json:"page_form"`  // 请求体
	Ip        string    `db:"ip"         json:"ip"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// SysLogOptSearchParams 操作日志列表搜索参数。
// 起止时间要带时区（例如 2026-11-15T00:00:00+08:00）：不带时区的话，数据库按它自己的时区去理解，和页面上选的差出几个小时
type SysLogOptSearchParams struct {
	CurrentPage uint64 `json:"currentPage" default:"1"`
	PageSize    uint64 `json:"pageSize"    default:"100"`
	UserType    int16  `json:"user_type"   validate:"oneof=0 1 2" label:"用户类型"` // 0 = 不过滤
	UserId      string `json:"user_id"`
	Type        string `json:"type"`
	SKey        string `json:"skey"`
	StartTime   string `json:"starttime"   validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00" msg:"开始时间要带时区，例如 2026-11-15T00:00:00+08:00"`
	EndTime     string `json:"endtime"     validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00" msg:"结束时间要带时区，例如 2026-11-15T23:59:59+08:00"`
}
