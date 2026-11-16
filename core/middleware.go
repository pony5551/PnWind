package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/pony5551/PnWind/core/pnlibs"
	log "github.com/pony5551/PnWind/core/pnlogrus"

	"github.com/gofiber/fiber/v3"
)

// -------------------------------------------------------------------------
// 基础中间件（Boot 建好 Fiber 实例之后、PreSetup 之前挂上）
//
// 顺序不能换：
//   1. recoverPanic 外面一层 panic 兜底：兜的是下面两层，以及访问日志里调用的错误出口
//   2. requestID    请求编号：每个请求一个，写进响应头 X-Request-ID，之后的日志都带上它
//   3. accessLog    访问日志：包住后面的全部处理；处理函数 return 的错误在这里交给统一的错误出口转成响应，
//                   再记下最终的状态码和耗时
//   4. recoverPanic 里面一层 panic 兜底：处理函数 panic 时转成错误交给第 3 步，进程不退出。
//                   它在访问日志的里面，panic 的请求才有访问日志和最终的状态码
// -------------------------------------------------------------------------

// requestID 请求编号中间件。之后用 ctx.RequestID() 取。
//
// 客户端可以自带编号（请求头 X-Request-ID），方便把前端、网关和这里的日志串起来；但只采用合规的，
// 其余的重新生成：编号会原样写进日志，不能让客户端往里塞伪造的内容或者特别长的串。
func requestID(ctx fiber.Ctx) error {
	id := ctx.Get(fiber.HeaderXRequestID)
	if !validRequestID(id) {
		id = newRequestID()
		// 请求头里的也换掉：响应头被清掉时（例如发送文件出错），ctx.RequestID() 会退回来读请求头
		ctx.Request().Header.Set(fiber.HeaderXRequestID, id)
	}
	ctx.Set(fiber.HeaderXRequestID, id)
	return ctx.Next()
}

// validRequestID 合规的请求编号：1 到 64 个字符，只有字母、数字、下划线和短横线。
func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

// newRequestID 生成 16 位十六进制的请求编号。
func newRequestID() string {
	var b [8]byte
	_, _ = rand.Read(b[:]) // 不会失败：随机源出问题时 crypto/rand 直接结束进程
	return hex.EncodeToString(b[:])
}

// accessLog 访问日志中间件：每个请求一行，记方法、地址、状态码、耗时、来源地址和请求编号。
// 各组路由处理的请求、被 PreSetup 的中间件拦下的请求都记；
// 落到 Setup（静态文件兜底）那一步的请求不记，见 noAccessLog。
func accessLog(ctx fiber.Ctx) error {
	start := time.Now()
	if err := ctx.Next(); err != nil {
		if herr := ctx.App().ErrorHandler(ctx, err); herr != nil {
			return herr
		}
	}
	if ctx.Locals(noAccessLogKey{}) == nil {
		log.Infof("%s %s %d %s %s [%s]", ctx.Method(), ctx.Path(), ctx.Response().StatusCode(),
			time.Since(start), ctx.IP(), ctx.RequestID())
	}
	return nil
}

// noAccessLogKey 是在请求上做"不记访问日志"记号时用的键。
type noAccessLogKey struct{}

// noAccessLog 给这次请求做上记号：不记访问日志。
//
// 挂在 Setup 之前：走到那一步的请求没有被任何一组路由接走，是冲着静态文件来的。
// 这样内核不用知道各组路由的地址前缀。
func noAccessLog(ctx fiber.Ctx) error {
	ctx.Locals(noAccessLogKey{}, true)
	return ctx.Next()
}

// recoverPanic panic 兜底中间件：记下 panic 和堆栈，把它转成普通的错误交给上一层。
// 堆栈只进日志，不放进错误里：错误的文字有可能被回给前端。
func recoverPanic(ctx fiber.Ctx) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("panic [%s] %s %s: %v\n%s", ctx.RequestID(), ctx.Method(), ctx.Path(), r, debug.Stack())
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return ctx.Next()
}

// -------------------------------------------------------------------------
// 健康检查（Boot 把它们注册在基础中间件的前面：不经过鉴权，不记访问日志，也不生成请求编号）
//
//   /livez   进程在就返回 200，给容器、守护进程探活用
//   /readyz  正常时返回 200；收到退出信号之后返回 503，负载均衡据此不再转发新请求
// -------------------------------------------------------------------------

func livez(ctx fiber.Ctx) error {
	return ctx.SendString("ok")
}

func readyz(ctx fiber.Ctx) error {
	if pnlibs.GetProcessQuitReady() {
		return ctx.Status(fiber.StatusServiceUnavailable).SendString("stopping")
	}
	return ctx.SendString("ok")
}
