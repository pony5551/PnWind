package sysmgr

import (
	"github.com/pony5551/PnWind/core/controller"
	"github.com/pony5551/PnWind/core/pncontext"
	"github.com/pony5551/PnWind/sysmgr/models"
)

// SysLogOptCont 操作日志控制器
type SysLogOptCont struct {
	controller.ControllerBase
	SVC SysLogOptService
}

// Page_list 操作日志列表（分页）
func (c *SysLogOptCont) Page_list(pnctx *pncontext.PnContext) error {
	var sParams models.SysLogOptSearchParams
	if err := pnctx.Bind(&sParams); err != nil {
		return err
	}
	dataRoot := new(SysLogOptListRoot)
	dataRoot.Data.List = []SysLogOptListItem{}
	dataRoot.Data.PageSize = sParams.PageSize
	dataRoot.Data.CurrentPage = sParams.CurrentPage
	c.SVC.ReadList(pnctx, dataRoot, sParams)
	return pnctx.RespOK(dataRoot.Data, "")
}

// Page_del 批量删除操作日志（仅限 superadmin）
func (c *SysLogOptCont) Page_del(pnctx *pncontext.PnContext) error {
	if pnctx.TokenList == nil || pnctx.TokenList.GetUserId() != "superadmin" {
		return pnctx.RespErr(403, "无权操作，仅限超级管理员")
	}
	var dataForm models.DataIdsForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	if err := c.SVC.Delete(pnctx, dataForm.Ids); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "删除操作日志成功.")
}

// Page_clear 清空操作日志（仅限 superadmin）
func (c *SysLogOptCont) Page_clear(pnctx *pncontext.PnContext) error {
	if pnctx.TokenList == nil || pnctx.TokenList.GetUserId() != "superadmin" {
		return pnctx.RespErr(403, "无权操作，仅限超级管理员")
	}
	if err := c.SVC.Clear(pnctx); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "操作日志清空成功.")
}
