package core

import (
	"github.com/pony5551/PnWind/core/database"
	"github.com/pony5551/PnWind/core/pnlibs"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"github.com/pony5551/PnWind/core/pntask"
	"github.com/pony5551/PnWind/core/router"
	"github.com/pony5551/PnWind/core/service"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v2"
)

// NewSysConfig 创建一份带默认值的配置。全局的 GConfig 就是这样建出来的；测试时可以自己建一份，不碰全局。
func NewSysConfig() *SysConfig {
	conf := SysConfig{
		ProcessDone:    make(chan bool, 1),
		Conf:           nil,
		DeviceId:       0,
		ServerAddr:     ":3000",
		SysApiEnabled:  true,
		OpenDataDelete: false,
		Debug:          true,
		Fiber: fiber.Config{
			// Fiber v3：Prefork / Network 已从 Config 移到 ListenConfig
			// （EnablePrefork / ListenerNetwork），此处不再设置。
			CaseSensitive: false,
			StrictRouting: false,
			ServerHeader:  "PnWind",
			AppName:       "PnWind v1.0.0",
		},
		IdGen:          nil,
		Tasks:          pntask.NewManager(),
		DbPool:         pnlibs.NewPnObjects[string, *database.DBConn](pnlibs.NewStringHash()),
		PagePool:       pnlibs.NewPnObjects[string, *pnlibs.PnObjects[string, any]](pnlibs.NewStringHash()),
		Routers:        []router.Router{},
		Services:       []service.Service{},
		SysConfigCache: pnlibs.NewPnObjects[string, string](pnlibs.NewStringHash()),
		SysExtension:   pnlibs.NewPnObjects[string, any](pnlibs.NewStringHash()),
	}
	// 雪花 ID 生成器不在这里建，也不用在这里读配置：这个函数在包初始化时就执行，比 main 早。
	// 节点号 DeviceId 要等 main 读完配置才有，生成器由 Boot 第 1 步按它创建
	return &conf
}

// -------------------------------------------------------------------------
// 全局变量
// -------------------------------------------------------------------------

var GConfig = NewSysConfig()

// -------------------------------------------------------------------------
// Boot 服务启动入口
//
// 执行顺序：
//   1. 雪花ID生成器初始化
//   2. 视图引擎初始化（Views 为空时使用默认 html 引擎）
//   3. Fiber 实例创建，注册健康检查，挂基础中间件（请求编号、访问日志、panic 兜底），见 middleware.go
//   4. PreSetup 执行（全局中间件：压缩等）
//   5. 业务路由注册（web / api）
//   6. Setup 执行（静态文件兜底路由；走到这一步的请求不记访问日志）
//   7. 安装服务插件（Services，见 core/service）：按顺序调各个插件的 Setup，插件在里面建好、启动自己。
//      放在开始接收连接之前：页面收到请求时各模块都已经建好；哪个插件出错就不往下走、不监听
//   8. 启动监听（配置里开了 ProxyProtocol 时解析 PROXY 协议头）和性能剖析（配置了 PprofAddr 才开），见 listen.go
//   9. 等服务开始接收连接后，开始任务调度（见 pntask），
//      等退出信号，优雅关闭（每一步都等到做完，不设超时）：
//      等定时任务、常驻协程（服务插件常驻的部分也是常驻协程）全部结束 → exitFn → 停 HTTP、断开数据库 → 停性能剖析
//
// 退出信号到第 9 步才登记：在这之前（服务还在启动）收到退出信号，进程按系统默认立刻结束。
//
// 返回错误是启动失败（第 1、7、8 步，原因已经记进日志），main 据此让进程以非零退出码结束；
// 返回 nil 是服务启动过，收到退出信号、退出流程走完了。
// -------------------------------------------------------------------------

