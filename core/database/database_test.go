package database

import (
	"database/sql"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/lib/pq"
)

// clearPGEnv 测试期间把进程里以 PG 开头的环境变量去掉，结束时恢复。
// lib/pq 解析连接串时会读它们（PGSERVICE、PGCLIENTENCODING 之类）：机器上碰巧设了的话，解析结果就不只取决于连接串了。
func clearPGEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		name, value, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(strings.ToUpper(name), "PG") {
			os.Unsetenv(name)
			t.Cleanup(func() { os.Setenv(name, value) })
		}
	}
}

// 口令的各种写法：空格、引号、反斜杠、URL 里有特殊含义的字符、空串。
var passwords = []string{"abc123", "pa ss", `pa\ss`, "'abc", "abc'def", `p' w\d`, "p@ss:w/rd#1?x=%41", "a=b", ""}

// PostgreSQL 的连接串交给 lib/pq 解析，每一项都要原样读得回来。
func TestPostgresDSNRoundTrip(t *testing.T) {
	clearPGEnv(t)
	for _, pass := range passwords {
		c := DBConfig{DBHost: "10.0.0.8", DBPort: "5432", DBUser: "u ser", DBPass: pass, DBName: "my db"}
		cfg, err := pq.NewConfig(c.postgresDSN())
		if err != nil {
			t.Errorf("口令 %q：连接串解析出错: %v", pass, err)
			continue
		}
		if cfg.Host != c.DBHost || cfg.Port != 5432 || cfg.User != c.DBUser || cfg.Database != c.DBName {
			t.Errorf("口令 %q：解析出来 host=%q port=%d user=%q dbname=%q", pass, cfg.Host, cfg.Port, cfg.User, cfg.Database)
		}
		if pass != "" && cfg.Password != pass { // 空口令不写进连接串，lib/pq 会另外去找（口令文件），这里不比
			t.Errorf("口令 %q 被解析成了 %q", pass, cfg.Password)
		}
		if cfg.SSLMode != pq.SSLModeDisable || cfg.ApplicationName != "PnWind" || !cfg.BinaryParameters {
			t.Errorf("固定的三项不对：sslmode=%q application_name=%q binary_parameters=%v", cfg.SSLMode, cfg.ApplicationName, cfg.BinaryParameters)
		}
		if len(cfg.Runtime) != 0 {
			t.Errorf("口令 %q：多出了连接参数 %v", pass, cfg.Runtime)
		}
	}
}

// 空值不写进连接串：口令是空的时候，尤其不能把后面的项吃掉。
func TestPostgresDSNSkipsEmpty(t *testing.T) {
	got := DBConfig{DBHost: "h", DBName: "db"}.postgresDSN()
	want := `host='h' dbname='db' sslmode='disable' application_name='PnWind' binary_parameters='yes'`
	if got != want {
		t.Errorf("连接串是 %s\n期望 %s", got, want)
	}
}

// ClickHouse 的连接串交给它的驱动解析，每一项都要原样读得回来。
func TestClickhouseDSNRoundTrip(t *testing.T) {
	for _, pass := range passwords {
		c := DBConfig{DBHost: "10.0.0.8", DBPort: "9000", DBUser: "us@er", DBPass: pass, DBName: "my_db"}
		opt, err := clickhouse.ParseDSN(c.clickhouseDSN())
		if err != nil {
			t.Errorf("口令 %q：连接串解析出错", pass) // 错误里可能带着连接串，不打印
			continue
		}
		if len(opt.Addr) != 1 || opt.Addr[0] != "10.0.0.8:9000" || opt.Auth.Username != c.DBUser || opt.Auth.Password != pass || opt.Auth.Database != c.DBName {
			t.Errorf("口令 %q：解析出来 addr=%v user=%q password=%q database=%q", pass, opt.Addr, opt.Auth.Username, opt.Auth.Password, opt.Auth.Database)
		}
	}
	// 主机是 IPv6 地址时要带方括号
	if got := (DBConfig{DBHost: "::1", DBPort: "9000", DBUser: "default", DBName: "default"}).clickhouseDSN(); got != "clickhouse://default:@[::1]:9000/default" {
		t.Errorf("IPv6 的连接串是 %s", got)
	}
}

