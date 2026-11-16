package sysmgr

import (
	"github.com/pony5551/PnWind/core/controller"
	"github.com/pony5551/PnWind/core/pncontext"
	"github.com/pony5551/PnWind/sysmgr/models"
)

type SysTokenListCont struct {
	controller.ControllerBase
	SVC SysTokenListService
}

// Page_list Token列表
func (c *SysTokenListCont) Page_list(pnctx *pncontext.PnContext) error {
	var sParams models.SysTokenListSearchParams
	if err := pnctx.Bind(&sParams); err != nil {
		return err
	}
	dataRoot := new(SysTokenListListRoot)
	dataRoot.Data.List = []SysTokenListListItem{}
	dataRoot.Data.PageSize = sParams.PageSize
	dataRoot.Data.CurrentPage = sParams.CurrentPage
	c.SVC.ReadList(pnctx, dataRoot, sParams)
	return pnctx.RespOK(dataRoot.Data, "")
}

// Page_del 强制下线（删除Token记录）
func (c *SysTokenListCont) Page_del(pnctx *pncontext.PnContext) error {
	var dataForm models.DataIdsForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	if err := c.SVC.Delete(pnctx, dataForm.Ids); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "强制下线成功.")
}
