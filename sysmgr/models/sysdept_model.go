package models

import "time"

// SysDept 系统部门（对应 db_sys_dept 表）。接口里的字段名和列名一样
type SysDept struct {
	Id        int64     `db:"id"         json:"id"`
	ParentId  int64     `db:"parent_id"  json:"parent_id"`
	Name      string    `db:"name"       json:"name"`
	Principal string    `db:"principal"  json:"principal"`
	Phone     string    `db:"phone"      json:"phone"`
	Email     string    `db:"email"      json:"email"`
	Sort      int       `db:"sort"       json:"sort"`
	Status    int16     `db:"status"     json:"status"`
	Remark    string    `db:"remark"     json:"remark"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// SysDeptForm 部门新增/修改表单
// label 是字段的中文名，校验不通过时用它生成提示，例如"部门名称不能为空"；
// trim:"true" 表示先去掉首尾的空白再校验。
type SysDeptForm struct {
	Id        int64     `db:"id"         json:"id"`
	ParentId  int64     `db:"parent_id"  json:"parent_id" validate:"gte=0"                    label:"上级部门"`
	Name      string    `db:"name"       json:"name"      validate:"required,min=1,max=64"    label:"部门名称"   trim:"true"`
	Principal string    `db:"principal"  json:"principal" validate:"omitempty,max=64"         label:"部门负责人"`
	Phone     string    `db:"phone"      json:"phone"     validate:"omitempty,max=20"         label:"手机号"`
	Email     string    `db:"email"      json:"email"     validate:"omitempty,email,max=128"  label:"邮箱"`
	Sort      int       `db:"sort"       json:"sort"      validate:"gte=0"                    label:"排序"`
	Status    int16     `db:"status"     json:"status"    validate:"oneof=0 1"                label:"状态"`
	Remark    string    `db:"remark"     json:"remark"    validate:"omitempty,max=256"        label:"备注"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// SysDeptSearchParams 部门搜索参数（全量返回，前端处理树结构）
type SysDeptSearchParams struct {
	Name   string `json:"name"   default:""`
	Status string `json:"status" default:""`
}
