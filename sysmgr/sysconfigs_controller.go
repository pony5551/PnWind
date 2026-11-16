package sysmgr

import (
	"github.com/pony5551/PnWind/core"
	"github.com/pony5551/PnWind/core/controller"
	"github.com/pony5551/PnWind/core/pncontext"
	"github.com/pony5551/PnWind/syscore/utils"
	"github.com/pony5551/PnWind/sysmgr/models"
)

type SysConfigsCont struct {
	controller.ControllerBase
	SVC SysConfigsService
}

// Page_list 配置列表
func (c *SysConfigsCont) Page_list(pnctx *pncontext.PnContext) error {
	var aParams models.SysConfigSearchParams
	if err := pnctx.Bind(&aParams); err != nil {
		return err
	}
	if aParams.PageSize == 0 {
		aParams.PageSize = 100
	}
	if aParams.CurrentPage == 0 {
		aParams.CurrentPage = 1
	}
	data, err := c.SVC.ReadList(pnctx, aParams)
	if err != nil {
		return err
	}
	return pnctx.RespOK(data, "")
}

// Page_add 新增配置项
func (c *SysConfigsCont) Page_add(pnctx *pncontext.PnContext) error {
	var form models.SysConfigAddForm
	if err := pnctx.Bind(&form); err != nil {
		return err
	}
	if err := c.SVC.Insert(pnctx, &form); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "新增成功.")
}

// Page_edit 修改配置值
func (c *SysConfigsCont) Page_edit(pnctx *pncontext.PnContext) error {
	var form models.SysConfigEditForm
	if err := pnctx.Bind(&form); err != nil {
		return err
	}
	if err := c.SVC.Update(pnctx, &form); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "修改成功.")
}

// Page_del 删除配置项（非 sys 项）
func (c *SysConfigsCont) Page_del(pnctx *pncontext.PnContext) error {
	if !core.GConfig.OpenDataDelete {
		return utils.RespDeleteDisabled(pnctx)
	}
	var form models.DataIdsForm
	if err := pnctx.Bind(&form); err != nil {
		return err
	}
	if err := c.SVC.Delete(pnctx, form.Ids); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "删除成功.")
}

// Page_loadsysconfig 重新加载系统配置到内存
func (c *SysConfigsCont) Page_loadsysconfig(pnctx *pncontext.PnContext) error {
	if err := utils.LoadSysConfig(pnctx.DB, core.GConfig.SysConfigCache); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "加载成功.")
}

// Page_getsysconfig 读取系统基础设置
func (c *SysConfigsCont) Page_getsysconfig(pnctx *pncontext.PnContext) error {
	confs := map[string]string{}
	if err := utils.GetSysConfig(pnctx.DB, confs); err != nil {
		return err
	}
	return pnctx.RespOK(confs, "获取成功.")
}
