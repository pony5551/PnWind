package sysmgr

import (
	"fmt"
	"github.com/pony5551/PnWind/core/pageservice"
	"github.com/pony5551/PnWind/core/pncontext"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"github.com/pony5551/PnWind/core/util"
	"github.com/pony5551/PnWind/syscore/utils"
	"github.com/pony5551/PnWind/sysmgr/models"
	"strconv"
	"strings"

	pg "github.com/lib/pq"
)

// SysUsersService 用户管理页的服务（用户表 db_sys_users）
type SysUsersService struct {
	pageservice.PageService
}

// SysUsersListData 用户列表分页数据
type SysUsersListData struct {
	List []SysUsersListItem `json:"list"`
	models.PageData
}

// SysUsersListRoot 用户列表响应根结构
type SysUsersListRoot struct {
	Data SysUsersListData
}

type SysUsersListItem map[string]interface{}

// ReadList 查询用户列表（分页，不返回密码字段，不含已删除的）
func (c *SysUsersService) ReadList(pnctx *pncontext.PnContext, JsonItem interface{}, params models.SysUsersSearchParams) {
	if pnctx.DB == nil {
		return
	}
	dataRoot := JsonItem.(*SysUsersListRoot)
	if params.PageSize == 0 {
		params.PageSize = 20
	}
	if params.CurrentPage == 0 {
		params.CurrentPage = 1
	}
	nOffset := (params.PageSize * params.CurrentPage) - params.PageSize
	dbParams := map[string]interface{}{
		"pageSize": params.PageSize,
		"offset":   nOffset,
	}

	strWhere := " WHERE deleted_at IS NULL"
	if params.UserId != "" {
		dbParams["user_id"] = params.UserId
		strWhere += " AND user_id=:user_id"
	}
	if params.Name != "" {
		dbParams["name"] = utils.EscapeLikeParam(params.Name)
		strWhere += ` AND name LIKE :name ESCAPE '\'`
	}
	if params.Email != "" {
		dbParams["email"] = utils.EscapeLikeParam(params.Email)
		strWhere += ` AND email LIKE :email ESCAPE '\'`
	}
	if params.Phone != "" {
		dbParams["phone"] = utils.EscapeLikeParam(params.Phone)
		strWhere += ` AND phone LIKE :phone ESCAPE '\'`
	}
	if params.Type != 0 {
		dbParams["type"] = params.Type
		strWhere += " AND type=:type"
	}
	if params.Status != nil {
		dbParams["status"] = *params.Status
		strWhere += " AND status=:status"
	}

	// 查询总数
	strSQL1 := "SELECT COUNT(*) AS totalrecord FROM db_sys_users" + strWhere
	rows1, err := pnctx.DB.NamedQuery(strSQL1, dbParams)
	if err != nil {
		log.Errorln(err)
		return
	}
	defer rows1.Close()
	for rows1.Next() {
		sl, err := rows1.SliceScan()
		if err != nil {
			continue
		}
		dataRoot.Data.Total = uint64(sl[0].(int64))
	}

	// 查询列表（显式排除密码字段）
	strSQL := `SELECT id, user_id, name, type, status, email, phone, roles,
cash_balance, funds_balance, frozen_balance, config, verified_id, remark,
last_login_ip, last_login_at, created_ip, created_at, updated_at FROM db_sys_users` + strWhere +
		" ORDER BY id DESC LIMIT :pageSize OFFSET :offset"
	rows, err := pnctx.DB.NamedQuery(strSQL, dbParams)
	if err != nil {
		log.Errorln(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		v := models.SysUsers{}
		if err = rows.StructScan(&v); err != nil {
			log.Errorln(err)
			return
		}
		mn := SysUsersListItem{}
		mn["key"] = v.Id
		utils.DbStructToMap(mn, &v, "db")
		dataRoot.Data.List = append(dataRoot.Data.List, mn)
	}
}

// Insert 新增用户。user_id、created_at 由表上的默认值填
func (c *SysUsersService) Insert(pnctx *pncontext.PnContext, dataForm *models.SysUsersAddForm) (err error) {
	if dataForm.Config == "" {
		dataForm.Config = "{}"
	}
	if dataForm.Roles == nil {
		dataForm.Roles = pg.StringArray{}
	}
	sqlStr := `INSERT INTO db_sys_users
(name, type, status, email, phone, roles, password, config, remark, created_ip)
VALUES (:name, :type, :status, :email, :phone, :roles, :password, :config, :remark, :created_ip)`
	dbParams := util.StructToMap(*dataForm, "db")
	dbParams["roles"] = pg.Array([]string(dataForm.Roles))
	dbParams["created_ip"] = pnctx.IP()
	_, err = pnctx.DB.NamedExec(sqlStr, dbParams)
	return c.dupErr(err)
}

// Update 修改用户基本信息（不修改密码、余额和实名信息）
func (c *SysUsersService) Update(pnctx *pncontext.PnContext, dataForm *models.SysUsersEditForm) (err error) {
	if dataForm.Config == "" {
		dataForm.Config = "{}"
	}
	if dataForm.Roles == nil {
		dataForm.Roles = pg.StringArray{}
	}
	sqlStr := `UPDATE db_sys_users SET
name=:name, type=:type, status=:status,
email=:email, phone=:phone, roles=:roles,
config=:config, remark=:remark,
updated_at=CURRENT_TIMESTAMP
WHERE id=:id`
	dbParams := util.StructToMap(*dataForm, "db")
	dbParams["roles"] = pg.Array([]string(dataForm.Roles))
	_, err = pnctx.DB.NamedExec(sqlStr, dbParams)
	return c.dupErr(err)
}

// dupErr 邮箱、手机号和别的用户重复时，把数据库的唯一冲突换成说得出原因的提示（-20002，不记日志）；别的错误原样返回
func (c *SysUsersService) dupErr(err error) error {
	return utils.DupErr(err, map[string]string{
		"db_sys_users_email_key": "邮箱已被其他用户使用",
		"db_sys_users_phone_key": "手机号已被其他用户使用",
	})
}

// RoleList 获取启用角色列表（按 parentId 过滤，parentId>0 时过滤，否则返回全部）
func (c *SysUsersService) RoleList(pnctx *pncontext.PnContext, parentId int64) (list []models.SysRoleSimple, err error) {
	list = []models.SysRoleSimple{}
	if parentId > 0 {
		err = pnctx.DB.Select(&list,
			"SELECT id, name, code FROM db_sys_role WHERE status=1 AND parent_id=$1 ORDER BY sort ASC, id ASC",
			parentId)
	} else {
		err = pnctx.DB.Select(&list,
			"SELECT id, name, code FROM db_sys_role WHERE status=1 ORDER BY sort ASC, id ASC")
	}
	return
}

// ResetPassword 重置登录密码或资金密码
func (c *SysUsersService) ResetPassword(pnctx *pncontext.PnContext, dataForm *models.SysUsersResetPwdForm) (err error) {
	sets := []string{"updated_at=CURRENT_TIMESTAMP"}
	dbParams := map[string]interface{}{"id": dataForm.Id}
	if dataForm.Password != "" {
		sets = append(sets, "password=:password")
		dbParams["password"] = dataForm.Password
	}
	if dataForm.PayPassword != "" {
		sets = append(sets, "pay_password=:pay_password")
		dbParams["pay_password"] = dataForm.PayPassword
	}
	if len(sets) == 1 {
		return nil // 无需更新
	}
	sqlStr := fmt.Sprintf("UPDATE db_sys_users SET %s WHERE id=:id", strings.Join(sets, ", "))
	_, err = pnctx.DB.NamedExec(sqlStr, dbParams)
	return err
}

// Delete 批量软删除用户（deleted_at 记下删除时间，不物理删除；已经删除的不再改，留着第一次删除的时间）
func (c *SysUsersService) Delete(pnctx *pncontext.PnContext, ids []int64) (err error) {
	if len(ids) == 0 {
		return nil
	}
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = strconv.FormatInt(id, 10)
	}
	sqlStr := fmt.Sprintf("UPDATE db_sys_users SET deleted_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE deleted_at IS NULL AND id IN (%s)", strings.Join(idStrs, ","))
	_, err = pnctx.DB.Exec(sqlStr)
	return err
}
