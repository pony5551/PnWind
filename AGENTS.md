# AGENTS.md

本文件写给在本仓库工作的 AI agent，人也适用。动手之前先读完，再读 `project-memory/README.md`：开发规范和 agent 工作规范在那里，每条规矩都带着理由，和本文件有出入时以那里为准。

文中的"维护者"指你正在为之工作的人。在维护者自己的工作区里，就是他本人。你是外部贡献者、或者在替贡献者干活时，就是这位贡献者：在自己的分支上提交、走 pull request，测不测由他定，发 pull request 时写明测了什么、没测什么。

## 项目

PnWind：Go 中后台用户管理框架（Fiber v3 · sqlx · PostgreSQL），配套管理前端（`web/`，基于 vue-pure-admin）。卖点排序：轻量 → 易上手 → 性能强。

现在有：内核、登录与鉴权、八个系统页面、任务管理、分区表管理和一张演示用的分区表。怎么编译、怎么跑起来、接口长什么样，见 `README.md`。

## 中心思想

这个项目给代码定的标准，每次动代码都适用：**认真，负责，严谨，简洁，高效，封装化**。

- 只做被要求的那一件事，不顺手添类型、参数、开关。
- 加任何东西之前先答三个问题：现在有没有人用得上；会带来什么别的后果；Windows、Linux、macOS 上各是什么表现。
- 先把现有的代码和它依赖的库读明白再动手，库的行为以源码为准，不凭印象。
- 现有的实现能用就不要整段改写；照已有的写法写。

## 目录

| 路径 | 内容 |
|---|---|
| `main.go` | 装配清单：读配置、连数据库、加载系统参数、检查分区、挂中间件和路由、启动 |
| `core/` | 内核，只做基础设施。文件结构、`Boot` 的启动流程、配置、连接池、任务管理、日志等都写在 `core/CORE.md` 里；**动了内核，这份文档跟着改** |
| `core/pncontext/` | 请求上下文 `PnContext`：`Bind`、`RespOK`、`RespErr`、`Tx`、`DBByName`，业务错误 `NewError` |
| `core/validation/` | 请求参数的解析和校验，`pnctx.Bind` 用的就是它 |
| `core/database/` | 数据库连接（PostgreSQL、ClickHouse） |
| `core/pnlogrus/` | 日志：`log.Println`、`log.Errorf` 这一套函数 |
| `core/pntask/` | 任务管理：定时任务和常驻的后台协程 |
| `core/pnpartmgr/` | 分区表管理 |
| `core/pnlibs/` | 通用库：并发 map `PnObjects`、雪花 ID、`ExitWatch` 等 |
| `syscore/routers/` | 页面注册（`RegisterController`）、路由安装、act 分发 |
| `syscore/middleware/` | 登录校验、权限、操作日志的入口（`auth_jwt.go`） |
| `syscore/logutils/` | 操作日志入库 |
| `syscore/utils/` | 响应码（`mgr_define.go`）、统一的错误出口（`ErrorHandler`）、页面常用的助手函数 |
| `syscore/PnWebSocket/` | WebSocket（`/ws`） |
| `sysauth/` | 登录、退出 |
| `sysmgr/` | 八个系统页面：`<页面>_controller.go`、`<页面>_service.go`，结构体在 `sysmgr/models/<页面>_model.go`，注册在 `00_registercontroller.go` |
| `sql/` | `0001_create_tables.sql` 建表，`0002_seed_data.sql` 初始数据，`upgrade_<日期>_<表名>.sql` 升级脚本 |
| `conf/` | 配置文件 `dbconfig.json`。运行时用的那一份里有数据库口令和签名密钥：内容不要输出，不要复制到别处 |
| `views/` | 模板目录 |
| `web/` | 管理前端，基于 vue-pure-admin |
| `project-memory/` | 开发规范和 agent 工作规范，入口 `project-memory/README.md` |
| `_bin/` | 编译产物和运行目录，不进仓库。服务进到这里启动 |

内核只做基础设施，业务的约定放 `syscore/`。往 `core/` 里加代码之前先问：它是不是在拆地址、取 act、判断登录、算权限、碰具体的表？是就不属于内核。新增代码放进已有的对应目录，不另起一层。

## 命令

在仓库根目录执行：

