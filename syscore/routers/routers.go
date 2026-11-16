package routers

import (
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/pony5551/PnWind/core"
	"github.com/pony5551/PnWind/core/pncontext"
	"github.com/pony5551/PnWind/core/pnlibs"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"github.com/pony5551/PnWind/core/router"
)

// 注册控制器：把控制器 pageObj 扫描成页面（见 page.go），登记到名字是 poolName 的页面池
// （"web"、"api" 等，和 LoadRuters 里的 name 对应），页面路径是 pagePath。
// 控制器写得不对、同一个路径注册了两次，都直接终止启动。
func RegisterController[T any](Conf *core.SysConfig, poolName string, pagePath string, pageObj *T) {
	registerPage(Conf, poolName, pagePath, pageObj, false)
}

// 注册一个不用登录的页面，例如登录页：它所在的路由组要登录时，这个页面也不挂鉴权。其余和 RegisterController 一样。
func RegisterPublicController[T any](Conf *core.SysConfig, poolName string, pagePath string, pageObj *T) {
	registerPage(Conf, poolName, pagePath, pageObj, true)
}

func registerPage[T any](Conf *core.SysConfig, poolName string, pagePath string, pageObj *T, public bool) {
	page, err := NewPage(pageObj)
	if err != nil {
		log.Panicf("RegisterController: %s %v，启动终止", pagePath, err)
	}
	page.Public = public
	if loaded := Conf.GetPagePool(poolName).Store(strings.ToLower(pagePath), page); loaded {
		log.Panicf("RegisterController: %s 重复注册，启动终止", pagePath)
	}
}

type SysRouter struct {
	Item *router.RouterItem
	Auth fiber.Handler // 这一组页面的鉴权中间件，挂在每个页面的路由上；nil 表示这一组不鉴权
}

func (r *SysRouter) GetItem() *router.RouterItem {
	return r.Item
}

func (r *SysRouter) Setup(app *fiber.App, pool *pnlibs.PnObjects[string, any]) {
	log.Printf("SysRouter.Setup: %s，前缀 [%s]，页面 %d 个", r.Item.Name, r.Item.Prefix, pool.Size())
	// 前缀为空时直接挂在 app 上，不使用 app.Group
	var group fiber.Router = app
	if r.Item.Prefix != "" {
		group = app.Group(r.Item.Prefix)
	}
	// 每个页面一行：地址和它的 act。装完路由后按地址排好打印，入口页也回这一份；
	// 用 make 建：没有页面时入口页回的是 []，不是 null
	pages := make([]string, 0, pool.Size())
	pool.Range(func(key string, value any) bool {
		//按照页面路径 安装路由：地址是 <页面路径>/<act>，例如 /sysmgr/sysset/sysadmin/list；act 可以不带
		page := value.(*Page) // 页面池里的都是 RegisterController 放进来的 *Page
		handler := func(ctx fiber.Ctx) error {
			pnCtx, err := pncontext.WithPnContext(ctx)
			if err != nil {
				log.Errorln(err)
				return err
			}
			return page.Dispatch(pnCtx)
		}
		if r.Auth != nil && !page.Public { // 这一组要登录，而且这个页面没有注册成不用登录的
			group.All(key+"/:act?", r.Auth, handler)
		} else {
			group.All(key+"/:act?", handler)
		}
		pages = append(pages, r.Item.Prefix+key+": "+page.String())
		return true
	})
	slices.Sort(pages)
	for _, line := range pages {
		log.Println("  " + line)
	}

	// 入口页 GET <前缀>（例如 /web、/api）：回这一组注册的全部页面，就是上面打印的那几行。
	// 不挂鉴权：它只列地址和 act，访问页面本身照样要登录。
	// 只在有前缀时挂：前缀为空时它就是网站的 "/"，会挡住静态文件的首页
	if r.Item.Prefix != "" {
		group.Get("/", func(c fiber.Ctx) error {
			return c.JSON(pages)
		})
	}
}
