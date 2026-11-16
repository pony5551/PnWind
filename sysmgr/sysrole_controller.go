package sysmgr

import (
	"github.com/pony5551/PnWind/core"
	"github.com/pony5551/PnWind/core/controller"
	"github.com/pony5551/PnWind/core/pncontext"
	"github.com/pony5551/PnWind/syscore/utils"
	"github.com/pony5551/PnWind/sysmgr/models"
)

type SysRoleCont struct {
	controller.ControllerBase
	SVC SysRoleService
}

// Page_menusetget 获取角色已配置的菜单集合
func (c *SysRoleCont) Page_menusetget(pnctx *pncontext.PnContext) error {
	var dataForm models.DataIdForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	menuSet := new(SysRoleMenuSet)
	if err := c.SVC.MenuSetGet(pnctx, &dataForm, menuSet); err != nil {
		return err
	}
	return pnctx.RespOK(menuSet.MenuSet, "")
}

// Page_menusetsave 保存角色菜单权限配置
func (c *SysRoleCont) Page_menusetsave(pnctx *pncontext.PnContext) error {
	var dataForm models.SysRole_MenuSetSaveForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	if err := c.SVC.MenuSetSave(pnctx, &dataForm); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "操作成功")
}

// Page_list 角色列表（全量返回，前端处理树结构）
func (c *SysRoleCont) Page_list(pnctx *pncontext.PnContext) error {
	var sParams models.SysRoleSearchParams
	if err := pnctx.Bind(&sParams); err != nil {
		return err
	}
	list := []models.SysRole{} // 没有数据时返回 []，不返回 null
	if err := c.SVC.ReadAll(pnctx, &list, sParams); err != nil {
		return err
	}
	return pnctx.RespOK(list, "")
}

// Page_add 新增角色
func (c *SysRoleCont) Page_add(pnctx *pncontext.PnContext) error {
	var dataForm models.SysRoleForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	if err := c.SVC.Insert(pnctx, &dataForm); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "新增角色成功.")
}

// Page_edit 修改角色（只改已有的，不会新增）
func (c *SysRoleCont) Page_edit(pnctx *pncontext.PnContext) error {
	var dataForm models.SysRoleForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	if dataForm.Id <= 0 {
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, "id不能为空")
	}
	if err := c.SVC.Update(pnctx, &dataForm); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "修改角色成功.")
}

// pageDel 删除角色（当前业务不启用，保留完整骨架供后续使用）
func (c *SysRoleCont) pageDel(pnctx *pncontext.PnContext) error {
	if !core.GConfig.OpenDataDelete {
		return utils.RespDeleteDisabled(pnctx)
	}
	var dataForm models.DataIdsForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	if err := c.SVC.Delete(pnctx, dataForm.Ids); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "删除角色成功.")
}
