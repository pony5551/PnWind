package models

import (
	"database/sql"

	"github.com/golang-jwt/jwt/v5"
)

// SecretKey 登录令牌（JWT）的签名密钥，不写在代码里：main.go 启动时从配置文件的 JwtSecret 读入，没有配置就不启动
var SecretKey string

const JwtIssuer = "PnWind"

type LoginClaims struct {
	jwt.RegisteredClaims
	UserId     string
	UserRoles  []string
	RemoteAddr string
}

type SysMenuAuth struct {
	Id        int64          `db:"id"`
	Title     sql.NullString `db:"title"`
	PageAct   sql.NullString `db:"pageact"`
	MenuTitle sql.NullString `db:"menutitle"`
	MenuPath  sql.NullString `db:"menupath"`
	Loged     bool           `db:"loged"`
}
