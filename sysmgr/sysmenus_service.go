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
)

type SysMenusService struct {
	pageservice.PageService
}

type SysMenusListRoot struct {
	Data []SysMenusListItem
}

type SysMenusListItem map[string]interface{}
type SysMenusListItemMate map[string]interface{}

// IntoTempMenu 从数据库加载全量菜单数据（按 parent_id / rank 排序，便于后续递归组装）
func (c *SysMenusService) IntoTempMenu(pnctx *pncontext.PnContext, menusData *[]models.SysMenus) (err error) {
	strSQL := "select * from db_sys_menus where id<>4 order By parent_id asc,rank asc"
	err = pnctx.DB.Select(menusData, strSQL)
	if err != nil {
		return
	}
	return err
}

// ReadLevelMenuMem 递归组装菜单树（用于菜单管理列表）
func (c *SysMenusService) ReadLevelMenuMem(pnctx *pncontext.PnContext, menusData []models.SysMenus, JsonItem interface{}, ParentId int64, menuclass string, menutag string, Isfirst bool) {
	if pnctx.DB == nil {
		return
	}
	if menuclass == "" {
		menuclass = "sysmgr"
	}
	for _, v := range menusData {
		if v.ParentId == ParentId {
			mn := SysMenusListItem{}
			mn["key"] = v.Name
			utils.DbStructToMap(mn, &v, "json")

			if Isfirst {
				mn["component"] = ""
			} else {
				mn["component"] = v.Component
			}
			mn["power_count"] = 0
			mn["children"] = []interface{}{}

			c.ReadLevelMenuMem(pnctx, menusData, &mn, v.Id, menuclass, menutag, false)

			LChildren := mn["children"].([]interface{})
			if len(LChildren) == 0 {
				delete(mn, "children")
			} else {
				mn["power_count"] = len(LChildren)
			}

			if Isfirst {
				myJsonItem := JsonItem.(*SysMenusListRoot)
				myJsonItem.Data = append(myJsonItem.Data, mn)
			} else {
				myJsonItem := JsonItem.(*SysMenusListItem)
				var myJsonChildren SysMenusListItem = *myJsonItem
				myJsonChildren["children"] = append(myJsonChildren["children"].([]interface{}), mn)
			}
		}
	}
}

// ReadLevelMenuAuths 读取某节点的直接子权限标识列表（auths）
func (c *SysMenusService) ReadLevelMenuAuths(pnctx *pncontext.PnContext, menusData []models.SysMenus, ParentId int64) []string {
	if pnctx.DB == nil {
		return nil
	}
	AuthsList := []string{}
	for _, v := range menusData {
		if v.ParentId == ParentId {
			Auths := strings.ReplaceAll(string(v.Auths), `"`, ``)
			if Auths != "" {
				AuthsList = append(AuthsList, Auths)
			}
		}
	}
	return AuthsList
}

// ReadLevelMenuMemForMain 递归组装前端路由菜单（仅包含 type 0/1/2 的菜单项，含完整 meta 结构）
// parentRank 为父节点的 effectiveRank（顶层传 0），子节点 effectiveRank = parentRank*1000 + v.Rank，
// 保证扁平化排序后父节点始终在子节点前面，面包屑 getParentPaths DFS 能正确识别层级。
func (c *SysMenusService) ReadLevelMenuMemForMain(pnctx *pncontext.PnContext, menusData []models.SysMenus, JsonItem interface{}, ParentId int64, menuclass string, menutag string, Isfirst bool, parentRank int64) {
	if pnctx.DB == nil {
		return
	}
	if menuclass == "" {
		menuclass = "sysmgr"
	}
	for _, v := range menusData {
		if v.ParentId == ParentId {
			if v.MenuType == 0 || v.MenuType == 1 || v.MenuType == 2 {
				mn := SysMenusListItem{}
				mn["key"] = v.Name
				if Isfirst {
					mn["component"] = ""
				} else {
					mn["component"] = v.Component
				}
				mn["path"] = v.Path
				mn["name"] = v.Name
				mn["redirect"] = v.Redirect
				mn["power_count"] = 0

				// effectiveRank 保证子节点排序值 > 父节点排序值，避免扁平化后子节点排到父节点前面
				effectiveRank := parentRank + 10000 + int64(v.Rank)

				mn["meta"] = SysMenusListItemMate{}
				mnMeta := mn["meta"].(SysMenusListItemMate)
				mnMeta["title"] = v.Title
				mnMeta["icon"] = v.Icon
				mnMeta["extraIcon"] = v.ExtraIcon
				mnMeta["showLink"] = v.ShowLink
				mnMeta["showParent"] = v.ShowParent
				mnMeta["roles"] = []string{"r_Admin"}
				mnMeta["keepAlive"] = v.KeepAlive
				mnMeta["frameSrc"] = v.FrameSrc
				mnMeta["frameLoading"] = v.FrameLoading
				mnMeta["hiddenTag"] = v.HiddenTag
				mnMeta["fixedTag"] = v.FixedTag
				mnMeta["activePath"] = v.ActivePath
				mnMeta["rank"] = effectiveRank
				mnMeta["auths"] = c.ReadLevelMenuAuths(pnctx, menusData, v.Id)

				mn["children"] = []interface{}{}
				c.ReadLevelMenuMemForMain(pnctx, menusData, &mn, v.Id, menuclass, menutag, false, effectiveRank)

				LChildren := mn["children"].([]interface{})
				if len(LChildren) == 0 {
					delete(mn, "children")
				} else {
					mn["power_count"] = len(LChildren)
				}

				if Isfirst {
					myJsonItem := JsonItem.(*SysMenusListRoot)
					myJsonItem.Data = append(myJsonItem.Data, mn)
				} else {
					myJsonItem := JsonItem.(*SysMenusListItem)
					var myJsonChildren SysMenusListItem = *myJsonItem
					myJsonChildren["children"] = append(myJsonChildren["children"].([]interface{}), mn)
				}
			}
		}
	}
}

