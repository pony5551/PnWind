package sysmgr

import (
	"time"

	"github.com/pony5551/PnWind/core"
	"github.com/pony5551/PnWind/core/controller"
	"github.com/pony5551/PnWind/core/pncontext"
	"github.com/pony5551/PnWind/syscore/utils"
	"github.com/pony5551/PnWind/sysmgr/models"
)

type SysDeptCont struct {
	controller.ControllerBase
	SVC SysDeptService
}

// Page_list 部门列表（全量返回，前端处理树结构）
func (c *SysDeptCont) Page_list(pnctx *pncontext.PnContext) error {
	var sParams models.SysDeptSearchParams
	if err := pnctx.Bind(&sParams); err != nil {
		return err
	}
	list := []models.SysDept{} // 没有数据时返回 []，不返回 null
	if err := c.SVC.ReadAll(pnctx, &list, sParams); err != nil {
		return err
	}
	return pnctx.RespOK(list, "")
}

// Page_add 新增部门
func (c *SysDeptCont) Page_add(pnctx *pncontext.PnContext) error {
	var dataForm models.SysDeptForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	dataForm.CreatedAt = time.Now()
	if err := c.SVC.Insert(pnctx, &dataForm); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "新增部门成功.")
}

// Page_edit 修改部门
func (c *SysDeptCont) Page_edit(pnctx *pncontext.PnContext) error {
	var dataForm models.SysDeptForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	if dataForm.Id <= 0 {
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, "id不能为空")
	}
	if err := c.SVC.Update(pnctx, &dataForm); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "修改部门成功.")
}

// Page_del 批量删除部门
func (c *SysDeptCont) Page_del(pnctx *pncontext.PnContext) error {
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
	return pnctx.RespOK(nil, "删除部门成功.")
}
