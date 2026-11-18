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

// SysAdminService 管理员管理页的服务（管理员表 db_sys_admin）
type SysAdminService struct {
	pageservice.PageService
}

// SysAdminListData 管理员列表分页数据
type SysAdminListData struct {
	List []models.SysAdmin `json:"list"`
	models.PageData
}

// SysAdminListRoot 管理员列表响应根结构
type SysAdminListRoot struct {
	Data SysAdminListData
}

// ReadList 查询用户列表（分页）
func (c *SysAdminService) ReadList(pnctx *pncontext.PnContext, dataRoot *SysAdminListRoot, params models.SysAdminSearchParams) {
	if pnctx.DB == nil {
		return
	}
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

	//超级管理员可见全部用户(含已删除),其他用户仅见未删除
	strWhere := " WHERE deleted_at IS NULL"
	if pnctx.TokenList != nil && pnctx.TokenList.GetUserId() == "superadmin" {
		strWhere = " WHERE 1=1"
	}
	if params.Username != "" {
		dbParams["username"] = utils.EscapeLikeParam(params.Username)
		strWhere += ` AND (username LIKE :username ESCAPE '\' OR real_name LIKE :username ESCAPE '\')`
	}
	if params.Phone != "" {
		dbParams["phone"] = utils.EscapeLikeParam(params.Phone)
		strWhere += ` AND phone LIKE :phone ESCAPE '\'`
	}
	if params.Status != "" {
		dbParams["status"] = params.Status
		strWhere += " AND status=:status"
	}
	if params.DeptId > 0 {
		dbParams["dept_id"] = params.DeptId
		strWhere += " AND dept_id=:dept_id"
	}

	// 查询总数
	strSQL1 := "SELECT COUNT(*) AS totalrecord FROM db_sys_admin" + strWhere
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

	// 查询列表
	strSQL := "SELECT id, user_id, username, real_name, gender, avatar, dept_id, " +
		"roles, phone, email, status, created_at, deleted_at " +
		"FROM db_sys_admin" + strWhere +
		" ORDER BY id DESC LIMIT :pageSize OFFSET :offset"
	rows, err := pnctx.DB.NamedQuery(strSQL, dbParams)
	if err != nil {
		log.Errorln(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var v models.SysAdmin
		if err = rows.StructScan(&v); err != nil {
			log.Errorln(err)
			return
		}
		dataRoot.Data.List = append(dataRoot.Data.List, v)
	}
	dataRoot.Data.PageSize = params.PageSize
	dataRoot.Data.CurrentPage = params.CurrentPage
}

// Insert 新增用户
func (c *SysAdminService) Insert(pnctx *pncontext.PnContext, form *models.SysAdminForm) (err error) {
	// 生成 user_id
	type maxRow struct {
		MaxId int64 `db:"maxid"`
	}
	var mr maxRow
	_ = pnctx.DB.Get(&mr, "SELECT COALESCE(MAX(id),100000) AS maxid FROM db_sys_admin")
	userId := fmt.Sprintf("u%d", mr.MaxId+1)

	// 密码：三重 MD5（默认 userId+"888"）
	pass := form.Password
	if pass == "" {
		pass = userId + "888"
	}
	hashed := util.Md5(util.Md5(util.Md5(pass)))

	roles := []string(form.Roles)
	if roles == nil {
		roles = []string{}
	}

	// 创建时间 created_at 由表上的默认值填
	_, err = pnctx.DB.Exec(
		`INSERT INTO db_sys_admin
		(roles, user_id, username, real_name, gender, password,
		 phone, email, status, avatar, dept_id, created_ip)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		pg.Array(roles), userId, form.Username, form.RealName, form.Gender, hashed,
		form.Phone, form.Email, form.Status, form.Avatar, form.DeptId, pnctx.IP(),
	)
	err = utils.GetVerfication(err)
	return
}

// Update 修改用户（不更改密码）
func (c *SysAdminService) Update(pnctx *pncontext.PnContext, form *models.SysAdminForm) (err error) {
	if form.Id <= 0 {
		return fmt.Errorf("id不能为空")
	}
	roles := []string(form.Roles)
	if roles == nil {
		roles = []string{}
	}
	_, err = pnctx.DB.Exec(
		`UPDATE db_sys_admin SET
		 username=$1, real_name=$2, gender=$3,
		 phone=$4, email=$5, status=$6,
		 roles=$7, avatar=$8, dept_id=$9
		 WHERE id=$10`,
		form.Username, form.RealName, form.Gender,
		form.Phone, form.Email, form.Status,
		pg.Array(roles), form.Avatar, form.DeptId,
		form.Id,
	)
	if err == nil && form.Status != 1 {
		err = c.dropTokens(pnctx, form.Id)
	}
	return
}

// Delete 批量逻辑删除用户（deleted_at 记下删除时间；已经删除的不再改，留着第一次删除的时间）
func (c *SysAdminService) Delete(pnctx *pncontext.PnContext, ids []int64) (err error) {
	if len(ids) == 0 {
		return nil
	}
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = strconv.FormatInt(id, 10)
	}
	_, err = pnctx.DB.Exec(
		fmt.Sprintf("UPDATE db_sys_admin SET deleted_at=CURRENT_TIMESTAMP WHERE deleted_at IS NULL AND id IN (%s)", strings.Join(idStrs, ",")),
	)
	if err == nil {
		err = c.dropTokens(pnctx, ids...)
	}
	return
}

// SwitchState 切换用户启用/停用状态
func (c *SysAdminService) SwitchState(pnctx *pncontext.PnContext, id int64, state int) (err error) {
	_, err = pnctx.DB.Exec("UPDATE db_sys_admin SET status=$1 WHERE id=$2", state, id)
	if err == nil && state != 1 {
		err = c.dropTokens(pnctx, id)
	}
	return
}

// dropTokens 删掉这些管理员的令牌行：他们已经登录的令牌立刻失效。
// 管理员不再是启用状态（status 不是 1）或者被删除之后调：鉴权只查令牌表，不删的话旧令牌能一直用到过期。
// 先改管理员表、再删令牌：这中间来登录的已经被拒了，不会留下新令牌。
func (c *SysAdminService) dropTokens(pnctx *pncontext.PnContext, ids ...int64) error {
	_, err := pnctx.DB.Exec(`DELETE FROM db_sys_tokenlist WHERE user_type=1
		AND user_id IN (SELECT user_id FROM db_sys_admin WHERE id = ANY($1))`, pg.Array(ids)) // user_type 1 = 管理员
	return err
}

// ResetPwd 重置用户的登录密码、二次密码：给了哪个改哪个，空串的不动
// 两个值都由前端完成三重 MD5 后提交（与登录比对、用户管理页重置密码口径一致），此处直接入库不再加密
func (c *SysAdminService) ResetPwd(pnctx *pncontext.PnContext, id int64, newPwd, newPwd2 string) (err error) {
	if newPwd == "" && newPwd2 == "" {
		return fmt.Errorf("密码不能为空")
	}
	_, err = pnctx.DB.Exec(`UPDATE db_sys_admin SET password=COALESCE(NULLIF($1,''),password),
		password2=COALESCE(NULLIF($2,''),password2) WHERE id=$3`, newPwd, newPwd2, id)
	return
}

// SetAvatar 保存 base64 头像
func (c *SysAdminService) SetAvatar(pnctx *pncontext.PnContext, id int64, avatar string) (err error) {
	_, err = pnctx.DB.Exec("UPDATE db_sys_admin SET avatar=$1 WHERE id=$2", avatar, id)
	return
}

// RoleSave 保存用户角色分配（更新 roles TEXT[]）
func (c *SysAdminService) RoleSave(pnctx *pncontext.PnContext, id int64, roles []string) (err error) {
	if roles == nil {
		roles = []string{}
	}
	_, err = pnctx.DB.Exec("UPDATE db_sys_admin SET roles=$1 WHERE id=$2", pg.Array(roles), id)
	return
}

// RoleList 获取启用角色列表（用于前端下拉选择）
// parentId > 0 时只返回指定上级角色的子角色，否则返回全部
func (c *SysAdminService) RoleList(pnctx *pncontext.PnContext, parentId int64) (list []models.SysRoleSimple, err error) {
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

// DeptList 获取全部部门（前端自行构建树结构）
func (c *SysAdminService) DeptList(pnctx *pncontext.PnContext) (list []models.SysDept, err error) {
	list = []models.SysDept{}
	err = pnctx.DB.Select(&list,
		"SELECT id, parent_id, name, sort, status FROM db_sys_dept ORDER BY sort ASC, id ASC")
	return
}
