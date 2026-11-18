package sysmgr

import (
	"fmt"
	"github.com/pony5551/PnWind/core/pageservice"
	"github.com/pony5551/PnWind/core/pncontext"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"github.com/pony5551/PnWind/syscore/utils"
	"github.com/pony5551/PnWind/sysmgr/models"
	"strconv"
	"strings"
)

// SysLogOptService 操作日志业务服务层
type SysLogOptService struct {
	pageservice.PageService
}

// SysLogOptListData 操作日志列表分页数据
type SysLogOptListData struct {
	List []SysLogOptListItem `json:"list"`
	models.PageData
}

// SysLogOptListRoot 操作日志列表响应根结构
type SysLogOptListRoot struct {
	Data SysLogOptListData
}

type SysLogOptListItem map[string]interface{}

// ReadList 查询操作日志列表（分页）
func (c *SysLogOptService) ReadList(pnctx *pncontext.PnContext, JsonItem interface{}, params models.SysLogOptSearchParams) {
	if pnctx.DB == nil {
		return
	}
	dataRoot := JsonItem.(*SysLogOptListRoot)
	if params.PageSize == 0 {
		params.PageSize = 100
	}
	if params.CurrentPage == 0 {
		params.CurrentPage = 1
	}
	nOffset := (params.PageSize * params.CurrentPage) - params.PageSize
	dbParams := map[string]interface{}{
		"pageSize": params.PageSize,
		"offset":   nOffset,
	}

	strWhere := " WHERE 1=1"
	if params.UserType != 0 {
		dbParams["user_type"] = params.UserType
		strWhere += " AND user_type=:user_type"
	}
	if params.UserId != "" {
		dbParams["user_id"] = params.UserId
		strWhere += " AND user_id=:user_id"
	}
	if params.Type != "" {
		dbParams["type"] = params.Type
		strWhere += " AND type=:type"
	}
	if params.SKey != "" {
		dbParams["skey"] = utils.EscapeLikeParam(params.SKey)
		strWhere += ` AND (events LIKE :skey ESCAPE '\' OR page_path LIKE :skey ESCAPE '\' OR page_query LIKE :skey ESCAPE '\' OR page_form LIKE :skey ESCAPE '\' OR ip LIKE :skey ESCAPE '\')`
	}
	// 起止时间带着时区（见 SysLogOptSearchParams），和带时区的 created_at 比的是同一个时刻
	if params.StartTime != "" {
		dbParams["starttime"] = params.StartTime
		strWhere += " AND created_at>=:starttime"
	}
	if params.EndTime != "" {
		dbParams["endtime"] = params.EndTime
		strWhere += " AND created_at<=:endtime"
	}

	// 查询总数
	strSQL1 := "SELECT COUNT(*) AS totalrecord FROM db_sys_logopt" + strWhere
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
	strSQL := `SELECT id, device_id, user_type, user_id, type, events, details,
page_path, page_query, page_form, ip, created_at FROM db_sys_logopt` + strWhere +
		" ORDER BY id DESC LIMIT :pageSize OFFSET :offset"
	rows, err := pnctx.DB.NamedQuery(strSQL, dbParams)
	if err != nil {
		log.Errorln(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		v := models.SysLogOpt{}
		if err = rows.StructScan(&v); err != nil {
			log.Errorln(err)
			return
		}
		mn := SysLogOptListItem{}
		mn["key"] = v.Id
		utils.DbStructToMap(mn, &v, "db")
		dataRoot.Data.List = append(dataRoot.Data.List, mn)
	}
}

// Delete 批量物理删除操作日志
func (c *SysLogOptService) Delete(pnctx *pncontext.PnContext, ids []int64) (err error) {
	if len(ids) == 0 {
		return nil
	}
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = strconv.FormatInt(id, 10)
	}
	sqlStr := fmt.Sprintf("DELETE FROM db_sys_logopt WHERE id IN (%s)", strings.Join(idStrs, ","))
	_, err = pnctx.DB.Exec(sqlStr)
	return err
}

// Clear 清空操作日志（TRUNCATE，仅限超级管理员调用）
func (c *SysLogOptService) Clear(pnctx *pncontext.PnContext) (err error) {
	_, err = pnctx.DB.Exec(`TRUNCATE TABLE "db_sys_logopt"`)
	return err
}