| 命令 | 作用 |
|---|---|
| `build_win.cmd` | 编译成 `_bin\PnWind.exe`。先停掉正在跑的服务 |
| `build_linux.cmd` | 在 Windows 上交叉编译出 Linux 用的 `_bin\PnWind` |
| `go build ./...` | 只检查编不编得过。交出去之前三个系统都要过：`GOOS` 设成 `windows`、`linux`、`darwin` 各编一遍（加 `CGO_ENABLED=0`） |
| `gofmt -l <改过的文件>` | 检查格式，应当没有输出 |
| `go vet <改过的包>` | 静态检查，连测试文件一起 |
| `go test ./core/database ./core/pnlogrus ./core/pntask ./core/validation` | 内核的单元测试。连数据库的要设 `PNWIND_TEST_DSN`，没设时自动跳过。不要写成 `./core/...`：第三方库的副本 `core/validation/govalidator` 自己的测试代码过不了 `go test` 自带的 `go vet` 检查，会显示失败 |
| `pnpm install`、`pnpm build`（在 `web/` 里） | 构建前端，产物在 `web/dist`，复制成 `_bin/public` |
| `pnpm dev`（在 `web/` 里） | 前端开发服务器，8848 端口，`/web` 开头的请求转给 `http://127.0.0.1:9001` |

需要 Go 1.26.11 及以上。

静态检查另有 golangci-lint，配置是仓库根目录的 `.golangci.yml`。不用安装，直接从源码运行（第一次要下载、构建，之后走缓存）：`go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run ./...`。现在的代码按这份配置还有几十条提示。不在检查范围里的有六处：第三方库的副本 `core/validation/govalidator/`、项目自带的通用库 `core/pnlibs/`、可选组件 `core/pnamqp/`，以及还没有整理的 `core/util/`、`syscore/utils/`、`syscore/PnWebSocket/`。

要起服务做测试时，用自己的端口和运行目录，不占 9001；收尾只结束自己启动的那个进程，不按名字结束。

## 写一个页面

照 `sysmgr/` 里部门管理的三个文件写（`sysdept_controller.go`、`sysdept_service.go`、`models/sysdept_model.go`），再在 `sysmgr/00_registercontroller.go` 里加一段。

- **结构体**（`sysmgr/models/<页面>_model.go`）：和表对应的结构体带 `db`、`json` 标签，接口里的字段名和列名一样。表单结构体另外带校验用的标签：`validate`（规则，写法见 go-playground/validator）、`label`（字段的中文名，校验不通过时用它生成提示）、`trim:"true"`（先去掉首尾空白再校验）、`default`（没传时的默认值）、`msg`（直接指定提示）。
- **服务**（`<页面>_service.go`）：结构体内嵌 `service.Service`，方法的第一个参数是 `pnctx *pncontext.PnContext`，用 `pnctx.DB` 查库。SQL 手写，值一律走参数；`LIKE` 的值先过 `utils.EscapeLikeParam`。要事务用 `pnctx.Tx`。
- **控制器**（`<页面>_controller.go`）：结构体内嵌 `controller.ControllerBase`。一个操作一个方法 `Page_<act>(pnctx *pncontext.PnContext) error`，act 是方法名去掉 `Page_` 再转成小写。方法里用 `pnctx.Bind(&表单)` 取参数并校验；成功 `pnctx.RespOK(数据, 提示)`，业务上不行 `pnctx.RespErr(错误码, 提示)`；别的错误直接 `return err`，由统一的错误出口转成响应、记日志。删除类的操作先看 `core.GConfig.OpenDataDelete`。列表没有数据时返回 `[]`，不返回 `null`。
- **注册**：`routers.RegisterController(core.GConfig, "web", 页面地址, 控制器)`；不用登录的页面用 `routers.RegisterPublicController`。接口地址是 `/web<页面地址>/<act>`。控制器写了 `PageMain` 方法时，这个页面的请求全部交给它，不按 act 分。
- **权限**：每个 act 要在菜单表 `db_sys_menus` 里有一行按钮（`menu_type = 3`，`name` 是 `m_<页面地址的最后一段>:<act>`），没有的话登录校验回 `-30001` 未配置权限。这一行的 `loged` 为真才记操作日志。超级管理员之外的人，菜单权限里要有这个 `name`，否则回 `-30002`。
- **前端**：照部门管理写。页面文件放在 `web/src/views/sysmgr/<组>/<页面>/`（`index.vue`、`form.vue`、`utils/`）；接口文件是 `web/src/api/sysmgr/<组>/<页面>.ts`，接口地址的常量加在 `web/src/api/sysmgr/sysmgr_define.ts` 里，请求的地址是常量后面接 `/<act>`。菜单表里页面那一行的 `path` 就是页面文件的位置（部门管理是 `/sysmgr/sysset/sysdept/index`，对应 `web/src/views/sysmgr/sysset/sysdept/index.vue`）；`index.vue` 里 `defineOptions` 的 `name` 要和菜单这一行的 `name` 一样（`m_sysdept`）。