// 两种驱动都编进来了；别的驱动在连接之前就被拒绝，错误里指出是哪个库。
func TestDrivers(t *testing.T) {
	for _, name := range []string{"postgres", "clickhouse"} {
		if !slices.Contains(sql.Drivers(), name) {
			t.Errorf("驱动 %s 没有注册", name)
		}
	}
	con := DBConn{DBConf: DBConfig{DBTag: "db9", DBDriver: "sqlserver"}}
	_, err := con.DBOpen()
	if err == nil || !strings.Contains(err.Error(), "db9") || !strings.Contains(err.Error(), "postgres、clickhouse") {
		t.Errorf("应当拒绝别的驱动并指出是哪个库，得到 %v", err)
	}
}

func TestDBCloseWithoutOpen(t *testing.T) {
	if err := (&DBConn{}).DBClose(); err != nil {
		t.Errorf("没打开过的连接，关闭应当没有错误，得到 %v", err)
	}
}

// 连真实的 PostgreSQL，只做只读的查询（select、查 pg_stat_activity），另外用错的口令连一次。
// 连接串取自环境变量 PNWIND_TEST_DSN（URL 形式），没有设置就跳过。
func TestDBOpenReal(t *testing.T) {
	conf, ok := testConfig(t)
	if !ok {
		t.Skip("没有设置 PNWIND_TEST_DSN，跳过连库测试")
	}
	open := func(c DBConfig) *DBConn {
		con := &DBConn{DBConf: c}
		db, err := con.DBOpen()
		if err != nil {
			t.Fatal("连不上数据库（错误内容不输出）")
		}
		con.DB = db
		return con
	}

	con := open(conf)
	var n int
	if err := con.DB.Get(&n, "select $1::int", 1); err != nil || n != 1 {
		t.Errorf("select 得到 %d，%v", n, err)
	}
	var app string
	if err := con.DB.Get(&app, "select application_name from pg_stat_activity where pid = pg_backend_pid()"); err != nil || app != "PnWind" {
		t.Errorf("连接在数据库那边的名字是 %q，%v", app, err)
	}
	if got := con.DB.Stats().MaxOpenConnections; got != defaultMaxConnNum {
		t.Errorf("没有配置时最大连接数是 %d，期望 %d", got, defaultMaxConnNum)
	}
	if err := con.DBClose(); err != nil {
		t.Error(err)
	}

	conf.MaxConnNum = 7
	con = open(conf)
	if got := con.DB.Stats().MaxOpenConnections; got != 7 {
		t.Errorf("MaxConnNum 没有生效：%d", got)
	}
	_ = con.DBClose()

	// 口令不对：返回错误，错误里要有库的标签，不能带口令。
	bad := conf
	bad.DBPass = "wrong password'for\\test"
	db, err := (&DBConn{DBConf: bad}).DBOpen()
	switch {
	case err == nil: // 这个库对本机不验口令（trust 认证），这一项测不了
		_ = db.Close()
		t.Log("用错的口令也连上了：这个库不验口令，跳过口令不对的检查")
	case !strings.Contains(err.Error(), "[db1]") || strings.Contains(err.Error(), "wrong") || strings.Contains(err.Error(), `\test`):
		t.Error("口令不对时的错误应当带库的标签，而且不能带出口令（错误内容不输出）")
	}
}

// testConfig 把 PNWIND_TEST_DSN 拆成 DBConfig。
func testConfig(t *testing.T) (DBConfig, bool) {
	t.Helper()
	raw := os.Getenv("PNWIND_TEST_DSN")
	if raw == "" {
		return DBConfig{}, false
	}
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		t.Fatal("PNWIND_TEST_DSN 不是合法的连接串，应当形如 postgres://用户:口令@主机:端口/库名")
	}
	pass, _ := u.User.Password()
	return DBConfig{
		DBTag:    "db1",
		DBDriver: "postgres",
		DBHost:   u.Hostname(),
		DBPort:   u.Port(),
		DBUser:   u.User.Username(),
		DBPass:   pass,
		DBName:   strings.TrimPrefix(u.Path, "/"),
	}, true
}
