package models

import (
	"time"

	pg "github.com/lib/pq"
)

// SysRole 角色（对应 db_sys_role 表）。接口里的字段名和列名一样
type SysRole struct {
	Id        int64          `db:"id"         json:"id"`
	ParentId  int64          `db:"parent_id"  json:"parent_id"` // 上级角色的 id，0 表示顶级
	Name      string         `db:"name"       json:"name"`
	Code      string         `db:"code"       json:"code"`
	Sort      int            `db:"sort"       json:"sort"`
	Status    int16          `db:"status"     json:"status"`
	Remark    string         `db:"remark"     json:"remark"`
	MenuSet   pg.StringArray `db:"menuset"    json:"menuset"` // 权限键：菜单表里页面行、按钮行的 name
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
}

// SysRoleForm 角色新增/修改表单。修改时 id 必填，由控制器检查；创建时间由表上的默认值填
type SysRoleForm struct {
	Id       int64  `db:"id"        json:"id"`
	ParentId int64  `db:"parent_id" json:"parent_id"`
	Code     string `db:"code"      json:"code"      validate:"required" label:"角色编码" trim:"true"`
	Name     string `db:"name"      json:"name"      validate:"required" label:"角色名称" trim:"true"`
	Sort     int    `db:"sort"      json:"sort"      validate:"gte=0"    label:"排序"`
	Status   int16  `db:"status"    json:"status"    validate:"gte=0"    label:"状态"`
	Remark   string `db:"remark"    json:"remark"`
}

// SysRoleSearchParams 角色列表搜索参数
type SysRoleSearchParams struct {
	Name   string `json:"name"        default:""`
	Code   string `json:"code"        default:""`
	Status string `json:"status"      default:""`
	Total  uint64 `json:"total"       default:"0"` // 修复：原为 son:"total"（tag 拼写错误）
}

// SysRole_MenuSetSaveForm 角色菜单权限保存表单。
// menuset 必须带：空数组表示清空全部权限；不带（或者是 null）算参数错误
type SysRole_MenuSetSaveForm struct {
	Id      int64          `db:"id"      json:"id"      validate:"required"               label:"角色ID"`
	MenuSet pg.StringArray `db:"menuset" json:"menuset" validate:"required,dive,required" label:"角色菜单权限"`
}

// SysRoleSimple 角色简要信息（管理员管理页、用户管理页的角色下拉用）
type SysRoleSimple struct {
	Id   int64  `db:"id"   json:"id"`
	Name string `db:"name" json:"name"`
	Code string `db:"code" json:"code"`
}
