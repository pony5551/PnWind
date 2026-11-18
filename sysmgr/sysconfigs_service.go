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

type SysConfigsService struct {
	pageservice.PageService
}

// SysConfigsListData 列表响应结构
type SysConfigsListData struct {
	List []models.SysConfigItem `json:"list"`
	models.PageData
}

// ReadList 查询配置列表（支持 sort 过滤、分页）
func (c *SysConfigsService) ReadList(pnctx *pncontext.PnContext, aParams models.SysConfigSearchParams) (data SysConfigsListData, err error) {
	data.PageSize = aParams.PageSize
	data.CurrentPage = aParams.CurrentPage
	data.List = []models.SysConfigItem{}

	dbParams := map[string]interface{}{
		"pageSize": aParams.PageSize,
		"offset":   aParams.PageSize * (aParams.CurrentPage - 1),
		"sort":     aParams.Sort,
	}

	where := " WHERE 1=1"
	if aParams.Sort != "" {
		where += " AND a.sort = :sort"
	}

	countSQL := "SELECT count(*) FROM db_sys_config a" + where
	countRows, err := pnctx.DB.NamedQuery(countSQL, dbParams)
	if err != nil {
		return
	}
	defer countRows.Close()
	// 用 for 读到头，不用 if 只读一行：结果集读到头才把连接还回去。只读一行的话连接要占到函数返回，
	// 下面的列表查询得再占一个，一个请求同时占两个连接
	for countRows.Next() {
		var total int64
		if e := countRows.Scan(&total); e == nil {
			data.Total = uint64(total)
		}
	}

	listSQL := "SELECT * FROM db_sys_config a" + where +
		" ORDER BY sort, config_id ASC LIMIT :pageSize OFFSET :offset"
	rows, err := pnctx.DB.NamedQuery(listSQL, dbParams)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var v models.SysConfig
		if e := rows.StructScan(&v); e != nil {
			log.Errorln(e)
			continue
		}
		data.List = append(data.List, models.SysConfigItem{
			ConfigId:  v.ConfigId,
			Sort:      v.Sort,
			Key:       v.Key,
			Value:     v.Value.String,
			Remark:    v.Remark.String,
			Patrn:     v.Patrn.String,
			Expansion: v.Expansion.String,
		})
	}
	return
}

// Insert 新增配置项（sort != 'sys'）
func (c *SysConfigsService) Insert(pnctx *pncontext.PnContext, form *models.SysConfigAddForm) (err error) {
	sqlStr := `INSERT INTO db_sys_config (sort, key, value, remark, patrn, expansion)
               VALUES (:sort, :key, :value, :remark, :patrn, :expansion)`
	_, err = pnctx.DB.NamedExec(sqlStr, map[string]interface{}{
		"sort":      form.Sort,
		"key":       form.Key,
		"value":     form.Value,
		"remark":    form.Remark,
		"patrn":     form.Patrn,
		"expansion": form.Expansion,
	})
	err = utils.GetVerfication(err)
	return
}

// Update 修改配置值（系统项 sort='sys' 也允许修改值）
func (c *SysConfigsService) Update(pnctx *pncontext.PnContext, form *models.SysConfigEditForm) (err error) {
	sqlStr := "UPDATE db_sys_config SET value = :value WHERE config_id = :id"
	_, err = pnctx.DB.NamedExec(sqlStr, map[string]interface{}{
		"value": form.Value,
		"id":    form.ConfigId,
	})
	return
}

// Delete 删除配置项（不允许删除 sort='sys' 的系统项）
func (c *SysConfigsService) Delete(pnctx *pncontext.PnContext, ids []int64) (err error) {
	if len(ids) == 0 {
		return nil
	}
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = strconv.FormatInt(id, 10)
	}
	sqlStr := fmt.Sprintf(
		"DELETE FROM db_sys_config WHERE config_id IN (%s) AND sort <> 'sys'",
		strings.Join(idStrs, ","))
	_, err = pnctx.DB.Exec(sqlStr)
	return
}
