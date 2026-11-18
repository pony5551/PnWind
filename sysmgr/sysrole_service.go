package sysmgr

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pony5551/PnWind/core/pageservice"
	"github.com/pony5551/PnWind/core/pncontext"
	"github.com/pony5551/PnWind/core/util"
	"github.com/pony5551/PnWind/syscore/utils"
	"github.com/pony5551/PnWind/sysmgr/models"

	pg "github.com/lib/pq"
)

type SysRoleService struct {
	pageservice.PageService
}

// SysRoleMenuSet 角色菜单集合（用于 menusetget / menusetsave）
type SysRoleMenuSet struct {
	MenuSet pg.StringArray `db:"menuset" json:"menuset"`
}

// MenuSetGet 获取角色已配置的菜单集合
func (c *SysRoleService) MenuSetGet(pnctx *pncontext.PnContext, dataForm *models.DataIdForm, JsonItem interface{}) (err error) {
	var role models.SysRole
	if err = pnctx.DB.Get(&role, "select * from db_sys_role where id=$1", dataForm.Id); err != nil {
		return
	}
	JsonItem.(*SysRoleMenuSet).MenuSet = role.MenuSet
	return
}

// MenuSetSave 保存角色菜单权限配置。menuset 是空数组就是清空这个角色的全部权限
func (c *SysRoleService) MenuSetSave(pnctx *pncontext.PnContext, dataForm *models.SysRole_MenuSetSaveForm) (err error) {
	dbParams := util.StructToMap(*dataForm, "db")
	_, err = pnctx.DB.NamedExec("UPDATE db_sys_role set menuset=:menuset where id=:id;", dbParams)
	return err
}

// ReadAll 查询所有角色（全量，不分页，前端自行处理树结构）
func (c *SysRoleService) ReadAll(pnctx *pncontext.PnContext, list *[]models.SysRole, params models.SysRoleSearchParams) error {
	if pnctx.DB == nil {
		return nil
	}
	dbParams := map[string]interface{}{}
	strWhere := " WHERE 1=1"
	if params.Name != "" {
		dbParams["name"] = utils.EscapeLikeParam(params.Name)
		strWhere += ` AND name LIKE :name ESCAPE '\'`
	}
	if params.Code != "" {
		dbParams["code"] = utils.EscapeLikeParam(params.Code)
		strWhere += ` AND code LIKE :code ESCAPE '\'`
	}
	if params.Status != "" {
		dbParams["status"] = params.Status
		strWhere += " AND status=:status"
	}
	strSQL := "SELECT * FROM db_sys_role" + strWhere + " ORDER BY parent_id ASC, sort ASC, id ASC"
	rows, err := pnctx.DB.NamedQuery(strSQL, dbParams)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v models.SysRole
		if err = rows.StructScan(&v); err != nil {
			return err
		}
		*list = append(*list, v)
	}
	return nil
}

// Insert 新增角色。创建时间 created_at 由表上的默认值填
func (c *SysRoleService) Insert(pnctx *pncontext.PnContext, dataForm *models.SysRoleForm) (err error) {
	dbParams := util.StructToMap(*dataForm, "db")
	sqlStr := `
INSERT INTO db_sys_role (parent_id, name, code, sort, status, remark)
VALUES (:parent_id, :name, :code, :sort, :status, :remark);`
	_, err = pnctx.DB.NamedExec(sqlStr, dbParams)
	return c.dupErr(err)
}

// dupErr 角色编码和别的角色重复时，换成说得出原因的提示；别的错误原样返回
func (c *SysRoleService) dupErr(err error) error {
	return utils.DupErr(err, map[string]string{"db_sys_role_code_key": "角色编码已存在"})
}

// Update 修改角色：只改已有的角色，id 不存在回"角色不存在"，不会新增。创建时间、菜单权限不动。
// 备注传空串表示不改，库里原来的备注留着（GenerateUpsertSQL 生成语句时，值是空串的文本列不进语句）
func (c *SysRoleService) Update(pnctx *pncontext.PnContext, dataForm *models.SysRoleForm) error {
	var err error
	var id int64
	sqlStr := `SELECT id FROM db_sys_role WHERE id=$1;`
	if err = pnctx.DB.Get(&id, sqlStr, dataForm.Id); err != nil {
		return pncontext.NewError(utils.Resp_code_paramParsel20002, "角色不存在")
	}
	// id 存在就更新
	dbParams := util.StructToMap(*dataForm, "db")
	fields := []string{"id", "parent_id", "name", "code", "sort", "status", "remark", "created_at"}
	strSQL := utils.GenerateUpsertSQL("db_sys_role", fields, dbParams,
		[]string{"id"}, []string{"created_at"}, []string{})
	if _, err = pnctx.DB.NamedExec(strSQL, dbParams); err != nil {
		return c.dupErr(err)
	}
	return err
}

// Delete 批量删除角色（ids 为强类型 []int64，安全拼接 SQL）
func (c *SysRoleService) Delete(pnctx *pncontext.PnContext, ids []int64) (err error) {
	if len(ids) == 0 {
		return nil
	}
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = strconv.FormatInt(id, 10)
	}
	sqlStr := fmt.Sprintf("delete from db_sys_role where id in (%s);", strings.Join(idStrs, ","))
	_, err = pnctx.DB.Exec(sqlStr)
	return err
}
