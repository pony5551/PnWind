# core 包文档

## 概述

`core` 包是 PnWind 的框架核心层，负责：

- 全局配置管理（`SysConfig` / `GConfig`）
- 数据库连接池初始化与管理
- Fiber HTTP 服务启动与生命周期管理
- 基础中间件（请求编号、访问日志、panic 兜底）和健康检查
- 监听（可选的 PROXY 协议）和性能剖析（pprof）
- 雪花ID生成器初始化
- 任务管理：定时任务和常驻的后台协程
- 服务插件：随进程启动和退出的独立功能（`service.Service`，在 `core/service`），见"服务插件"
- 优雅退出机制

内核只做基础设施，不知道业务的约定：页面的注册和分发、鉴权、操作日志都在 `syscore`。

业务代码不应直接依赖本包内部实现，应通过 `core.GConfig` 访问全局资源，通过 `core.Boot()` 启动服务。

---

## 文件结构

```
core/
├── 00_init.go         包初始化：初始化日志
├── core.go            NewSysConfig、GConfig、Boot
├── config.go          SysConfig 结构体和它的方法（读配置文件、数据库连接池、页面对象池）
├── middleware.go      基础中间件（请求编号、访问日志、panic 兜底）和健康检查
├── listen.go          监听（PROXY 协议）和性能剖析（pprof）
├── router/            路由接口定义（Router、RouterItem）
├── pncontext/         请求上下文 PnContext（Bind、RespOK、RespErr、Tx、业务错误 Error）
├── validation/        请求参数的解析和校验（Bind）；govalidator/ 是字符串校验函数库
├── controller/        控制器基类（ControllerBase）
├── pageservice/       页面服务的基类（PageService）：每个页面的 XxxService 内嵌它
├── service/           服务插件的接口（Service），见"服务插件"
├── database/          数据库连接封装（DBConn、DBConfig），支持 PostgreSQL、ClickHouse
├── pnlibs/            通用工具库（PnObjects、PnIDGenerator、PnQueue、ExitWatch 等）
├── pnlogrus/          日志（log.Println、log.Errorf 这一套函数，底层是 zap），见"日志"
├── pnpartmgr/         分区表管理器（PnPartitionManager），见"分区表管理"
├── pntask/            任务管理：定时任务、常驻的后台协程（Manager、NewTask）
├── pnamqp/            RabbitMQ 客户端（PnMQClient）
├── util/              通用函数（JSON、压缩、MD5、HTTP 请求等）
└── CORE.md            本文档
```

---

## SysConfig 结构体

```go
type SysConfig struct { ... }
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `ProcessDone` | `chan bool` | 进程退出信号通道（现在没有代码用它）|
| `Conf` | `*viper.Viper` | 配置文件实例（dbconfig.json），由 `LoadConfig` 读进来 |
| `DeviceId` | `int64` | 节点号（0 到 1023）：`Boot` 第 1 步按它创建雪花ID生成器，也记进操作日志 |
| `ServerAddr` | `string` | HTTP 监听地址，例如 `:9001` |
| `ProxyProtocol` | `bool` | 是否解析 PROXY 协议头，见"PROXY 协议说明" |
| `ProxyTrusted` | `[]string` | 可信代理的 IP 或网段，只采用它们发来的 PROXY 头 |
| `PprofAddr` | `string` | 性能剖析的监听地址，空着不开，见"性能剖析（pprof）说明" |
| `SysApiEnabled` | `bool` | 是否启用系统内置 API（现在没有代码读它）|
| `OpenDataDelete` | `bool` | 是否允许数据删除操作 |
| `Debug` | `bool` | 调试模式，影响模板热重载 |
| `Fiber` | `fiber.Config` | Fiber 框架配置 |
| `IdGen` | `*PnIDGenerator` | 雪花ID生成器，`Boot` 第 1 步创建，在这之前是 nil |
| `Tasks` | `*pntask.Manager` | 任务管理器：定时任务和常驻的后台协程，见"任务管理" |
| `DbPool` | `*PnObjects[string, *database.DBConn]` | 数据库连接池，key 为数据库标签（如 "db1"）|
| `PagePool` | `*PnObjects[string, *PnObjects[string, any]]` | 页面对象池，每组路由一个，key 为这一组的名字（如 "web"）|
| `Routers` | `[]router.Router` | 各组路由，`Boot` 按顺序逐组安装 |
| `Services` | `[]service.Service` | 服务插件，服务开始接收连接后 `Boot` 按顺序调它们的 `Setup`，见"服务插件" |
| `SysConfigCache` | `*PnObjects[string, string]` | 系统参数缓存：参数表读进内存的一份，key 为 "分组.参数名"，值是参数值。由 `syscore/utils` 的 `LoadSysConfig` 在启动时和点"加载设置"时填；现在还没有代码从里面读 |
| `PreSetup` | `func(*fiber.App)` | 中间件注册钩子，在业务路由之前执行 |
| `Setup` | `func(*fiber.App)` | 兜底路由钩子，在业务路由之后执行 |
| `SetupViewEngine` | `func(any)` | 视图引擎自定义配置钩子 |

默认值（`NewSysConfig()` 给的）：`ServerAddr` 是 `:3000`，`SysApiEnabled`、`Debug` 是 `true`，其余是零值。`main.go` 读完配置文件之后，把 `DeviceId`、`ServerAddr`、`ProxyProtocol`、`ProxyTrusted`、`PprofAddr`、`SysApiEnabled`、`OpenDataDelete` 都换成配置里的值，配置里不写就成了零值，所以这几项以配置文件为准。`ServerAddr` 一定要写：不写是空串，服务会监听在一个随机端口上。

方法（都在 `config.go`）：

| 方法 | 说明 |
|------|------|
| `LoadConfig(path)` | 读配置文件，见"配置文件" |
| `InitDbPool()` | 按配置连接各个数据库，有一个写错了或者连不上就返回错误，见"数据库连接池" |
| `GetMainDB()` / `GetDBByName(name)` | 取主库 / 取指定的库 |
| `GetPagePool(name)` | 取一组路由的页面对象池，没有就建一个 |

---

## 全局变量

```go
var GConfig = NewSysConfig()  // *SysConfig，全局唯一配置实例：main.go 往里填值后传入 Boot()
```

`NewSysConfig()` 返回一份带默认值的配置，各个对象池和任务管理器也在这里建好。雪花ID生成器不再是单独的全局变量，用 `GConfig.IdGen`；它要等 `Boot` 第 1 步才创建，因为节点号 `DeviceId` 是 `main.go` 读完配置之后才填进来的。

---

## PreSetup 与 Setup 的设计说明

`Boot()` 内部的执行顺序形成**三明治结构**：

```
健康检查、基础中间件     ← 内核自己挂：/livez、/readyz、请求编号、访问日志、panic 兜底
    ↓
