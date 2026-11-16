package core

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/jmoiron/sqlx"
	"github.com/pony5551/PnWind/core/database"
	"github.com/pony5551/PnWind/core/pnlibs"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"github.com/pony5551/PnWind/core/pntask"
	"github.com/pony5551/PnWind/core/router"
	"github.com/spf13/viper"
)

// -------------------------------------------------------------------------
// SysConfig 全局系统配置结构体
//
// PreSetup 和 Setup 的执行顺序（见 Boot 函数）：
//   PreSetup(中间件) → 业务路由注册(web/api) → Setup(静态文件兜底)
//
// PreSetup：在业务路由注册之前执行，用于注册全局中间件（压缩等）
// Setup  ：在业务路由注册之后执行，用于注册静态文件兜底路由（SPA fallback）
// -------------------------------------------------------------------------

type SysConfig struct {
	ProcessDone     chan bool
	Conf            *viper.Viper
	DeviceId        int64 // 节点号（0 到 1023）：Boot 第 1 步按它创建雪花 ID 生成器
	ServerAddr      string
	ProxyProtocol   bool     // PROXY 协议：服务前面有负载均衡时，用它取得真实的客户端地址,开启后只采用来自 ProxyTrusted 这些网段的 PROXY 头，其他来源的一律忽略，防止伪造来源地址。
	ProxyTrusted    []string // 可信代理的 IP 或网段（例如 10.0.0.0/8）：只采用它们发来的 PROXY 协议头
	PprofAddr       string   // 性能剖析（pprof）的监听地址（例如 127.0.0.1:6060）：空着不开；单独一个端口，只能是本机地址
	SysApiEnabled   bool
	OpenDataDelete  bool
	Debug           bool
	Fiber           fiber.Config
	IdGen           *pnlibs.PnIDGenerator                                     // 雪花 ID 生成器，Boot 第 1 步按 DeviceId 创建，在这之前是 nil
	Tasks           *pntask.Manager                                           // 任务管理：定时任务和常驻的后台协程（见 pntask）；退出时 Boot 一直等它们结束
	DbPool          *pnlibs.PnObjects[string, *database.DBConn]               //数据连接池
	PagePool        *pnlibs.PnObjects[string, *pnlibs.PnObjects[string, any]] // 页面对象池，每组路由一个，key 为路由的名字
	Routers         []router.Router
	SysConfigCache  *pnlibs.PnObjects[string, string] // 系统参数缓存：参数表读进内存的一份，key 为 "分组.参数名"，例如 web.pageSize
	PreSetup        func(app *fiber.App)              // 中间件注册钩子，在业务路由之前执行
	Setup           func(app *fiber.App)              // 兜底路由钩子，在业务路由之后执行
	SetupViewEngine func(e any)
}

// LoadConfig 读取配置文件 path（例如 ./conf/dbconfig.json），放进 c.Conf。
// 相对路径按当前目录找，格式按扩展名认。找不到文件、内容有错都返回错误，由调用方决定不启动。
func (c *SysConfig) LoadConfig(path string) error {
	conf := viper.New()
	conf.SetConfigFile(path)
	if err := conf.ReadInConfig(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("找不到配置文件 %s", path)
		}
		return fmt.Errorf("配置文件 %s 出错: %w", path, err)
	}
	c.Conf = conf
	return nil
}

// initIdGen 按节点号 c.DeviceId 创建雪花 ID 生成器，由 Boot 第 1 步调用：这时 main 已经把配置里的 DeviceId 填进来了。
// c.IdGen 已经有值（调用方自己放了一个）就用它的，不再建。
func (c *SysConfig) initIdGen() error {
	if c.IdGen != nil {
		return nil
	}
	idConfig := pnlibs.DefaultConfig()
	idConfig.NodeID = c.DeviceId
	idGen, err := pnlibs.NewPnIDGenerator(idConfig)
	if err != nil {
		return fmt.Errorf("初始化雪花 ID 生成器失败: %w", err)
	}
	c.IdGen = idGen
	return nil
}