## 数据库

- 表结构以 `sql/0001_create_tables.sql` 为准。改表结构要两处一起改：`0001` 改成新的样子，再加一个 `sql/upgrade_<日期>_<表名>.sql` 给已有的库用。升级脚本的写法：开头说明做什么、和别的脚本的先后、执行之前要不要停服务；整个放在一个事务里；先检查库是不是处在该升级的状态，对不上就中止、什么都不改。
- 列名用下划线分词（`user_id`、`created_at`）；时间列用带时区的类型（`timestamptz`）。
- 分区表照 `db_demo_partition` 的样子：没有自增的 `id`；第一列是单号（`bigint`，雪花 ID，由程序给）；`created_at` 记创建的时刻；另有一列日期 `created_date` 专门当分区键；主键是 `(单号, created_date)`。每个月的分区由分区管理器建，见 `core/CORE.md` 的"分区表管理"。
- PostgreSQL 的连接固定带 `binary_parameters=yes`：`[]byte` 类的值传给 jsonb 列之前要过 `utils.JsonbParam`。
- 连接池同时支持 PostgreSQL 和 ClickHouse。ClickHouse 的支持必须保留。

## 日志、后台任务、启动和退出

- 日志：`import log "github.com/pony5551/PnWind/core/pnlogrus"`，用 `log.Println`、`log.Printf`、`log.Warnf`、`log.Errorf` 这一套。失败记 Error，警告记 Warn。不要用 `log.Fatal` 那一组：它直接结束进程，绕过优雅退出。
- 后台的活（定时任务、常驻的循环）一律登记到任务管理器 `core.GConfig.Tasks`（`AddTask`、`Go`），不要自己起 `go` 循环：退出时内核只等登记过的。常驻协程要看 `Tasks.Quit()`，它关闭了就返回。
- `main.go` 里启动失败的写法是定好的：把错误赋给开头声明的 `err`、记一条日志、`return`，由开头那个 `defer` 给退出码 1。`if` 里要写 `err = …`，写成 `err := …` 退出码就成了 0，编译器不报。只记日志、接着往下走的地方用 `:=`。
- 退出信号到 `Boot` 的最后一步才登记：启动完成之前收到的，进程按系统默认直接结束；之后收到的走优雅退出。不要加"提前登记信号"。

## 红线：改动之前必须先得到维护者同意

1. **优雅退出绝对等待。** 退出时必须等后台任务和在途请求全部完成。不加超时，不加强制退出，第二个信号也不能打断等待。动到退出流程之前先问。
2. **凭据不外流。** 密码、密钥、内网地址、数据库连接串不写进代码、示例、测试、文档，不打印到输出里。运行时用的配置文件，内容不输出、不复制。
3. **疑点先问。** 现有代码里看起来像缺陷的行为，先报告证据和两种可能的解释，等维护者确认，不擅自"修复"。
4. **不精简掉已有的功能。** 现在没有人调用不等于可以删，代码可以改进。要去掉什么，先数清楚哪里在用、去掉之后怎么办，报给维护者定。

## 做完一件事

1. 编译：三个系统的 `go build ./...` 都过，改过的文件 `gofmt` 干净。
2. 汇报里分开写：改了什么、行为有什么变化、三个系统上各是什么表现、顺带看到但没有动的。顺带看到的事等维护者回一句再动。
3. 测试先问维护者。他说测再测；说这次先不测的，记进 `project-memory/` 下的待验证清单（第一次用到时建这个文件），下次一起测；说不需要测的，不记。
4. 在维护者的工作区里不替他提交，提交由他自己做。
5. 新定下的规矩写进 `project-memory/`，不写到别处。

## 环境变量

| 变量 | 用途 |
|---|---|
| `PNWIND_TEST_DSN` | 连数据库的测试用的连接串，形如 `postgres://账号:口令@地址:端口/库名`。值不要打印出来 |
