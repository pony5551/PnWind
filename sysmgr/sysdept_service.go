package sysmgr

import (
	"fmt"
	"github.com/pony5551/PnWind/core/pncontext"
	"github.com/pony5551/PnWind/core/service"
	"github.com/pony5551/PnWind/core/util"
	"github.com/pony5551/PnWind/syscore/utils"
	"github.com/pony5551/PnWind/sysmgr/models"
	"strconv"
	"strings"
)

// SysDeptService 部门业务服务层
type SysDeptService struct {
	service.Service
}

// ReadAll 查询部门列表（全量，前端自行处理树结构）
// 支持按 name 模糊搜索、按 status 精确过滤；不分页，部门数量有限。
func (c *SysDeptService) ReadAll(pnctx *pncontext.PnContext, list *[]models.SysDept, params models.SysDeptSearchParams) error {
	if pnctx.DB == nil {
		return nil
	}
	dbParams := map[string]interface{}{}
	strWhere := " WHERE 1=1"
	if params.Name != "" {
		dbParams["name"] = utils.EscapeLikeParam(params.Name)
		strWhere += ` AND name LIKE :name ESCAPE '\'`
	}
	if params.Status != "" {
		dbParams["status"] = params.Status
		strWhere += " AND status=:status"
	}
	strSQL := "SELECT * FROM db_sys_dept" + strWhere + " ORDER BY sort ASC, id ASC"

	rows, err := pnctx.DB.NamedQuery(strSQL, dbParams)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v models.SysDept
		if err = rows.StructScan(&v); err != nil {
			return err
		}
		*list = append(*list, v)
	}
	return nil
}

// Insert 新增部门
func (c *SysDeptService) Insert(pnctx *pncontext.PnContext, dataForm *models.SysDeptForm) (err error) {
	sqlStr := `INSERT INTO db_sys_dept
		(parent_id, name, principal, phone, email, sort, status, remark, created_at)
		VALUES (:parent_id, :name, :principal, :phone, :email, :sort, :status, :remark, :created_at)`
	dbParams := util.StructToMap(*dataForm, "db")
	_, err = pnctx.DB.NamedExec(sqlStr, dbParams)
	err = utils.GetVerfication(err)
	return err
}

// Update 修改部门
func (c *SysDeptService) Update(pnctx *pncontext.PnContext, dataForm *models.SysDeptForm) (err error) {
	sqlStr := `UPDATE db_sys_dept SET
		parent_id=:parent_id, name=:name, principal=:principal,
		phone=:phone, email=:email, sort=:sort, status=:status, remark=:remark
		WHERE id=:id`
	dbParams := util.StructToMap(*dataForm, "db")
	_, err = pnctx.DB.NamedExec(sqlStr, dbParams)
	return err
}

// Delete 批量删除部门
func (c *SysDeptService) Delete(pnctx *pncontext.PnContext, ids []int64) (err error) {
	if len(ids) == 0 {
		return nil
	}
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = strconv.FormatInt(id, 10)
	}
	sqlStr := fmt.Sprintf("DELETE FROM db_sys_dept WHERE id IN (%s)", strings.Join(idStrs, ","))
	_, err = pnctx.DB.Exec(sqlStr)
	return err
}
