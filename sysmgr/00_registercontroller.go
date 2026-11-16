package sysmgr

import (
	"github.com/pony5551/PnWind/core"
	"github.com/pony5551/PnWind/syscore/routers"
)

func init() {
	poolName := "web"
	// 集中在这里注册：一个页面一段，新增页面在这里加一段。
	// 花括号只是为了让每一段都能用 pagePath、LController 这两个名字。
	{
		pagePath := "/sysmgr/sysset/sysadmin"
		LController := &SysAdminCont{}
		LController.CntlrParentName = "系统设置"
		LController.CntlrName = "管理员管理"
		routers.RegisterController(core.GConfig,poolName, pagePath, LController)
	}
	{
		pagePath := "/sysmgr/sysset/sysrole"
		LController := &SysRoleCont{}
		LController.CntlrParentName = "系统管理"
		LController.CntlrName = "角色管理"
		routers.RegisterController(core.GConfig,poolName, pagePath, LController)
	}
	{
		pagePath := "/sysmgr/sysset/sysmenus"
		LController := &SysMenusCont{}
		LController.CntlrParentName = "系统设置"
		LController.CntlrName = "菜单管理"
		routers.RegisterController(core.GConfig,poolName, pagePath, LController)
	}
	{
		pagePath := "/sysmgr/sysset/sysdept"
		LController := &SysDeptCont{}
		LController.CntlrParentName = "系统设置"
		LController.CntlrName = "部门管理"
		routers.RegisterController(core.GConfig,poolName, pagePath, LController)
	}
	{
		pagePath := "/sysmgr/sysset/sysconfigs"
		LController := &SysConfigsCont{}
		LController.CntlrParentName = "系统设置"
		LController.CntlrName = "基础设置"
		routers.RegisterController(core.GConfig,poolName, pagePath, LController)
	}
	{
		pagePath := "/sysmgr/sysset/systokenlist"
		LController := &SysTokenListCont{}
		LController.CntlrParentName = "系统设置"
		LController.CntlrName = "在线用户"
		routers.RegisterController(core.GConfig,poolName, pagePath, LController)
	}
	{
		pagePath := "/sysmgr/syslog/syslogopt"
		LController := &SysLogOptCont{}
		LController.CntlrParentName = "系统日志"
		LController.CntlrName = "操作日志"
		routers.RegisterController(core.GConfig,poolName, pagePath, LController)
	}
	{
		pagePath := "/sysmgr/sysuser/sysusers"
		LController := &SysUsersCont{}
		LController.CntlrParentName = "用户管理"
		LController.CntlrName = "用户列表"
		routers.RegisterController(core.GConfig,poolName, pagePath, LController)
	}
}
