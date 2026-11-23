// Package service 服务插件的接口 Service。
package service

import "github.com/gofiber/fiber/v3"

// Service 服务插件的接口：插件实现它，放进 core.GConfig.Services。页面的服务不是插件，内嵌的是 core/pageservice 的 PageService。
//
// Boot 在路由建好之后、开始接收连接之前，按 Services 的顺序调插件的 Setup：插件在这里建好、启动自己，常驻的部分用
// core.GConfig.Tasks.Go 登记（退出时内核等它返回）；返回错误，Boot 就不启动。路由都已经装好，在 Setup 里挂的路由排在它们后面，要挂路由放进 PreSetup。
// 说明见 core/CORE.md 的"服务插件"。
type Service interface {
	Setup(app *fiber.App, args ...interface{}) error
}
