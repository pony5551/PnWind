package models

import (
	"time"

	pg "github.com/lib/pq"
)

// SysUsers 用户（对应 db_sys_users 表，列表查询不返回密码字段）。接口里的字段名和列名一样
type SysUsers struct {
	Id            int64          `db:"id"             json:"id"`
	UserId        string         `db:"user_id"        json:"user_id"`
	Name          string         `db:"name"           json:"name"`
	Type          int16          `db:"type"           json:"type"`
	Status        int16          `db:"status"         json:"status"`
	Email         string         `db:"email"          json:"email"`
	Phone         string         `db:"phone"          json:"phone"`
	Roles         pg.StringArray `db:"roles"          json:"roles"`
	CashBalance   float64        `db:"cash_balance"   json:"cash_balance"`
	FundsBalance  float64        `db:"funds_balance"  json:"funds_balance"`
	FrozenBalance float64        `db:"frozen_balance" json:"frozen_balance"`
	Config        string         `db:"config"         json:"config"`
	VerifiedId    int64          `db:"verified_id"    json:"verified_id"` // 实名信息的 id，0 表示还没有实名
	Remark        string         `db:"remark"         json:"remark"`
	LastLoginIp   string         `db:"last_login_ip"  json:"last_login_ip"`
	LastLoginAt   *time.Time     `db:"last_login_at"  json:"last_login_at"`
	CreatedIp     string         `db:"created_ip"     json:"created_ip"`
	CreatedAt     time.Time      `db:"created_at"     json:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"     json:"updated_at"`
}

// SysUsersSearchParams 用户列表搜索参数
type SysUsersSearchParams struct {
	UserId      string `json:"user_id"     default:""`
	Name        string `json:"name"        default:""`
	Email       string `json:"email"       default:""`
	Phone       string `json:"phone"       default:""`
	Type        int16  `json:"type"        default:"0"`
	Status      *int16 `json:"status"` // nil = 不过滤
	CurrentPage uint64 `json:"currentPage" default:"1"`
	PageSize    uint64 `json:"pageSize"    default:"20"`
}

// SysUsersAddForm 用户新增表单。user_id 不在表单里，由库里的序列给
type SysUsersAddForm struct {
	Name     string         `db:"name"     json:"name"     validate:"required,max=128"        label:"用户名称" trim:"true"`
	Type     int16          `db:"type"     json:"type"     validate:"required,oneof=1 2 3"    label:"用户类型"`
	Status   int16          `db:"status"   json:"status"   validate:"oneof=0 1 2 3"           label:"用户状态"`
	Email    string         `db:"email"    json:"email"    validate:"required,email,max=128"  label:"邮箱"`
	Phone    string         `db:"phone"    json:"phone"    validate:"required,max=20"         label:"手机号"`
	Roles    pg.StringArray `db:"roles"    json:"roles"`
	Password string         `db:"password" json:"password" validate:"omitempty,min=6,max=255" label:"登录密码"`
	Config   string         `db:"config"   json:"config"   validate:"omitempty,json"          msg:"用户配置不是合法的 JSON"` // JSON 文本，空串按 {} 存
	Remark   string         `db:"remark"   json:"remark"`
}

// SysUsersEditForm 用户修改表单（不允许直接修改密码、余额和实名信息）
type SysUsersEditForm struct {
	Id     int64          `db:"id"     json:"id"`
	Name   string         `db:"name"   json:"name"   validate:"required,max=128"       label:"用户名称" trim:"true"`
	Type   int16          `db:"type"   json:"type"   validate:"required,oneof=1 2 3"   label:"用户类型"`
	Status int16          `db:"status" json:"status" validate:"oneof=0 1 2 3"          label:"用户状态"`
	Email  string         `db:"email"  json:"email"  validate:"required,email,max=128" label:"邮箱"`
	Phone  string         `db:"phone"  json:"phone"  validate:"required,max=20"        label:"手机号"`
	Roles  pg.StringArray `db:"roles"  json:"roles"`
	Config string         `db:"config" json:"config" validate:"omitempty,json"         msg:"用户配置不是合法的 JSON"` // JSON 文本，空串按 {} 存
	Remark string         `db:"remark" json:"remark"`
}

// SysUsersResetPwdForm 重置密码表单
type SysUsersResetPwdForm struct {
	Id          int64  `db:"id"           json:"id"           validate:"required,gt=0"           label:"id"`
	Password    string `db:"password"     json:"password"     validate:"omitempty,min=6,max=255" label:"登录密码"`
	PayPassword string `db:"pay_password" json:"pay_password" validate:"omitempty,min=6,max=255" label:"资金密码"`
}
