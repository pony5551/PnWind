// Package service 服务插件的接口 Service。
package service

import "github.com/gofiber/fiber/v3"

// Service 服务插件的接口：插件实现它，放进 core.GConfig.Services。页面的服务不是插件，内嵌的是 core/pageservice 的 PageService。
//
// Boot 在服务开始接收连接之后调插件的 Setup：插件在这里启动自己，常驻的部分用 core.GConfig.Tasks.Go 登记
// （退出时内核等它返回）。这时路由已经建好，在 Setup 里挂路由不生效。说明见 core/CORE.md 的"服务插件"。
type Service interface {
	Setup(app *fiber.App)
}
