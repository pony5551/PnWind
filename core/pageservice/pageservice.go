// Package pageservice 页面服务的基类 PageService。
package pageservice

// PageService 页面服务的基类：每个页面的业务服务（<页面>_service.go 里的 XxxService，例如 sysmgr 的 SysDeptService）内嵌它。
//
// 它不是服务插件：服务插件实现的是 core/service 的 Service、放进 core.GConfig.Services，页面服务不放进去。
// 写成空结构体：不占空间；以后页面服务要有公共的方法，加在它上面，内嵌它的服务都能直接用。
type PageService struct{}