PreSetup(app)           ← 中间件层（压缩、WebSocket 等）
    ↓
业务路由注册             ← 按 Routers 的顺序逐组安装（web、api 等）
    ↓
Setup(app)              ← 兜底静态文件层（前端页面在这里）
```

**为什么不能合并？**

`Setup` 中的 `app.Use("/", static.New("./public"))` 是通配的兜底路由：前面各组路由都没有接走的请求，到 `public` 目录里找文件，找不到回 404。
若提前注册到 `PreSetup`，每个 GET 请求都要先到 `public` 目录里找一遍文件，地址碰巧和某个文件同名的接口还会被静态文件接走。

**各自职责边界：**

- `PreSetup`：只放全局中间件（`app.Use(...)`）及需要在业务路由之前生效的特殊路由
- `Setup`：只放静态文件或其他需要在业务路由之后兜底的路由
- 要对业务请求生效的中间件**不要**放在 `Setup` 里：它排在业务路由后面，被业务路由接走的请求走不到它
- 鉴权不在这两个钩子里：它挂在每组路由的每个页面上（`syscore/routers` 的 `SysRouter.Auth`）

服务插件不在这里面：它的 `Setup` 在服务开始接收连接之后才调，那时往 `app` 上挂路由不生效（见"服务插件"）。

---

## Boot() 启动流程

```
Boot(c *SysConfig, exitFn func() error) error
│
├── 1. 按 c.DeviceId 创建雪花ID生成器（c.IdGen）
│       └── DeviceId 不在 0～1023 之内时 Boot 直接返回错误，不启动
│
├── 2. 初始化视图引擎
│       └── 若 c.Fiber.Views == nil，使用默认 html 引擎（./views/*.tpl），
│           Debug 为 true 时模板热重载；有 SetupViewEngine 就把引擎交给它再配置
│
├── 3. 创建 Fiber 实例（BodyLimit = 500MB）
│       ├── 注册健康检查：/livez、/readyz（在基础中间件的前面，不经过它们）
│       └── 挂基础中间件：panic 兜底 → 请求编号 → 访问日志 → panic 兜底（middleware.go）
│
├── 4. 执行 PreSetup（若不为 nil）
│       └── main.go 里注册的是：compress、首页不缓存和静态文件的 Content-Type、WebSocket（/ws）
│
├── 5. 注册业务路由
│       └── 按 Routers 的顺序，每一组：Setup(app, GetPagePool(这一组的名字))
│           现在的实现是 syscore/routers 的 SysRouter，
│           组名和地址前缀来自配置的 LoadRuters（例如 web → /web、api → /api）
│
├── 6. 执行 Setup（若不为 nil）
│       └── 注册静态文件路由：/（public目录）、/uploads；走到这一步的请求不记访问日志
│
├── 7. 启动监听和性能剖析（listen.go）
│       ├── net.Listen("tcp", ServerAddr)：端口被占用时 Boot 直接返回错误，不启动
│       ├── 配置里开了 ProxyProtocol 时包装为 proxyproto.Listener（见下面"PROXY 协议说明"）
│       ├── 配置了 PprofAddr 时启动性能剖析服务（见下面"性能剖析（pprof）说明"）
│       └── goroutine 里 app.Listener(ln)，Boot 等它开始接收连接，然后开始任务调度（Tasks.StartTask），
│           再按 Services 的顺序调每个插件的 Setup(app)（见"服务插件"）
│
└── 8. ExitWatch 登记退出信号并阻塞等待，收到后依次执行（每一步都等到做完，不设超时）
        ├── Tasks.Stop()：不再调度，等正在执行的定时任务和常驻协程（服务插件登记的也在内）全部结束（这期间 HTTP 还在线）
        ├── exitFn（业务自己的清理；这时任务都已经结束，HTTP 还在线）
        ├── app.Shutdown()：不再接收新连接，等进行中的请求全部完成
        ├── 清空各页面对象池，断开所有数据库连接
        └── 停止性能剖析服务：等正在进行的剖析做完
```

第 8 步要等服务开始接收连接之后才开始：在那之前 `app.Shutdown()` 关不掉监听。退出信号也是到这一步才登记（在 `pnlibs.ExitWatch` 里）：在这之前——读配置、连库、装路由、打开端口、调服务插件的 Setup——收到退出信号，按系统默认处理，进程立刻结束，不走退出流程。这段时间要它退出的只能是人或者部署工具，直接退出就是对的。从这一步起收到的退出信号才走上面的退出流程。

等待期间再来的退出信号只记一条日志，不打断正在做的清理。

退出信号是 SIGINT（Ctrl+C）、SIGTERM（`kill <进程号>`、`systemctl stop`、`docker stop`）、SIGQUIT 这三个；Windows 上是 Ctrl+C、Ctrl+Break 和关闭控制台窗口。SIGHUP 不算退出信号，被忽略：终端断开（ssh 会话结束、关掉终端窗口）、有人 `kill -HUP` 时服务照常运行、照常应答，要停它得发上面三个之一。终端断开之后控制台那一路写不出去了，日志文件照常写。这说的是标准输出接着终端的情况；接着管道时另有一条限制，见"注意事项"里 SIGPIPE 那一条。

退出码：`Boot` 启动失败（第 1 步、第 7 步）返回错误（原因它自己已经记进日志），服务启动过、收到退出信号走完退出流程返回 nil。`main.go` 的第一句是 `var err error` 和一个 `defer`：`main` 返回时 `err` 不是 nil，就 `os.Exit(1)`。各处启动失败把错误赋给 `err`、记一条日志、`return`，`Boot` 的返回值也赋给它。这个 `defer` 排在最前面、最后才执行，以后加在它后面的 `defer` 都先跑完。`main` 里的 `if` 要写 `err = …`：写成 `err := …` 是新建一个 `err`，外面那个还是 nil，退出码就成了 0，编译器不报。所以启动失败时进程的退出码是 1，正常退出是 0；systemd 的 `Restart=on-failure`、docker 的重启策略靠它判断服务有没有起来。启动过程中被退出信号直接结束的，退出码由系统给，不是 0。退出流程里某一步出错，退出码还是 0：`exitFn`、`app.Shutdown()` 返回的错误记一条日志；断开数据库出的错由对象池用 `fmt.Println` 打到标准输出，不进日志文件。

启动失败（第 1 步、第 7 步直接返回）时，`Boot` 返回之前同样等任务管理器停下：`Boot` 之前就启动的常驻协程、手动触发的任务照样等它们做完。还没走到 `Boot` 的那几处（配置读不到、日志目录用不了、`JwtSecret` 不合格、连不上库、主库取不到、系统参数读不出、检查分区的定时任务登记不上）是记日志后直接返回、以退出码 1 结束，不等任务。所以在 `main.go` 里启动常驻协程，要写在这些检查之后、紧挨着 `core.Boot` 之前。

---

## 基础中间件

`Boot` 建好 Fiber 实例之后、`PreSetup` 之前自己挂上（代码在 `middleware.go`），业务代码不用再挂。顺序不能换：

| 顺序 | 中间件 | 作用 |
|------|--------|------|
| 1 | panic 兜底（外面一层） | 兜的是下面两层，以及访问日志里调用的错误出口：它们 panic 时进程也不退出 |
| 2 | 请求编号 | 每个请求一个 16 位十六进制的编号，写进响应头 `X-Request-ID`，用 `ctx.RequestID()` 取。客户端可以自带（请求头 `X-Request-ID`），方便把前端、网关和这里的日志串起来；但只采用 1 到 64 个字符、只含字母、数字、下划线、短横线的，其余的重新生成 |
| 3 | 访问日志 | 每个请求一行：方法、地址、状态码、耗时、来源地址、请求编号。处理函数 `return` 的错误在这里交给统一的错误出口（`Fiber.ErrorHandler`）转成响应。落到 `Setup`（静态文件）的请求不记 |
| 4 | panic 兜底（里面一层） | 处理函数 panic 时记下 panic 和堆栈（带请求编号），转成普通的错误交给上一步。它在访问日志的里面，panic 的请求才有访问日志和最终的状态码 |

---

## 健康检查

注册在基础中间件的前面：不经过鉴权，不记访问日志，响应里也没有 `X-Request-ID`。

- `GET /livez`：进程在就返回 200 `ok`，给容器、守护进程探活用。
- `GET /readyz`：正常时返回 200 `ok`；收到退出信号之后返回 503 `stopping`，负载均衡据此不再转发新请求（等任务结束和 `exitFn` 执行期间 HTTP 还在线，这段时间看得到 503）。

---

## 配置文件

```go
if err = core.GConfig.LoadConfig("./conf/dbconfig.json"); err != nil {
    log.Errorln(err)
    return
}
```

读取配置文件，放进 `GConfig.Conf`（viper 实例），之后用 `GConfig.Conf.GetString("键名")` 这样取值（代码在 `config.go`），键名不分大小写。相对路径按当前目录找，格式按扩展名认。找不到文件、内容有错时返回错误，里面带着文件路径和原因，`GConfig.Conf` 保持原样。

`main.go` 从里面取 `DeviceId`、`ServerAddr`、`ProxyProtocol`、`ProxyTrusted`、`PprofAddr`、`SysApiEnabled`、`OpenDataDelete`、`LogDir`、`LogFileOnly`、`LogKeepDays`、`JwtSecret`、`LoadRuters`；内核自己只读 `dbs`。

---

## 数据库连接池

### 初始化

```go
if err := core.GConfig.InitDbPool(); err != nil {
    log.Errorln("连接数据库出错:", err)
    return
}
```

从配置文件 `dbs` 节点读取所有数据库配置，按标签的顺序逐个建立连接并存入 `DbPool`。写错的、连不上的不在第一个就停，全部看完之后一起返回，错误里列出每一个有问题的库；`main.go` 见到错误就不启动。

配置文件示例（dbconfig.json）：
```json
{
  "dbs": {
    "db1": {
      "DBDriver": "postgres",
      "DBHost": "127.0.0.1",
      "DBPort": "5432",
      "DBUser": "xxx",
      "DBPass": "xxx",
      "DBName": "xxx",
      "maxConnNum": 50,
      "maxIdleNum": 10
    }
  }
}
```

- `DBDriver` 支持 `postgres`、`clickhouse`，写成别的不启动。没有 `DBDriver` 的条目不当成库，整个跳过。
- `DBHost`、`DBPort`、`DBUser`、`DBName` 必须写，少了不启动，日志里写明哪个库缺哪几项；`DBPass` 可以是空的。`DBPort` 写成字符串、数字都行。
- `maxConnNum`（最大连接数）、`maxIdleNum`（最大空闲连接数）可以不写，默认 50 和 10。最大连接数要比数据库允许的连接数小（PostgreSQL 的 `max_connections` 默认 100，各个库、各个节点、查库的工具一起算）：超过了数据库直接拒绝新连接，请求报错；比它小，就是在连接池里排队。连接最长存活 30 分钟、最长空闲 10 分钟是固定的，没有配置项。
- `db1` 是主库。请求上下文里 `pnctx.DB` 是 `db1`，`pnctx.DB2` 是 `db2`（配置了才有），其余的用 `pnctx.DBByName("库名")` 取。

### 连接串

连接串由 `database/database.go` 拼，用户名、口令这些值都做了转义，里面有空格、引号、反斜杠、`@`、`:`、`/` 都不会被解析错。

ClickHouse（`clickhouseDSN()`）是 `clickhouse://用户:口令@主机:端口/库名`，用户名和口令按 URL 的规则转义。

PostgreSQL（`postgresDSN()`）是 `host='…' port='…' …` 这种写法，每个值加单引号并转义，空值不写。另外固定带三项：

- `sslmode=disable`。
- `application_name=PnWind`：在数据库那边（`pg_stat_activity`）认得出是谁的连接。
- `binary_parameters=yes`：带参数的 SQL 一次网络往返发完，不开是两次。代价是 `[]byte` 类型的参数会按二进制格式发，所以**传给 jsonb 列的值不能直接用 `[]byte`、`json.RawMessage`、`json.Marshal` 的结果**，要过一下 `syscore/utils` 的 `JsonbParam`（空的发 NULL，其余转成 string），否则数据库报 `unsupported jsonb version number`。

写 SQL 时还有一条：结果集要读到头（`for rows.Next()`）或者及时 `Close()`，再发下一条。没读完的结果集一直占着连接，一个请求就同时占了两个，连接池小的时候会自己等自己。

PostgreSQL 的驱动 lib/pq 会读进程环境里以 `PG` 开头的变量，连接串里没写的项以它们为准：`PGOPTIONS`、`PGHOSTADDR`、`PGSERVICE`、`PGCONNECT_TIMEOUT`、`PGCLIENTENCODING` 等，`DBPass` 是空的时候还有 `PGPASSWORD` 和口令文件。**正式运行的环境里不要留这些变量。** 测试时要让服务只看某个 schema，可以在启动它之前设 `PGOPTIONS=-c search_path=<schema>`，用完要清掉，不然同一个终端里再启动的服务读写的也是那个 schema。

### 获取连接

```go
// 获取主库（db1）
db, err := core.GConfig.GetMainDB()

// 获取指定库
db, err := core.GConfig.GetDBByName("db2")
```

---

## 任务管理

`GConfig.Tasks` 是全局唯一的任务管理器（代码在 `pntask/`），管两样东西：按 cron 表达式定时执行的任务，和常驻的后台协程。

```go
tasks := core.GConfig.Tasks

// 定时任务：名字、六段 cron 表达式（秒 分 时 日 月 周）、任务函数
err := tasks.AddTask(pntask.NewTask("清理过期令牌", "0 0 3 * * *", func(ctx context.Context) error {
    return svc.DeleteExpiredTokens(ctx)
}))

// 常驻协程：例如消息队列的消费循环。要留意 Quit()：进程开始退出时它会关闭
err = tasks.Go("消费订单消息", func(ctx context.Context) error {
    for {
        select {
        case <-tasks.Quit():
            return nil
        case msg := <-deliveries:
            handle(ctx, msg)
        }
    }
})
```

- 定时任务在 `Boot` 等服务开始接收连接之后才开始调度；在这之前、之后都可以 `AddTask`，放在包的 `init()` 里也行。表达式写错、名字重复，`AddTask` 返回错误。`main.go` 里现在登记了一个：每天 3 点检查分区，见"分区表管理"。
- `Go` 是立刻启动协程，要等它用到的东西（配置、数据库）就绪之后再调，例如放在 `main.go` 里初始化完数据库之后、调 `Boot` 之前；不要放在 `init()` 里。
- cron 表达式的每一段可以写 `*`、单个值、范围 `1-5`、列表 `1,3,5`、步长 `*/15`。例如 `0 0 3 * * *` 每天 3 点，`*/10 * * * * *` 每 10 秒，`0 30 9 * * 1-5` 周一到周五 9 点半。"日""周"两段都写了限制时，任意一段对上就触发；其中一段以 `*` 开头（单独的 `*`，或者 `*/2` 这样带步长的）时，两段要同时对上：`0 0 9 */2 * 1-5` 是"单数日，并且是周一到周五"。
- 同一个任务上一轮没跑完，到点的这一轮跳过（记一条日志），不会重叠。删除之后马上重新添加同名的任务也一样：旧的那一轮跑完之前，新任务不启动。任务返回错误、panic 都只记日志，不影响下一轮，进程不退出。
- 管理用的方法：`GetAllTasks()`（每个任务上次、下次执行的时间，耗时，执行、失败、跳过的次数）、`RunTask(名字)` 手动触发、`PauseTask` / `ResumeTask` 暂停和恢复、`DeleteTask`、`GetAllGos()`。
- 退出时 `Boot` 一直等正在执行的任务和所有常驻协程返回，不设超时（见"Boot() 启动流程"第 8 步）。任务函数的 `ctx` 不会因为退出而取消，任务照常跑完；常驻协程不看 `Quit()` 的话，进程就退不掉；`exitFn` 在任务全部结束之后才执行，不能指望它来通知常驻协程。不要在任务函数、常驻协程里调 `Tasks.Stop()`，它会等自己。

---

## 服务插件

服务插件是不属于哪个页面、随进程启动和退出的一块功能。接口是 `core/service` 包的 `Service`：

```go
type Service interface {
    Setup(app *fiber.App)
}
```

- `main.go` 里建好插件，在调 `Boot` 之前放进列表：`core.GConfig.Services = append(core.GConfig.Services, 插件)`。现在列表是空的。
- `Boot` 等服务开始接收连接、开始调度定时任务之后，按列表的顺序调每个插件的 `Setup`，都调完了才登记退出信号。
- 插件在 `Setup` 里启动自己，要很快返回：常驻的部分用 `core.GConfig.Tasks.Go` 登记成常驻协程，看 `Quit()` 收尾，退出时内核一直等它返回（见"任务管理"）。`Setup` 执行期间收到退出信号，进程按系统默认立刻结束。
- 这时路由树已经建好，在 `Setup` 里往 `app` 上挂的路由、中间件不生效；要挂路由的功能放进 `PreSetup`。
- `Setup` 的参数里不带内核的配置：`core` import 了 `core/service`，接口再提到 `SysConfig` 就成了循环引用。插件要用内核的东西，从全局的 `core.GConfig` 取。
- 页面的业务服务（`XxxService`）不是插件，内嵌的是 `core/pageservice` 的 `PageService`，不放进 `Services`。

---

## 分区表管理

`pnpartmgr/` 给按月分区的表（PostgreSQL，`PARTITION BY RANGE`，分区键是日期或时间列）提前建好后面几个月的分区。它是可选组件，不在 `Boot` 的流程里，由 `main.go` 装配：

```go
partTables := []string{"db_demo_partition"} // 要管的表：业务上按月分区的表，往这里加表名
pm := pnpartmgr.NewPnPartitionManager(DBOne)
_ = pm.CheckPartitionsAll(partTables, 3, 1) // 启动时检查一次：过去 1 个月到未来 3 个月
err = core.GConfig.Tasks.AddTask(pntask.NewTask("检查分区", "0 0 3 * * *", func(ctx context.Context) error {
    return pm.CheckPartitionsAll(partTables, 3, 1) // 之后每天 3 点一次
}))
```

- 主表要先建好：`db_demo_partition` 是演示用的，建表语句在 `sql/0001_create_tables.sql`。分区的名字是 `<表名>_<年月>`，例如 `db_demo_partition_202611`。
- 分区表的样子照演示表来：没有自增的 `id`；第一列是单号 `order_no`（`bigint`，雪花 ID，由程序给）；`created_at`（`timestamptz`）记创建的时刻；另有一列日期 `created_date`（`date`，默认 `CURRENT_DATE`）专门当分区键；主键是 `(order_no, created_date)`；`PARTITION BY RANGE (created_date)`。
- 表名按连接的 search_path 找，也可以写成 `schema.table`。表名是直接拼进 SQL 的，只能是代码里写死的名字，不要把请求里的值当表名。
- 每个月份各自建：有一个建不出来，接着建后面的，最后把失败的一起返回，定时任务的那一轮记成失败。启动时那一次出错只记日志，服务照常启动；主表不存在时也是这样，每次检查记一条"基础表不存在"。
- 分区的范围写的是不带时区的日期（`'2026-11-01'` 到 `'2026-12-01'`）。分区键是日期列（`date`，像演示表的 `created_date`）时，范围就是这两个日期本身，和时区没有关系。分区键是带时区的时间（`timestamptz`）时，数据库按会话的时区解释它：哪一刻换月跟着数据库的时区走，不跟程序所在机器的时区（数据库的时区是 UTC、业务在东八区时，`_202611` 这个分区是从北京时间 11 月 1 日 8 点开始的）。两种情况下，连同一个库的几个实例不管各自在什么系统、什么时区，建出来的分区都一样。
- 建分区要拿主表上最强的锁，最多等 5 秒，等不到这一次就放弃，下一轮再建。
- 每建一个分区往 `db_sys_partition_creation_log` 记一行。这张表不在 `sql/` 的脚本里，是 `NewPnPartitionManager` 建的（没有才建）。
- 另外几个方法：`GetPartitions`（分区的大小、行数、最近一次清理和分析的时间，可以按表、按大小、按行数过滤）、`PrintPartitionTreeWithStats`（把分区树打印到标准输出）、`RunScheduledTask`（按固定间隔检查的循环，要交给 `Tasks.Go` 运行，`Tasks.Quit()` 关闭时返回）。

---

## PROXY 协议说明

服务前面有四层的负载均衡时，连接的来源地址是负载均衡的地址。PROXY 协议让负载均衡在每个连接的最前面带上真实的客户端地址，服务用 `go-proxyproto` 包解析它（代码在 `listen.go`，v1、v2 两种格式都认）。

配置（dbconfig.json）：
```json
{
  "ProxyProtocol": true,
  "ProxyTrusted": ["10.0.0.0/8", "192.168.1.10"]
}
```

- `ProxyProtocol`：开关，不写就是不开。不开时是普通监听，带 PROXY 头的连接会被当成不合法的请求（400）。
- `ProxyTrusted`：可信代理，每一项是一个 IP 或一个网段，有一项写错服务不启动。`main.go` 在开启时另外加了本机地址 `127.0.0.0/8` 和 `::1`，所以配置里不写它服务也能启动，这时只采用本机发来的 PROXY 头。内核自己（`listen.go`）见到空清单是不启动的，不经 `main.go`、直接调 `Boot` 时才会碰到。
- 来自可信代理的连接：带了 PROXY 头就采用头里的客户端地址；没带头按普通连接处理，所以本机直接访问照常可用。
- 其他来源的连接：PROXY 头一律忽略，按连接本身的地址处理，这样外面的人伪造不了来源地址。

PROXY 协议是连接层的协议，不是 HTTP 请求头，要由四层转发来发送：
```nginx
# nginx 的 stream 模块
stream {
    server {
        listen 80;
        proxy_pass 127.0.0.1:9001;
        proxy_protocol on;
    }
}
```

HAProxy 是在 `server` 行上加 `send-proxy`；云上的负载均衡在四层（TCP）监听上打开 Proxy Protocol。nginx 的 http 反向代理（`proxy_pass http://…`）不发 PROXY 头，开了 `ProxyProtocol` 也取不到真实地址。

