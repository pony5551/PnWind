package sysmgr

import (
	"strconv"
	"strings"
	"time"

	"github.com/pony5551/PnWind/core"
	"github.com/pony5551/PnWind/core/controller"
	"github.com/pony5551/PnWind/core/pncontext"
	"github.com/pony5551/PnWind/syscore/utils"
	"github.com/pony5551/PnWind/sysmgr/models"
)

type SysMenusCont struct {
	controller.ControllerBase
	SVC SysMenusService
}

// parseMenuParams 解析菜单通用查询参数（parentId / menuclass / menutag）
func (c *SysMenusCont) parseMenuParams(pnctx *pncontext.PnContext) (parentId int64, menuclass, menutag string, err error) {
	parentIdStr := pnctx.Query("parentId", "0")
	menuclass = pnctx.Query("menuclass", "")
	menutag = pnctx.Query("menutag", "")
	parentId, err = strconv.ParseInt(parentIdStr, 10, 64)
	return
}

// loadMenuData 从数据库一次性加载全量菜单数据（菜单为树形结构，全量加载后在内存中递归组装）
func (c *SysMenusCont) loadMenuData(pnctx *pncontext.PnContext) (menusData []models.SysMenus, err error) {
	err = c.SVC.IntoTempMenu(pnctx, &menusData)
	return
}

// 获取主体菜单（用于前端路由渲染，含 meta 结构）：只给这个账号有权限的菜单和按钮，见 menuAllow
func (c *SysMenusCont) Page_getmainmenu(pnctx *pncontext.PnContext) error {
	menusData, err := c.loadMenuData(pnctx)
	if err != nil {
		return err
	}
	parentId, menuclass, menutag, err := c.parseMenuParams(pnctx)
	if err != nil {
		return err
	}
	menuRoot := &SysMenusListRoot{Data: []SysMenusListItem{}}
	c.SVC.ReadLevelMenuMemForMain(pnctx, menusData, menuRoot, parentId, menuclass, menutag, true, 0, menuAllow(pnctx))
	return pnctx.RespOK(menuRoot.Data, "")
}

// menuAllow 主菜单按它过滤：这个账号的权限键（登录时由这个账号的各个角色的权限合起来，记在令牌表里）。
// 超级管理员回 nil，表示不过滤：和登录校验一样，不查个人权限。
func menuAllow(pnctx *pncontext.PnContext) map[string]bool {
	token, _ := pnctx.TokenList.(*models.SysTokenList) // 取主菜单要登录，登录校验放进来的就是它
	if token == nil {
		return map[string]bool{}
	}
	if token.UserId == "superadmin" {
		return nil
	}
	allow := make(map[string]bool, len(token.UserMenuSet))
	for _, key := range token.UserMenuSet {
		allow[key] = true
	}
	return allow
}

// 获取菜单列表（用于菜单管理页面，含完整字段）
func (c *SysMenusCont) Page_menulist(pnctx *pncontext.PnContext) error {
	menusData, err := c.loadMenuData(pnctx)
	if err != nil {
		return err
	}
	parentId, menuclass, menutag, err := c.parseMenuParams(pnctx)
	if err != nil {
		return err
	}
	menuRoot := &SysMenusListRoot{Data: []SysMenusListItem{}}
	c.SVC.ReadLevelMenuMem(pnctx, menusData, menuRoot, parentId, menuclass, menutag, true)
	return pnctx.RespOK(menuRoot.Data, "")
}

// Page_list 默认列表（同 Page_menulist）
func (c *SysMenusCont) Page_list(pnctx *pncontext.PnContext) error {
	return c.Page_menulist(pnctx)
}

// 新增菜单
func (c *SysMenusCont) Page_add(pnctx *pncontext.PnContext) error {
	var dataForm models.MenusForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	dataForm.CreatedAt = time.Now()
	dataForm.UpdatedAt = time.Now()
	if err := fixAuths(&dataForm); err != nil {
		return err
	}
	if err := c.SVC.Insert(pnctx, &dataForm); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "新增成功.")
}

// fixAuths 整理表单里的权限标识 auths，新增、修改都先过它：
// 没传、传的是 JSON 的 null，都存成 JSON 的空串 ""（和前端不填时传的一样）：这一列存了 NULL，
// 读菜单时那一行读不出来，所有人的菜单就都取不到了；
// 按钮（MenuType 3）的 name 取自 auths（去掉引号），所以按钮的 auths 不能没有、不能是空串。
func fixAuths(dataForm *models.MenusForm) error {
	if len(dataForm.Auths) == 0 || string(dataForm.Auths) == "null" {
		dataForm.Auths = []byte(`""`)
	}
	if dataForm.MenuType == 3 {
		dataForm.Name = strings.ReplaceAll(string(dataForm.Auths), `"`, ``)
		if strings.TrimSpace(dataForm.Name) == "" {
			return pncontext.NewError(utils.Resp_code_paramParsel20002, "权限标识不能为空")
		}
	}
	return nil
}

// 修改菜单（只改已有的，不会新增）
func (c *SysMenusCont) Page_edit(pnctx *pncontext.PnContext) error {
	var dataForm models.MenusForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	if dataForm.Id <= 0 {
		return pnctx.RespErr(utils.Resp_code_paramParsel20002, "id不能为空")
	}
	dataForm.CreatedAt = time.Now()
	dataForm.UpdatedAt = time.Now()
	if err := fixAuths(&dataForm); err != nil {
		return err
	}
	if err := c.SVC.Update(pnctx, &dataForm); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "修改菜单成功.")
}

// 删除菜单（使用强类型 DataIdsForm，支持批量删除）
func (c *SysMenusCont) Page_del(pnctx *pncontext.PnContext) error {
	if !core.GConfig.OpenDataDelete {
		return utils.RespDeleteDisabled(pnctx)
	}
	var dataForm models.DataIdsForm
	if err := pnctx.Bind(&dataForm); err != nil {
		return err
	}
	if err := c.SVC.Delete(pnctx, dataForm.Ids); err != nil {
		return err
	}
	return pnctx.RespOK(nil, "删除菜单成功.")
}
