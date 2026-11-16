package models

import (
	"time"

	pg "github.com/lib/pq"
)

// SysAdmin 管理员（对应 db_sys_admin 表）。接口里的字段名和列名一样
type SysAdmin struct {
	Id        int64          `db:"id"         json:"id"`
	UserId    string         `db:"user_id"    json:"user_id"`
	Username  string         `db:"username"   json:"username"`
	RealName  string         `db:"real_name"  json:"real_name"`
	Gender    int16          `db:"gender"     json:"gender"`
	Avatar    string         `db:"avatar"     json:"avatar"`
	DeptId    int64          `db:"dept_id"    json:"dept_id"`
	Roles     pg.StringArray `db:"roles"      json:"roles"`
	Phone     string         `db:"phone"      json:"phone"`
	Email     string         `db:"email"      json:"email"`
	Status    int            `db:"status"     json:"status"`
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
	DeletedAt *time.Time     `db:"deleted_at" json:"deleted_at"` // 删除时间，没有删除时是 null
}

// SysAdminForm 管理员新增/修改表单
type SysAdminForm struct {
	Id       int64          `db:"id"        json:"id"`
	Username string         `db:"username"  json:"username"  validate:"required,min=2,max=64"   label:"用户名称" trim:"true"`
	RealName string         `db:"real_name" json:"real_name" validate:"omitempty,max=64"        label:"真实姓名"`
	Password string         `db:"password"  json:"password"  validate:"omitempty,min=6,max=128" label:"密码"`
	Gender   int16          `db:"gender"    json:"gender"    validate:"oneof=0 1 2"             label:"性别"`
	DeptId   int64          `db:"dept_id"   json:"dept_id"   validate:"gte=0"                   label:"所属部门"`
	Phone    string         `db:"phone"     json:"phone"     validate:"omitempty,max=20"        label:"手机号"`
	Email    string         `db:"email"     json:"email"     validate:"omitempty,email,max=128" label:"邮箱"`
	Status   int            `db:"status"    json:"status"    validate:"oneof=-1 0 1"            label:"状态"`
	Roles    pg.StringArray `db:"roles"     json:"roles"`
	Avatar   string         `db:"avatar"    json:"avatar"    validate:"omitempty,max=204800"    label:"头像"`
}

// SysAdminSearchParams 管理员列表搜索参数
type SysAdminSearchParams struct {
	Username string `json:"username"`
	Phone    string `json:"phone"`
	Status   string `json:"status"`  // "" 全部 / "0" 停用 / "1" 启用
	DeptId   int64  `json:"dept_id"` // 0 = 不过滤，>0 = 按部门过滤
	PageParams
}