// -------------------------------------------------------------------------
// 数据库相关方法
// -------------------------------------------------------------------------

// 解析数据库配置：dbs 下每一项是一个库，逐个连接，放进 DbPool。
// 写错的、连不上的不在第一个就停，全部看完一起返回：启动一次就知道一共有哪几处要改。
func (c *SysConfig) dbsParser(dbsMap map[string]any) error {
	var errs []error
	for _, k := range slices.Sorted(maps.Keys(dbsMap)) { // 按库的标签排好再连：顺序固定，同一个故障每次的结果一样
		v2, isOk := dbsMap[k].(map[string]any)
		if !isOk {
			errs = append(errs, fmt.Errorf("数据库 [%s] 的配置不是一组键值", k))
			continue
		}
		if _, isOk := v2["dbdriver"]; !isOk { // 没有 DBDriver 的条目不当成库，跳过
			continue
		}
		// 各项按路径从 viper 取，不从 v2 里取：v2 的键全是小写，值的类型跟着配置文件走（JSON 的数字是 float64）；
		// viper 取出来统一成字符串、整数，没写的是空串、0，不会因为少写一项、把端口写成数字就 panic
		get := func(name string) string { return c.Conf.GetString("dbs." + k + "." + name) }
		dbc := database.DBConn{DBConf: database.DBConfig{
			DBTag:      k,
			DBDriver:   get("DBDriver"),
			DBHost:     get("DBHost"),
			DBPort:     get("DBPort"),
			DBUser:     get("DBUser"),
			DBPass:     get("DBPass"),
			DBName:     get("DBName"),
			MaxConnNum: c.Conf.GetInt("dbs." + k + ".maxConnNum"), // 这两项可以不写：0 表示用默认值
			MaxIdleNum: c.Conf.GetInt("dbs." + k + ".maxIdleNum"),
		}}
		var missing []string
		for _, item := range [][2]string{{"DBHost", dbc.DBConf.DBHost}, {"DBPort", dbc.DBConf.DBPort}, {"DBUser", dbc.DBConf.DBUser}, {"DBName", dbc.DBConf.DBName}} {
			if item[1] == "" {
				missing = append(missing, item[0])
			}
		}
		if len(missing) > 0 {
			errs = append(errs, fmt.Errorf("数据库 [%s] 的配置缺少 %s", k, strings.Join(missing, "、")))
			continue
		}

		log.Printf("连接数据库: [%s],[%s],[%s]", k, dbc.DBConf.DBDriver, dbc.DBConf.DBName)
		db, err := dbc.DBOpen()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		dbc.DB = db
		c.DbPool.Store(k, &dbc)
	}
	return errors.Join(errs...)
}

// InitDbPool 按配置文件里的 dbs 连接各个数据库。有一个写错了或者连不上就返回错误（里面列出全部有问题的库），
// 由调用方决定不启动。
func (c *SysConfig) InitDbPool() error {
	return c.dbsParser(c.Conf.GetStringMap("dbs"))
}

func (c *SysConfig) GetMainDB() (*sqlx.DB, error) {
	return c.GetDBByName("db1")
}

func (c *SysConfig) GetDBByName(dbName string) (*sqlx.DB, error) {
	if c.DbPool == nil {
		return nil, errors.New("数据库连接池无效")
	}
	obj, ok := c.DbPool.Load(dbName)
	if !ok || obj == nil {
		return nil, fmt.Errorf("数据库[%s]已释放", dbName)
	}
	db := obj.DB
	return db, nil
}

func (c *SysConfig) GetPagePool(name string) *pnlibs.PnObjects[string, any] {
	if pool, isOk := c.PagePool.Load(name); isOk { // 已经有了就直接给：每个请求都要取，不白建一个
		return pool
	}
	pool := pnlibs.NewPnObjects[string, any](pnlibs.NewStringHash())
	pool, _ = c.PagePool.LoadOrStore(name, pool)
	return pool
}