---

## 性能剖析（pprof）说明

配置（dbconfig.json）里写了 `PprofAddr` 才开，例如 `"PprofAddr": "127.0.0.1:6060"`，然后访问 `http://127.0.0.1:6060/debug/pprof/`（代码在 `listen.go`）。

- 单独一个端口，不挂在业务端口上；只能是本机地址（`127.0.0.1`、`localhost`、`[::1]`）。写成别的地址、或者端口被占用，服务不启动。要从别的机器看，用 SSH 端口转发。
- 退出时最后才停，并且等正在进行的剖析做完。退出等得久的时候，可以用它看卡在哪里（`/debug/pprof/goroutine?debug=2`）。

---

## 日志

业务代码里这样用（包名是 `pnlogrus`，一律起别名 `log`；代码在 `pnlogrus/pnlogrus.go`）：

```go
import log "github.com/pony5551/PnWind/core/pnlogrus"

log.Println("连接数据库:", name)          // 带 ln：参数之间总有一个空格
log.Errorf("请求 %s 出错: %v", path, err) // 带 f：按格式串
```

- 一共 21 个函数：`Print`、`Debug`、`Info`、`Warn`、`Error`、`Panic`、`Fatal` 各三个（不带后缀、带 `f`、带 `ln`）。`Print` 一组等同 `Info`；级别固定是 Info，`Debug` 一组不输出。
- 不带后缀的（`log.Error(a, b)`）把参数直接拼在一起，只有相邻两个都不是字符串时才加空格；要空格就用带 `ln` 的，字符串末尾不用自己留空格。
- `Panic` 一组写完日志后 panic。`Fatal` 一组写完日志后直接结束进程：不执行 `defer`，不等定时任务和常驻协程，绕过优雅退出，业务代码不要用。
- 两路输出：控制台和日志文件（一行一条 JSON：`level`、`time`、`msg`）。控制台那一路只有标准输出是终端时级别才带颜色；被重定向到文件、接到管道、被 systemd 或 docker 收集时不带颜色码（docker 加了 `-t`、分配了终端的除外）。
- 标准输出交给 systemd、docker 收集的部署，同一条日志会在日志文件里和收集的地方各存一份。在配置（dbconfig.json）里写 `"LogFileOnly": true` 就只写日志文件：启动时记一行"日志只写文件：控制台的输出到这一行为止，往后的看 <日志目录>"，之后的日志（包括启动失败的原因）控制台里不再有，要到日志文件里看。不经日志包的输出不归这个开关管：读配置之前的那几行、Fiber 的启动横幅、退出时断开数据库出错的那一行仍然在标准输出里；日志文件写不进去时的 write error、Fiber 自己的警告、panic 的堆栈在标准错误里。不写或写 `false` 是两路都写。
- 日志文件是 `logs/[日期]/<进程名>-<进程号>-mylog.log`，例如 `PnWind-608-mylog.log`；进程名是可执行文件的名字，Windows 上不带 `.exe`。一天一个目录，过了零点自动换；一个进程一个文件，超过 100MB 切分，切出来的旧文件压缩保存，同一天里最多留 30 个。文件的权限是 0640（属主读写，同组可读；还要过一遍进程的 umask，Windows 上没有这套权限）：日志采集程序用别的用户运行时，把它加进服务所在的组。运行中不要手工删当天的日志文件：之后再切分出来的新文件又是只有属主能读，要到第二天才恢复。
- `logs` 默认在可执行文件所在的目录下。要放到别处，在配置（dbconfig.json）里写 `"LogDir": "/var/log/pnwind"`（相对路径按运行目录算，不认 `~` 和环境变量）。不管是默认的还是配的，目录没有就自己建；启动时确认一遍能用（把当天的日志文件建出来），建不出来、没有权限写时服务不启动，报"日志目录 … 用不了"。运行服务的用户要对它有写权限：`/var/log` 下面普通用户建不了目录，要预先建好再交给它；可执行文件装在 root 的目录里、服务用普通用户运行时，要么配 `LogDir`，要么把可执行文件旁边的 `logs` 预先建好交给它。
- 日期目录保留 365 天：今天和之前的 365 天留着，更早的整个删掉，删一个记一条日志。启动时清理一次，之后每换一天清理一次。要改天数，在配置（dbconfig.json）里写 `"LogKeepDays": 30`（不写、写成 0 或负数都按 365 天，最大 36500）；启动时日志里有一行"日志保留 N 天"，配成了多少看它。只删日志目录下名字是 `[年-月-日]` 的目录，别的不动；配了 `LogDir` 之后，可执行文件旁边原来的 `logs` 不再有人清理。
- 不配 `LogDir` 时日志目录跟着可执行文件走，配置跟着运行目录走：同一个可执行文件在几个运行目录里各起一个实例时，它们共用一个 `logs`，保留天数配得最小的那个说了算。各配各的 `LogDir` 就分开了。
- 用符号链接切换版本的发布方式（`current -> releases/v1`）下，可执行文件所在的目录是解析过链接的那个版本目录，日志会跟着旧版本一起被清掉：配 `LogDir` 放到版本目录之外。
- 几个进程把日志写到同一个目录时，靠文件名分开：不同的程序进程名不同；同一个程序的几个进程（同一台机器、同一个容器里起了几个）进程号不同。**几个容器把同一个目录挂进去当日志目录时分不开**：各个容器里的服务进程号常常都是 1，名字一样，会写进同一个文件，切分时互相截断。这种部署要让每个容器用各自的日志目录（`LogDir`）。
- 日志里的时间、日期目录什么时候换、定时任务的 cron，都按进程的时区。Linux 上看环境变量 `TZ`，没设就用系统的时区；容器里通常是 UTC，要东八区就在启动时设 `TZ=Asia/Shanghai`（镜像里写 `ENV TZ=Asia/Shanghai`，systemd 的单元里写 `Environment=TZ=Asia/Shanghai`）。时区数据已经编进程序里，镜像里没有装 tzdata 也认。`TZ` 要写这种"洲/城市"的名字，写错了、写成 `CST-8` 都不报错、按 UTC 算；启动时日志里有一行"时区: …"，以它为准。
- `core` 包的 `init()` 里调 `InitZapLog` 接上日志文件，这之前的日志只写控制台。配置里的 `LogDir`、`LogFileOnly` 要等 `main.go` 读完配置才生效，在那之前出的错（配置文件读不了、`LogDir` 用不了）写在默认的目录里，控制台也有。默认的目录本身用不了时，"日志目录 … 用不了"这一条只在控制台看得到，后面跟着的一行 write error 说的是同一件事。