// Insert 新增菜单
func (c *SysMenusService) Insert(pnctx *pncontext.PnContext, dataForm *models.MenusForm) (err error) {
	sqlStr := `
INSERT INTO db_sys_menus (
  menu_type, parent_id, title, name, path, component, rank, redirect,
  icon, extra_icon, enter_transition, leave_transition, active_path, auths,
  frame_src, frame_loading, keep_alive, hidden_tag, fixed_tag, show_link,
  show_parent, loged, created_at, updated_at
) VALUES (
  :menuType, :parentId, :title, :name, :path, :component, :rank, :redirect,
  :icon, :extraIcon, :enterTransition, :leaveTransition, :activePath, :auths,
  :frameSrc, :frameLoading, :keepAlive, :hiddenTag, :fixedTag, :showLink,
  :showParent, :loged, :createdAt, :updatedAt
);`
	dbParams := util.StructToMap(*dataForm, "json")
	dbParams["auths"] = utils.JsonbParam(dataForm.Auths) // auths 是 jsonb 列，不能直接传 json.RawMessage
	_, err = pnctx.DB.NamedExec(sqlStr, dbParams)
	err = utils.GetVerfication(err)
	return err
}

// Update 修改菜单（使用 GenerateUpsertSQL 自动生成 UPDATE 语句）
func (c *SysMenusService) Update(pnctx *pncontext.PnContext, dataForm *models.MenusForm) (err error) {
	// 先查原记录（用于 menucode 同步等后续扩展）
	var id int64
	if err = pnctx.DB.Get(&id, "select id from db_sys_menus where id=$1", dataForm.Id); err != nil {
		return pncontext.NewError(utils.Resp_code_paramParsel20002, "菜单不存在")
	}
	dbParams := util.StructToMap(*dataForm, "db")
	dbParams["auths"] = utils.JsonbParam(dataForm.Auths) // auths 是 jsonb 列，不能直接传 json.RawMessage
	fields := []string{
		"id", "menu_type", "parent_id", "title", "name", "path", "component",
		"rank", "redirect", "icon", "extra_icon", "enter_transition", "leave_transition",
		"active_path", "auths", "frame_src", "frame_loading", "keep_alive", "hidden_tag",
		"fixed_tag", "show_link", "show_parent", "loged", "created_at", "updated_at",
	}
	strSQL := utils.GenerateUpsertSQL("db_sys_menus", fields, dbParams,
		[]string{"id"}, []string{"created_at"}, []string{})
	if _, err = pnctx.DB.NamedExec(strSQL, dbParams); err != nil {
		return err
	}
	return err
}

// Delete 批量删除菜单（ids 为强类型 []int64，安全拼接 SQL）
func (c *SysMenusService) Delete(pnctx *pncontext.PnContext, ids []int64) (err error) {
	if len(ids) == 0 {
		return nil
	}
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = strconv.FormatInt(id, 10)
	}
	sqlStr := fmt.Sprintf("delete from db_sys_menus where id in (%s);", strings.Join(idStrs, ","))
	_, err = pnctx.DB.Exec(sqlStr)
	return err
}
