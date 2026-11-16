package utils

import (
	"errors"
	"strings"

	"github.com/pony5551/PnWind/core/pncontext"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"github.com/pony5551/PnWind/core/validation"

	"github.com/gofiber/fiber/v3"
)

// ---------- 统一响应助手 ----------

// RespDeleteDisabled 数据删除功能未开启（code 90001）
func RespDeleteDisabled(pnctx *pncontext.PnContext) error {
	return pnctx.RespErr(
		Resp_code_OptFail90001,
		RespCodeEnum(Resp_code_OptFail90001).CodeParse(),
	)
}

// ---------- 统一的错误出口 ----------

// ErrorHandler 处理函数、服务、中间件 return 的 error 都到这里，转成统一的响应。
// main.go 里把它交给 Fiber：core.GConfig.Fiber.ErrorHandler = utils.ErrorHandler
//
//	pncontext.NewError 建的业务错误   原样返回 code 和 message          不记日志
//	请求体解析失败                    -20001 参数解析错误               不记日志
//	参数校验失败                      -20002，message 是字段的提示      不记日志
//	Fiber 自己的错误（地址不存在等）  保持它的 HTTP 状态码              不记日志
//	其他错误、panic                   -90002 操作失败，附请求编号       记一次日志，带请求编号
//
// 其他错误的原文只进日志，不回给前端：里面可能有 SQL、表名这类内部信息。
// 前端看到的是请求编号，拿它到日志里找原文。
func ErrorHandler(ctx fiber.Ctx, err error) error {
	resp := func(code int, msg string) error {
		return ctx.Status(fiber.StatusOK).JSON(pncontext.Resp{Code: code, Message: msg})
	}
	var (
		bizErr   *pncontext.Error
		parseErr *validation.ParseError
		validErr *validation.Error
		fiberErr *fiber.Error
	)
	switch {
	case errors.As(err, &bizErr):
		return resp(bizErr.Code, bizErr.Msg)
	case errors.As(err, &parseErr):
		return resp(Resp_code_paramParsel20001, RespCodeEnum(Resp_code_paramParsel20001).CodeParse())
	case errors.As(err, &validErr):
		return resp(Resp_code_paramParsel20002, validErr.Error())
	case errors.As(err, &fiberErr):
		return ctx.Status(fiberErr.Code).SendString(fiberErr.Message)
	}
	reqID := ctx.RequestID()
	log.Errorf("请求处理出错 [%s] %s %s: %v", reqID, ctx.Method(), ctx.Path(), err)
	return resp(Resp_code_OptFail90002, RespCodeEnum(Resp_code_OptFail90002).CodeParse()+"，请求编号 "+reqID)
}

// ---------- SQL 安全助手 ----------

// EscapeLikeParam 将用户输入转义为安全的 LIKE 模糊匹配参数。
// 转义 \、% 和 _ 三个 LIKE 通配符，再包裹 %...%。
// 用法：
//
//	dbParams["name"] = escapeLikeParam(params.Name)
//	strWhere += " AND name LIKE :name ESCAPE '\\'"
func EscapeLikeParam(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return "%" + s + "%"
}