---

## 注意事项

- `GConfig.Debug = true` 会开启模板热重载，**生产环境应改为 false**（`main.go` 现在写死为 `true`）
- `OpenDataDelete` 控制是否开放数据删除接口，生产环境应保持 `false`
- `GConfig.IdGen` 由 `Boot` 第 1 步创建，在这之前是 nil；`Boot()` 返回时会调用 `Stop()`，业务代码不应在 `Boot()` 返回后继续使用
- 多个节点同时运行时，每个节点的 `DeviceId` 要不一样，雪花ID才不会重复
- 退出时内核无条件等定时任务和常驻协程结束：有一个卡住，进程就退不掉。退出等得久的时候，用性能剖析看卡在哪里
- 进程自己会一直等，外面的不一定等：关闭控制台窗口、注销、关机时，Windows 只给几秒就强行结束进程；`docker stop`、systemd、k8s 也各有停止超时，到点就强杀。停服务用 Ctrl+C（或者发 SIGTERM），不要直接关窗口；部署时把这些超时调到够任务跑完
- Linux 上终端断开（ssh 会话结束、关掉终端窗口）不会让服务退出：SIGHUP 被忽略，进程留在后台继续运行，再启动一个会报端口被占用。要停它用 `kill <进程号>`（SIGTERM）；正式部署交给 systemd 这类守护程序来管
- Linux 上标准输出或标准错误接着管道时（`./PnWind | tee x.log`、不带终端的 `ssh 主机 '命令'`、被别的程序收集），读的那一头先退出之后，服务再写一条日志就会被系统以 SIGPIPE 直接结束（退出状态 141）：不走退出流程，不等定时任务和常驻协程。在终端里对 `./PnWind | tee x.log` 按 Ctrl+C 就是这样，`tee` 先退出。正式的服务不要这样运行：要留一份输出就重定向到文件（`> out.log 2>&1`）；打开 `LogFileOnly` 之后稳定运行时基本不再写标准输出，碰上的机会小得多，但不是完全没有
- 请求体上限 500MB 是 `Boot` 里写死的，在 `GConfig.Fiber.BodyLimit` 里设别的值不起作用
- 在老式的 Windows 控制台窗口里运行时，用鼠标选中窗口里的文字会让写日志的地方停住，所有请求跟着卡住，按 Esc 或回车恢复。Windows Terminal 和 Linux 没有这个问题
