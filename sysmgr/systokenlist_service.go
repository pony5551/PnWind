package sysmgr

import (
	"fmt"
	"github.com/pony5551/PnWind/core/pncontext"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"github.com/pony5551/PnWind/core/service"
	"github.com/pony5551/PnWind/syscore/utils"
	"github.com/pony5551/PnWind/sysmgr/models"
	"strings"
)

// SysTokenListService Token列表业务服务层
type SysTokenListService struct {
	service.Service
}

// SysTokenListListData Token列表分页数据
type SysTokenListListData struct {
	List []SysTokenListListItem `json:"list"`
	models.PageData
}

// SysTokenListListRoot Token列表响应根结构
type SysTokenListListRoot struct {
	Data SysTokenListListData
}

type SysTokenListListItem map[string]interface{}

// ReadList 查询Token列表（分页）
func (c *SysTokenListService) ReadList(pnctx *pncontext.PnContext, JsonItem interface{}, params models.SysTokenListSearchParams) {
	if pnctx.DB == nil {
		return
	}
	dataRoot := JsonItem.(*SysTokenListListRoot)
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

	strWhere := " WHERE (1=1)"
	if params.UserId != "" {
		dbParams["user_id"] = params.UserId
		strWhere += " AND user_id=:user_id"
	}
	if params.UserType > 0 {
		dbParams["user_type"] = params.UserType
		strWhere += " AND user_type=:user_type"
	}
	if params.Ip != "" {
		dbParams["ip"] = "%" + params.Ip + "%"
		strWhere += " AND ip LIKE :ip"
	}
	// 起止时间带着时区（见 SysTokenListSearchParams），和带时区的 created_at 比的是同一个时刻
	if params.StartTime != "" {
		dbParams["starttime"] = params.StartTime
		strWhere += " AND created_at >= :starttime"
	}
	if params.EndTime != "" {
		dbParams["endtime"] = params.EndTime
		strWhere += " AND created_at <= :endtime"
	}

	strSQL1 := "SELECT COUNT(*) AS totalrecord FROM db_sys_tokenlist" + strWhere
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

	strSQL := "SELECT id, token_id, user_type, user_id, ip, created_at, updated_at FROM db_sys_tokenlist" +
		strWhere +
		" ORDER BY id DESC LIMIT :pageSize OFFSET :offset"
	rows, err := pnctx.DB.NamedQuery(strSQL, dbParams)
	if err != nil {
		log.Errorln(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		v := models.SysTokenListRow{}
		if err = rows.StructScan(&v); err != nil {
			log.Errorln(err)
			return
		}
		mn := SysTokenListListItem{}
		mn["key"] = v.Id
		utils.DbStructToMap(mn, &v, "db")
		dataRoot.Data.List = append(dataRoot.Data.List, mn)
	}
}

// Delete 批量强制下线（删除Token记录）
func (c *SysTokenListService) Delete(pnctx *pncontext.PnContext, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = fmt.Sprintf("%d", id)
	}
	sqlStr := fmt.Sprintf("DELETE FROM db_sys_tokenlist WHERE id IN (%s)", strings.Join(idStrs, ","))
	_, err := pnctx.DB.Exec(sqlStr)
	return err
}
