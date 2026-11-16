package models

import (
	"encoding/json"
	"time"
)

// SysMenus 系统菜单路由配置表 (public.db_sys_menus)
type SysMenus struct {
	Id              int64           `db:"id" json:"id"`                            // 主键 ID
	MenuType        int16           `db:"menu_type" json:"menuType"`               // 菜单类型：0=菜单，1=iframe，2=外链，3=按钮
	ParentId        int64           `db:"parent_id" json:"parentId"`               // 父级菜单 ID，0 为根节点
	Title           string          `db:"title" json:"title"`                      // 菜单名称（兼容国际化，若为 key 需在 locales 中配置）
	Name            string          `db:"name" json:"name"`                        // 路由名称（必须唯一，需与组件 defineOptions name 一致）
	Path            string          `db:"path" json:"path"`                        // 路由路径
	Component       string          `db:"component" json:"component"`              // 组件路径（不传时默认同 path）
	Rank            int             `db:"rank" json:"rank"`                        // 菜单排序（home 路由为 0，其他建议从 1 开始）
	Redirect        string          `db:"redirect" json:"redirect"`                // 路由重定向地址
	Icon            string          `db:"icon" json:"icon"`                        // 菜单图标
	ExtraIcon       string          `db:"extra_icon" json:"extraIcon"`             // 右侧额外图标
	EnterTransition string          `db:"enter_transition" json:"enterTransition"` // 进场动画（页面加载动画）
	LeaveTransition string          `db:"leave_transition" json:"leaveTransition"` // 离场动画（页面加载动画）
	ActivePath      string          `db:"active_path" json:"activePath"`           // 指定激活菜单的 path（解决传参路由高亮问题）
	Auths           json.RawMessage `db:"auths" json:"auths"`                      // 权限标识集合（按钮级别权限，JSON 数组格式）
	FrameSrc        string          `db:"frame_src" json:"frameSrc"`               // iframe 内嵌链接地址
	FrameLoading    bool            `db:"frame_loading" json:"frameLoading"`       // 内嵌 iframe 是否开启首次加载动画
	KeepAlive       bool            `db:"keep_alive" json:"keepAlive"`             // 是否缓存该路由页面
	HiddenTag       bool            `db:"hidden_tag" json:"hiddenTag"`             // 是否禁止添加到标签页
	FixedTag        bool            `db:"fixed_tag" json:"fixedTag"`               // 是否固定显示在标签页且不可关闭
	ShowLink        bool            `db:"show_link" json:"showLink"`               // 是否显示该菜单
	ShowParent      bool            `db:"show_parent" json:"showParent"`           // 是否显示父级菜单（针对只有一个子菜单的父级）
	Loged           bool            `db:"loged" json:"loged"`                      // 是否记录操作日志（仅按钮类型有效）
	CreatedAt       time.Time       `db:"created_at" json:"createdAt"`             // 创建时间
	UpdatedAt       time.Time       `db:"updated_at" json:"updatedAt"`             // 更新时间
}

type MenusForm struct {
	Id                int64           `db:"id" json:"id"` // 修改时必填，由控制器检查
	MenuType          int             `db:"menu_type" json:"menuType"        validate:"oneof=0 1 2 3" label:"菜单类型"`
	HigherMenuOptions []any           `db:"higher_menu_options" json:"higherMenuOptions"`
	ParentId          int             `db:"parent_id" json:"parentId"`
	Title             string          `db:"title" json:"title"           validate:"required" label:"菜单名称" trim:"true"`
	Name              string          `db:"name" json:"name"            validate:"required_unless=MenuType 3" label:"路由名称" trim:"true"`
	Path              string          `db:"path" json:"path"            validate:"required_unless=MenuType 3" label:"路由路径" trim:"true"`
	Component         string          `db:"component" json:"component"       validate:"omitempty"`
	Rank              int             `db:"rank" json:"rank"            validate:"gte=0" label:"菜单排序"`
	Redirect          string          `db:"redirect" json:"redirect"        validate:"omitempty"`
	Icon              string          `db:"icon" json:"icon"            validate:"omitempty"`
	ExtraIcon         string          `db:"extra_icon" json:"extraIcon"       validate:"omitempty"`
	EnterTransition   string          `db:"enter_transition" json:"enterTransition" validate:"omitempty"`
	LeaveTransition   string          `db:"leave_transition" json:"leaveTransition" validate:"omitempty"`
	ActivePath        string          `db:"active_path" json:"activePath"      validate:"omitempty"`
	Auths             json.RawMessage `db:"auths" json:"auths"           validate:"omitempty"`
	FrameSrc          string          `db:"frame_src" json:"frameSrc"        validate:"omitempty,url" label:"链接地址"`
	FrameLoading      bool            `db:"frame_loading" json:"frameLoading"`
	KeepAlive         bool            `db:"keep_alive" json:"keepAlive"`
	HiddenTag         bool            `db:"hidden_tag" json:"hiddenTag"`
	FixedTag          bool            `db:"fixed_tag" json:"fixedTag"`
	ShowLink          bool            `db:"show_link" json:"showLink"`
	ShowParent        bool            `db:"show_parent" json:"showParent"`
	Loged             bool            `db:"loged" json:"loged"`
	CreatedAt         time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt         time.Time       `db:"updated_at" json:"updatedAt"`
}

type SysMenusSearchParams struct {
	Current   uint64 `json:"current" default:"1"`
	PageSize  uint64 `json:"pageSize" default:"100"`
	Cateid    uint64 `json:"cateid" default:"0"`
	MenuClass string `json:"menuclass" default:"0"`
	SKey      string `json:"skey"`
}
