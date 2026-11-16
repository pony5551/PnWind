package sysmgr

import (
	"strconv"

	"github.com/pony5551/PnWind/core/controller"
	"github.com/pony5551/PnWind/core/pncontext"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"github.com/pony5551/PnWind/syscore/utils"
	"github.com/pony5551/PnWind/sysmgr/models"
)

// SysAdminCont 管理员管理页的控制器（管理员表 db_sys_admin）
type SysAdminCont struct {
	controller.ControllerBase
	SVC SysAdminService
}

// Page_list 用户列表（分页）
func (c *SysAdminCont) Page_list(pnctx *pncontext.PnContext) error {
	var sParams models.SysAdminSearchParams
	if err := pnctx.Bind(&sParams); err != nil {
		return err
	}
	dataRoot := &SysAdminListRoot{}
	dataRoot.Data.List = []models.SysAdmin{}
	c.SVC.ReadList(pnctx, dataRoot, sParams)
	return pnctx.RespOK(dataRoot.Data, "")
}

// Page_add 新增用户
func (c *SysAdminCont) Page_add(pnctx *pncontext.PnContext) error {
	var form models.SysAdminForm
	if err := pnctx.Bind(&form); err != nil {
		return err
	}
	if err := c.SVC.Insert(pnctx, &form); err != nil {
		log.Errorln(err)
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, err.Error())
	}
	return pnctx.RespOK(nil, "新增用户成功.")
}

// Page_edit 修改用户（不修改密码）
func (c *SysAdminCont) Page_edit(pnctx *pncontext.PnContext) error {
	var form models.SysAdminForm
	if err := pnctx.Bind(&form); err != nil {
		return err
	}
	if form.Id <= 0 {
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, "id不能为空")
	}
	if err := c.SVC.Update(pnctx, &form); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "修改用户成功.")
}

// Page_del 批量删除用户（逻辑删除）
func (c *SysAdminCont) Page_del(pnctx *pncontext.PnContext) error {
	var dataForm models.DataIdsForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	if err := c.SVC.Delete(pnctx, dataForm.Ids); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "删除用户成功.")
}

// Page_switch 切换用户启用/停用状态
// 请求体：{ "id": 1, "status": 1 }
func (c *SysAdminCont) Page_switch(pnctx *pncontext.PnContext) error {
	params := make(map[string]interface{})
	if err := pnctx.Bind(&params); err != nil {
		return err
	}
	idStr := utils.GetInterfaceToString(params["id"])
	if idStr == "" {
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, "id不能为空")
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)
	stateVal, ok := params["status"]
	if !ok {
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, "status不能为空")
	}
	state := -999 //哨兵值:类型不合法时保持越界,由下方校验拦截
	switch t := stateVal.(type) {
	case float64:
		state = int(t)
	case string:
		if n, err := strconv.Atoi(t); err == nil {
			state = n
		}
	}
	//状态开关仅允许 1=启用 / 0=停用(-1禁用/-2冻结不经此接口)
	if state != 0 && state != 1 {
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, "status仅允许0(停用)或1(启用)")
	}
	if err := c.SVC.SwitchState(pnctx, id, state); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "状态更新成功.")
}

// Page_resetpwd 重置用户的登录密码、二次密码（至少给一个，给了哪个改哪个）
// 请求体：{ "id": 1, "new_pwd": "xxx", "new_pwd2": "yyy" }
func (c *SysAdminCont) Page_resetpwd(pnctx *pncontext.PnContext) error {
	params := make(map[string]interface{})
	if err := pnctx.Bind(&params); err != nil {
		return err
	}
	idStr := utils.GetInterfaceToString(params["id"])
	if idStr == "" {
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, "id不能为空")
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)
	newPwd, _ := params["new_pwd"].(string)
	newPwd2, _ := params["new_pwd2"].(string)
	if newPwd == "" && newPwd2 == "" {
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, "新密码不能为空")
	}
	if err := c.SVC.ResetPwd(pnctx, id, newPwd, newPwd2); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "密码重置成功.")
}

// Page_setavatar 保存用户 base64 头像
// 请求体：{ "id": 1, "avatar": "data:image/png;base64,..." }
func (c *SysAdminCont) Page_setavatar(pnctx *pncontext.PnContext) error {
	params := make(map[string]interface{})
	if err := pnctx.Bind(&params); err != nil {
		return err
	}
	idStr := utils.GetInterfaceToString(params["id"])
	if idStr == "" {
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, "id不能为空")
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)
	avatar, _ := params["avatar"].(string)
	if err := c.SVC.SetAvatar(pnctx, id, avatar); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "头像保存成功.")
}

// Page_rolelist 获取角色列表（用于前端下拉选择，支持按上级角色 parent_id 过滤）
// 请求体可选：{ "parent_id": 1 }
func (c *SysAdminCont) Page_rolelist(pnctx *pncontext.PnContext) error {
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

// Page_rolesave 保存用户角色分配
// 请求体：{ "id": 1, "roles": ["code1","code2"] }
func (c *SysAdminCont) Page_rolesave(pnctx *pncontext.PnContext) error {
	params := make(map[string]interface{})
	if err := pnctx.Bind(&params); err != nil {
		return err
	}
	idStr := utils.GetInterfaceToString(params["id"])
	if idStr == "" {
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, "id不能为空")
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)

	var roles []string
	if raw, ok := params["roles"]; ok {
		if arr, ok := raw.([]interface{}); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					roles = append(roles, s)
				}
			}
		}
	}
	if err := c.SVC.RoleSave(pnctx, id, roles); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "角色分配成功.")
}

// Page_deptlist 获取部门列表（用于左侧树及表单下拉）
func (c *SysAdminCont) Page_deptlist(pnctx *pncontext.PnContext) error {
	list, err := c.SVC.DeptList(pnctx)
	if err != nil {
		return err
	}
	return pnctx.RespOK(list, "")
}
