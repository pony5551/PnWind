package sysmgr

import (
	"strconv"

	"github.com/pony5551/PnWind/core/controller"
	"github.com/pony5551/PnWind/core/pncontext"
	"github.com/pony5551/PnWind/syscore/utils"
	"github.com/pony5551/PnWind/sysmgr/models"
)

// SysUsersCont 用户管理页的控制器（用户表 db_sys_users）
type SysUsersCont struct {
	controller.ControllerBase
	SVC SysUsersService
}

// Page_list 用户列表（分页）
func (c *SysUsersCont) Page_list(pnctx *pncontext.PnContext) error {
	var sParams models.SysUsersSearchParams
	if err := pnctx.Bind(&sParams); err != nil {
		return err
	}
	dataRoot := new(SysUsersListRoot)
	dataRoot.Data.List = []SysUsersListItem{}
	dataRoot.Data.PageSize = sParams.PageSize
	dataRoot.Data.CurrentPage = sParams.CurrentPage
	c.SVC.ReadList(pnctx, dataRoot, sParams)
	return pnctx.RespOK(dataRoot.Data, "")
}

// Page_add 新增用户
func (c *SysUsersCont) Page_add(pnctx *pncontext.PnContext) error {
	var dataForm models.SysUsersAddForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	if err := c.SVC.Insert(pnctx, &dataForm); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "新增用户成功.")
}

// Page_edit 修改用户基本信息
func (c *SysUsersCont) Page_edit(pnctx *pncontext.PnContext) error {
	var dataForm models.SysUsersEditForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	if dataForm.Id <= 0 {
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, "id不能为空")
	}
	if err := c.SVC.Update(pnctx, &dataForm); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "修改用户成功.")
}

// Page_resetpwd 重置用户密码
func (c *SysUsersCont) Page_resetpwd(pnctx *pncontext.PnContext) error {
	var dataForm models.SysUsersResetPwdForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	if dataForm.Password == "" && dataForm.PayPassword == "" {
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, "请至少填写一个密码")
	}
	if err := c.SVC.ResetPassword(pnctx, &dataForm); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "密码重置成功.")
}

// Page_del 批量软删除用户（仅限 superadmin）
func (c *SysUsersCont) Page_del(pnctx *pncontext.PnContext) error {
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
	return pnctx.RespOK(nil, "删除用户成功.")
}

// Page_rolelist 获取用户角色列表（按上级角色 parent_id 过滤）
// 请求体可选：{ "parent_id": 2 }
func (c *SysUsersCont) Page_rolelist(pnctx *pncontext.PnContext) error {
	params := make(map[string]interface{})
	_ = pnctx.Bind(&params)
	var parentId int64
	if v, ok := params["parent_id"]; ok {
		switch t := v.(type) {
		case float64:
			parentId = int64(t)
		case string:
			parentId, _ = strconv.ParseInt(t, 10, 64)
		}
	}
	list, err := c.SVC.RoleList(pnctx, parentId)
	if err != nil {
		return err
	}
	return pnctx.RespOK(list, "")
}
