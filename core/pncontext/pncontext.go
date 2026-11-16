package pncontext

import (
	"errors"
	"fmt"

	"github.com/pony5551/PnWind/core"
	"github.com/pony5551/PnWind/core/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/jmoiron/sqlx"
	"github.com/valyala/fasthttp"
)

// TokenInfo 认证上下文接口，由具体的 Token 结构体实现
type TokenInfo interface {
	GetUserId() string
}

type PnContext struct {
	fiber.Ctx // Fiber v3：Ctx 由结构体指针改为接口，嵌入不带 *
	DB        *sqlx.DB
	DB2       *sqlx.DB
	Message   string
	Act       string
	TokenList TokenInfo
	Extend    any
}

// BodyParser 兼容垫片：Fiber v3 移除了 Ctx.BodyParser，统一改走 Bind().Body()。
// 保留此方法名：写成 pnctx.BodyParser(&x) 的代码不用改。
func (c *PnContext) BodyParser(out any) error {
	return c.Ctx.Bind().Body(out)
}

// Context 兼容垫片：Fiber v3 的 Ctx.Context() 改为返回 context.Context，
// 而现有代码把 pnctx.Context() 当 *fasthttp.RequestCtx 使用
// （例如登录时取 RemoteAddr）。这里遮蔽为返回底层 RequestCtx，
// 保持原有的调用语义不变。注意：PnContext 仅用于业务层，从不作为 fiber.Ctx 传回框架，
// 因此遮蔽接口的 Context() 不会影响 Fiber 自身。
func (c *PnContext) Context() *fasthttp.RequestCtx {
	return c.Ctx.RequestCtx()
}

func (c *PnContext) IsMobileDevice() bool {
	uVal := c.Ctx.Get("x-mobile-device")
	if uVal == "1" {
		return true
	}
	return false
}

// tokenKey 是鉴权中间件把登录信息存进 fiber.Ctx 时用的键
type tokenKey struct{}

// SetToken 由鉴权中间件调用：把当前登录信息记在这次请求上，之后创建的 PnContext 会带上它
func SetToken(ctx fiber.Ctx, token TokenInfo) {
	ctx.Locals(tokenKey{}, token)
}

func (c *PnContext) GetToken() {
	if obj, isOk := c.Ctx.Locals(tokenKey{}).(TokenInfo); isOk {
		c.TokenList = obj
	}
}

// Bind 把 JSON 请求体解析进 out，一步完成四件事：
// 填 default 标签的默认值、解析、去掉 trim 标签字段的首尾空白、按 validate 标签校验。
// 出错直接 return err，统一的错误出口会回对应的错误响应（解析失败 -20001，校验失败 -20002）。
//
//	var dataForm models.SysDeptForm
//	if err := pnctx.Bind(&dataForm); err != nil {
//		return err
//	}
func (c *PnContext) Bind(out any) error {
	return validation.Bind(c.BodyRaw(), out)
}

// Resp 统一的响应格式 {"code":0, "message":"", "data":null}。code 为 0 表示成功。
type Resp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// RespOK 返回标准成功响应
//
//	{"code":0, "message":msg, "data":data}
func (c *PnContext) RespOK(data any, msg string) error {
	return c.Status(fiber.StatusOK).JSON(Resp{Code: 0, Message: msg, Data: data})
}

// RespErr 返回标准错误响应
//
//	{"code":code, "message":msg, "data":null}
func (c *PnContext) RespErr(code int, msg string) error {
	c.Message = msg
	return c.Status(fiber.StatusOK).JSON(Resp{Code: code, Message: msg})
}

// UserId 返回当前登录用户的标识；未登录返回空串。
func (c *PnContext) UserId() string {
	if c.TokenList == nil {
		return ""
	}
	return c.TokenList.GetUserId()
}

// DBByName 按库名取主库以外的数据库，例如 pnctx.DBByName("db2")。库名是配置文件里 dbs 下的键。
func (c *PnContext) DBByName(name string) (*sqlx.DB, error) {
	return core.GConfig.GetDBByName(name)
}

// Tx 在主库的一个事务里执行 fn：fn 返回错误或 panic 时回滚，否则提交；提交失败作为错误返回。
//
//	err := pnctx.Tx(func(tx *sqlx.Tx) error {
//		if _, err := tx.Exec(sql1, ...); err != nil {
//			return err
//		}
//		_, err := tx.Exec(sql2, ...)
//		return err
//	})
func (c *PnContext) Tx(fn func(tx *sqlx.Tx) error) error {
	tx, err := c.DB.Beginx()
	if err != nil {
		return fmt.Errorf("开启事务: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Join(err, fmt.Errorf("回滚事务: %w", rbErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务: %w", err)
	}
	return nil
}

// Error 业务错误：错误码加提示。处理函数或服务 return 它，统一的错误出口原样回给前端，不记日志。
//
//	return pncontext.NewError(utils.Resp_code_loginFail10002, "用户不存在")
type Error struct {
	Code int
	Msg  string
}

// NewError 创建业务错误
func NewError(code int, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s（%d）", e.Msg, e.Code)
}

func WithPnContext(ctx fiber.Ctx) (*PnContext, error) {
	var err error
	var LDB1 *sqlx.DB
	var LDB2 *sqlx.DB
	dbs := core.GConfig.Conf.GetStringMap("dbs")
	if len(dbs) > 0 {
		for k, _ := range dbs {
			//log.Printf("%v, %v", k, v)
			if k == "db1" {
				LDB1, err = core.GConfig.GetDBByName("db1")
				if err != nil {
					return nil, err
				}
			} else if k == "db2" {
				LDB2, err = core.GConfig.GetDBByName("db2")
				if err != nil {
					return nil, err
				}
			}
		}
	}
	pnContext := PnContext{
		Ctx:       ctx,
		DB:        LDB1,
		DB2:       LDB2,
		Message:   "",
		Act:       ctx.Params("act"), // 地址的最后一段，路由里的 :act
		TokenList: nil,
		Extend:    nil,
	}
	pnContext.GetToken()
	return &pnContext, nil
}
