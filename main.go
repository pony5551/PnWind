package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"
	_ "time/tzdata" // 把时区数据编进程序：精简的容器镜像里没有时区文件，不带的话设了环境变量 TZ 也还是 UTC（clickhouse 的驱动也带了它，这里明写一遍，不靠别人）

	"github.com/pony5551/PnWind/core"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"github.com/pony5551/PnWind/core/pnpartmgr"
	"github.com/pony5551/PnWind/core/pntask"
	"github.com/pony5551/PnWind/core/router"
	authmodels "github.com/pony5551/PnWind/sysauth/models"
	"github.com/pony5551/PnWind/syscore/PnWebSocket"
	sys_middleware "github.com/pony5551/PnWind/syscore/middleware"
	sys_routers "github.com/pony5551/PnWind/syscore/routers"
	"github.com/pony5551/PnWind/syscore/utils"

	// 页面所在的包：它们的 init() 里注册页面
	_ "github.com/pony5551/PnWind/sysauth"
	_ "github.com/pony5551/PnWind/sysmgr"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func main() {
	// 启动失败的出口：下面各处失败时把错误赋给 err 再 return，由这个 defer 给退出码。
	// 它要一直是 main 的第一句：排在它前面的 defer 会被 os.Exit 跳过。
	// 下面的 if 里要写 err = …：写成 err := … 是新建一个 err，这里看到的还是 nil，退出码就成了 0
	var err error
	defer func() {
		if err != nil {
			// 退出码 1：systemd、docker 这些靠退出码判断服务有没有起来、要不要重启
			os.Exit(1)
		}
	}()
	// -------------------------------------------------------------------------
	// 配置文件加载
	// -------------------------------------------------------------------------

	if err = core.GConfig.LoadConfig("./conf/dbconfig.json"); err != nil {
		log.Errorln(err)
		return
	}
	config := core.GConfig.Conf

	// -------------------------------------------------------------------------
	// 全局配置赋值
	// -------------------------------------------------------------------------

	core.GConfig.DeviceId = config.GetInt64("DeviceId")               // 节点号：用来生成雪花 ID，也记进操作日志
	core.GConfig.ServerAddr = config.GetString("ServerAddr")          // 监听地址，例如 :9001
	core.GConfig.ProxyProtocol = config.GetBool("ProxyProtocol")      // PROXY 协议的开关：服务前面有负载均衡时打开，用它取得真实的客户端地址
	core.GConfig.ProxyTrusted = config.GetStringSlice("ProxyTrusted") // 可信代理的 IP 或网段，只采用它们发来的 PROXY 头
	core.GConfig.PprofAddr = config.GetString("PprofAddr")            // 性能剖析（pprof）：配置了才开（例如 127.0.0.1:6060），单独一个端口，只能是本机地址
	core.GConfig.SysApiEnabled = config.GetBool("SysApiEnabled")
	core.GConfig.OpenDataDelete = config.GetBool("OpenDataDelete") // 是否开放删除数据
	core.GConfig.Debug = true                                      // 调试模式：模板文件改了不用重启就生效
	core.GConfig.Fiber.ErrorHandler = utils.ErrorHandler           // 统一的错误出口

	// 日志文件放在哪：不写是可执行文件所在目录下的 logs。目录没有就建；建不出来、没有权限写就不启动
	if err = log.SetDir(config.GetString("LogDir")); err != nil {
		log.Errorln(err)
		return
	}
	// 日志只写文件、不写控制台：标准输出交给 systemd、docker 收集的时候打开，不然同一条日志存两份；不写是两路都写
	log.SetFileOnly(config.GetBool("LogFileOnly"))
	// 日志保留天数：过期的日期目录现在清理一次，之后每换一天清理一次；不写按 365 天
	log.SetKeepDays(config.GetInt("LogKeepDays"))
	// 日志里的时间、日期目录哪一刻换、定时任务的 cron 都按进程的时区。记一行：容器里忘了设环境变量 TZ 的话，这里看得出是 UTC
	log.Println("时区:", time.Now().Format("MST -07:00"))

	if core.GConfig.ProxyProtocol {
		// 本机也算可信代理（IPv4 和 IPv6）
		core.GConfig.ProxyTrusted = append(core.GConfig.ProxyTrusted, "127.0.0.0/8", "::1")
	}

	// 登录令牌的签名密钥：不写在代码里，没有配置或者太短就不启动
	authmodels.SecretKey = config.GetString("JwtSecret")
	if len(authmodels.SecretKey) < 32 {
		err = errors.New("配置文件里的 JwtSecret（登录令牌的签名密钥）没有配置，或者不足 32 个字符")
		log.Errorln(err)
		return
	}

	// -------------------------------------------------------------------------
	// 数据库初始化
	// -------------------------------------------------------------------------

	// 按配置文件里的 dbs 连接各个数据库：有一个写错了或者连不上，就不启动
	if err = core.GConfig.InitDbPool(); err != nil {
		log.Errorln("连接数据库出错:", err)
		return
	}
	// 取主库（db1），配置里没有它就不启动
	DBOne, err := core.GConfig.GetMainDB() // 和上面的 var err 在同一层，这个 := 用的就是它，不是新建
	if err != nil {
		log.Errorln("初始化DBOne错误:", err.Error())
		return
	}

	// -------------------------------------------------------------------------
	// 系统初始化
	// -------------------------------------------------------------------------

	// 从主库的 db_sys_config 表加载系统设置
	err = utils.LoadSysConfig(DBOne, core.GConfig.SysConfigCache)
	if err != nil {
		log.Errorln(err.Error())
		return
	}

	// -------------------------------------------------------------------------
	// 分区表
	// -------------------------------------------------------------------------

	// 按月分区的表，后面几个月的分区要提前建出来，交给分区管理器：启动时检查一次，之后每天 3 点再检查一次。
	// 现在只有演示用的 db_demo_partition（建表语句在 sql/0001_create_tables.sql）；业务上按月分区的表，往这个列表里加表名
	partTables := []string{"db_demo_partition"}
	pm := pnpartmgr.NewPnPartitionManager(DBOne)
	// 过去 1 个月到未来 3 个月。这一次出错只记日志（分区管理器自己记了）、照常启动：分区是提前建的，每天还会再检查
	_ = pm.CheckPartitionsAll(partTables, 3, 1)
	// 每天那一次登记成定时任务：进程退出时等它做完，每一轮的结果任务管理器里有（GetAllTasks）
	if err = core.GConfig.Tasks.AddTask(pntask.NewTask("检查分区", "0 0 3 * * *", func(ctx context.Context) error {
		return pm.CheckPartitionsAll(partTables, 3, 1)
	})); err != nil {
		log.Errorln("登记检查分区的定时任务出错:", err)
		return
	}

	// -------------------------------------------------------------------------
	// 服务
	// -------------------------------------------------------------------------

	// // 服务：登录的核心和两个业务模块，各在 syscore/service 下自己的包里，按这个顺序建好、启动
	// core.GConfig.Services = append(core.GConfig.Services,
	// 	authservice.Plugin{}, // 登录
	// 	devservice.Plugin{},  // 设备管理
	// 	kvmservice.Plugin{},  // KVM管理
	// )

	// -------------------------------------------------------------------------
	// HTTP 中间件配置（PreSetup）
	// 注意：此处注册的中间件和路由在业务路由之前生效
	// 执行顺序：内核的健康检查和基础中间件 → PreSetup → 业务路由(web/api) → Setup
	// -------------------------------------------------------------------------

	core.GConfig.PreSetup = func(app *fiber.App) {
		// 健康检查、请求编号、访问日志、panic 兜底由内核在这之前挂好了（见 core/middleware.go）

		// 压缩
		app.Use(compress.New())

		// 首页（/ 和 /index.html）不让浏览器缓存；
		// 地址里带 .js、.css、.htm 的，把响应的 Content-Type 设成带 charset=utf-8 的
		app.Use(func(ctx fiber.Ctx) error {
			if ctx.Path() == "/" || ctx.Path() == "/index.html" {
				ctx.Response().Header.Add("Cache-Control", "no-cache")
				ctx.Response().Header.Add("Expires", "0")
			}
			pagePath := strings.ToLower(ctx.Path())
			if strings.Contains(pagePath, ".js") {
				ctx.Response().Header.Set("Content-Type", "application/javascript; charset=utf-8")
			} else if strings.Contains(pagePath, ".css") {
				ctx.Response().Header.Set("Content-Type", "text/css; charset=utf-8")
			} else if strings.Contains(pagePath, ".htm") {
				ctx.Response().Header.Set("Content-Type", "text/html; charset=utf-8")
			} else if strings.Contains(pagePath, ".html") {
				ctx.Response().Header.Set("Content-Type", "text/html; charset=utf-8")
			}
			return ctx.Next()
		})

		// WebSocket 聊天（/ws）
		chatCenter := PnWebSocket.NewPnChatCenter()
		chatCenter.UseWebSocket(app)
	}

	// -------------------------------------------------------------------------
	// 业务路由加载
	// -------------------------------------------------------------------------

	// 按配置里的 LoadRuters 建各组路由：每一项是一组的名字和地址前缀（例如 web → /web）
	var LoadRuters []router.RouterItem
	// 这里的 err 是 if 自己的（:=），不动上面那个：这一处只记日志、接着往下走
	if err := config.UnmarshalKey("LoadRuters", &LoadRuters); err != nil {
		log.Errorln("LoadRuters 配置出错:", err.Error())
	}
	for _, v := range LoadRuters {
		LRouter := sys_routers.SysRouter{
			Item: &v,
		}
		// 每一组路由都要登录；不用登录的页面在注册时声明（routers.RegisterPublicController）
		LRouter.Auth = sys_middleware.IsAuthenticated
		core.GConfig.Routers = append(core.GConfig.Routers, &LRouter)
	}

	// -------------------------------------------------------------------------
	// 静态文件配置（Setup）
	// 注意：静态路由注册在业务路由之后，作为 SPA 兜底；
	// 若提前注册，"/" 通配可能拦截 API 请求
	// -------------------------------------------------------------------------

	core.GConfig.Setup = func(app *fiber.App) {
		app.Use("/", static.New("./public"))
		app.Use("/uploads", static.New("./uploads"))
	}

	// -------------------------------------------------------------------------
	// 启动服务
	// -------------------------------------------------------------------------

	// Boot 一直到收到退出信号、退出流程走完才返回 nil；返回错误是启动失败（原因它已经记进日志）
	err = core.Boot(core.GConfig, func() error {
		// 退出时要做的清理放在这里：收到退出信号、定时任务和常驻协程全部结束之后执行。
		// 这时 HTTP 还在线，不要释放请求处理还要用的东西
		return nil
	})
}
