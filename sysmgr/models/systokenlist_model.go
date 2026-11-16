package models

import (
	"time"

	pg "github.com/lib/pq"
)

// SysTokenList 系统Token完整结构（认证中间件使用，对应 db_sys_tokenlist 全部字段）
// user_type: 1=管理员, 2=用户
type SysTokenList struct {
	Id          uint64         `db:"id"`
	TokenId     string         `db:"token_id"`
	UserType    int16          `db:"user_type"`
	UserId      string         `db:"user_id"`
	Roles       pg.StringArray `db:"roles"`
	UserMenuSet pg.StringArray `db:"menuset"`
	Ip          string         `db:"ip"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
}

// GetUserId 实现 pncontext.TokenInfo 接口
func (t *SysTokenList) GetUserId() string { return t.UserId }

// SysTokenListRow 管理列表查询专用结构（不 SELECT 大字段 menuset）。接口里的字段名和列名一样
// user_type: 1=管理员, 2=用户
type SysTokenListRow struct {
	Id        uint64    `db:"id"         json:"id"`
	TokenId   string    `db:"token_id"   json:"token_id"`
	UserType  int16     `db:"user_type"  json:"user_type"`
	UserId    string    `db:"user_id"    json:"user_id"`
	Ip        string    `db:"ip"         json:"ip"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// SysTokenListSearchParams Token列表搜索参数
// UserType: 0=不过滤, 1=管理员, 2=用户
// 起止时间（按登录时间筛）要带时区，例如 2026-11-15T00:00:00+08:00：不带时区的话，数据库按它自己的时区去理解，和页面上选的差出几个小时
type SysTokenListSearchParams struct {
	CurrentPage uint64 `json:"currentPage" default:"1"`
	PageSize    uint64 `json:"pageSize"    default:"20"`
	UserId      string `json:"user_id"`
	UserType    int16  `json:"user_type"`
	Ip          string `json:"ip"`
	StartTime   string `json:"starttime"   validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00" msg:"开始时间要带时区，例如 2026-11-15T00:00:00+08:00"`
	EndTime     string `json:"endtime"     validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00" msg:"结束时间要带时区，例如 2026-11-15T23:59:59+08:00"`
}
