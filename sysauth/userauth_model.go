package sysauth

import (
	pg "github.com/lib/pq"
)

// SysAdmin 登录用到的管理员信息（db_sys_admin 表的一部分列）
type SysAdmin struct {
	UserId    string         `db:"user_id"`
	Username  string         `db:"username"`
	Password  string         `db:"password"`
	Password2 string         `db:"password2"` // 二次密码；空串表示没有单独设置，等同登录密码
	Avatar    string         `db:"avatar"`
	Roles     pg.StringArray `db:"roles"`
	Status    int            `db:"status"` // 1 启用，0 停用，-1 禁用，-2 冻结
}