func Boot(c *SysConfig, exitFn func() error) error {
	// 不管从哪一步返回，都先等任务管理器停下，再停雪花 ID 生成器。
	// 正常退出时任务管理器在退出清单里已经停过，这里不用再等；启动失败从中途返回时，
	// Boot 之前就启动的常驻协程、手动触发的任务照样等它们做完（不设超时）
	defer func() {
		c.Tasks.Stop()
		if c.IdGen != nil {
			c.IdGen.Stop()
		}
	}()

	var err error
	// 初始化雪花id生成器
	err = c.initIdGen()
	if err != nil {
		log.Errorln("Boot Error:", err.Error())
		return err
	}

	if c.Fiber.Views == nil {
		log.Println("Setting up default view engine")
		engine := html.New("./views", ".tpl")
		engine.Reload(c.Debug)
		if c.SetupViewEngine != nil {
			c.SetupViewEngine(engine)
		}
		c.Fiber.Views = engine
	}
	// post 数据限制
	c.Fiber.BodyLimit = 500 * 1024 * 1024
	app := fiber.New(c.Fiber)

	// 开启服务器
	// 健康检查：注册在基础中间件的前面，不经过鉴权，不记访问日志
	app.Get("/livez", livez)
	app.Get("/readyz", readyz)
	// 基础中间件：panic 兜底 → 请求编号 → 访问日志 → panic 兜底，顺序不能换（见 middleware.go）
	app.Use(recoverPanic, requestID, accessLog, recoverPanic)
	if c.PreSetup != nil {
		c.PreSetup(app)
	}

	//安装路由
	for _, iRouter := range c.Routers {
		pool := c.GetPagePool(iRouter.GetItem().Name)
		iRouter.Setup(app, pool)
	}
	if c.Setup != nil {
		app.Use(noAccessLog) // 走到这一步的请求没有被任何一组路由接走，是冲着静态文件来的，不记访问日志
		c.Setup(app)
	}
	// 安装服务插件：按 Services 的顺序调各个插件的 Setup，插件在里面建好、启动自己（见 core/service）。
	// 放在开始接收连接之前：页面收到请求时各模块都已经建好。哪个出错就不往下走、不监听，
	// 已经登记的常驻协程由上面的 defer 等它们收尾
	for _, plugin := range c.Services {
		err := plugin.Setup(app)
		if err != nil {
			log.Errorln("Boot Error:", err)
			return err
		}
	}
	// 监听：配置里开了 ProxyProtocol 才解析 PROXY 协议头，而且只采用 ProxyTrusted 里的可信代理发来的（见 listen.go）。
	// 先监听再起协程：端口被占用、ProxyTrusted 写错时直接返回，不让进程带着一个没有 HTTP 的服务空等
	ln, err := listen(c.ServerAddr, c.ProxyProtocol, c.ProxyTrusted)
	if err != nil {
		log.Errorln("Boot Error:", err)
		return err
	}
	// 性能剖析：配置了 PprofAddr 才开，单独一个端口，只能是本机地址（见 listen.go）
	pprof, err := startPprof(c.PprofAddr)
	if err != nil {
		_ = ln.Close()
		log.Errorln("Boot Error:", err)
		return err
	}
	go func() {
		defer ln.Close()
		// 启动 Fiber 服务
		if err := app.Listener(ln); err != nil {
			log.Errorln("app.Listen Error:", err)
		}
	}()
	// 等服务开始接收连接，再去等退出信号：在这之前 Shutdown 关不掉监听（见 readyListener），
	// 退出会变成一边断开数据库、一边还在接收请求
	<-ln.Ready()
	// 服务开始接收连接了，开始按时间表调度定时任务（见 pntask）
	c.Tasks.StartTask()

	// 监听信号 优雅退出http服务
	pnlibs.ExitWatch("core.Boot",
		// 先停任务调度，一直等正在执行的定时任务和常驻协程全部结束（无超时）。这期间 HTTP 还在线
		func() error {
			c.Tasks.Stop()
			return nil
		},
		// 任务都结束了，再做业务自己的清理：它释放的东西不会还有任务在用
		exitFn,
		func() error {
			// 先停 HTTP：不再接收新连接，一直等进行中的请求全部完成（无超时）
			err := app.Shutdown()
			// 再释放对象池和数据库
			GConfig.PagePool.Range(func(key string, pool *pnlibs.PnObjects[string, any]) bool {
				pool.Clear(nil)
				return true
			})
			GConfig.DbPool.Clear(func(value *database.DBConn) error {
				if value != nil {
					log.Printf("断开数据库: [%s],[%s],[%s]", value.DBConf.DBTag, value.DBConf.DBDriver, value.DBConf.DBName)
					return value.DBClose()
				}
				return nil
			})
			return err
		},
		// 最后停性能剖析：等正在进行的剖析做完（无超时）。留到最后，退出等得久的时候还能用它看卡在哪里
		pprof.stop,
	)

	log.Warnln("系统退出.")
	return nil
}
