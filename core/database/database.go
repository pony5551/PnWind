// Package database 数据库连接：DBConfig 是一个库的连接配置，DBConn 持有配置和连接池。
// 支持 PostgreSQL 和 ClickHouse。
package database

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2" // ClickHouse 驱动
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL 驱动
)

// -------------------------------------------------------------------------
// 连接池参数
//
// MaxConnNum（最大连接数）：
//   控制连接池中同时存在的最大连接数（活跃 + 空闲）。
//   超出后新请求将排队等待，直到有连接释放。
//   默认 50。它要比数据库允许的连接数小（PostgreSQL 的 max_connections 默认 100，
//   各个库、各个节点、查库的工具一起算）：超过了数据库直接拒绝新连接，请求报错。
//   并发高、数据库那边也调大了，再在配置文件里用 maxConnNum 调大。
//
// MaxIdleNum（最大空闲连接数）：
//   低峰期连接池保留的空闲连接数，用于快速响应突发请求，避免频繁建立新连接。
//   不应超过 MaxConnNum；设置过大会长期占用数据库连接资源。
//   默认 10，配置文件里用 maxIdleNum 调。
//
// 连接最大存活时间：
//   一条连接从创建起最长可使用的时间，到期后在下次被取出时关闭并重建。
//   用于防止使用过旧的连接（数据库侧可能已超时断开）。固定 30 分钟。
//
// 连接最大空闲时间：
//   一条连接在池中空闲的最长时间，超过后主动关闭回收。
//   比最大存活时间小，避免取出时连接已过期。固定 10 分钟。
// -------------------------------------------------------------------------

const (
	defaultMaxConnNum = 50
	defaultMaxIdleNum = 10
	connMaxLifetime   = 30 * time.Minute
	connMaxIdleTime   = 10 * time.Minute
)

// DBConfig 数据库连接配置，对应配置文件里 dbs 下的一项
type DBConfig struct {
	DBTag      string // 库的标签，即 dbs 下的键，例如 db1
	DBDriver   string // postgres 或 clickhouse
	DBHost     string
	DBPort     string
	DBUser     string
	DBPass     string
	DBName     string
	MaxConnNum int // 最大连接数，0 则使用 defaultMaxConnNum
	MaxIdleNum int // 最大空闲连接数，0 则使用 defaultMaxIdleNum
}

// DBConn 数据库连接实例，持有配置与连接池
type DBConn struct {
	DBConf DBConfig
	DB     *sqlx.DB
}

// DBOpen 初始化数据库连接池
//
// 使用 sqlx.Connect（Open + Ping），依赖操作系统的 TCP 连接超时（Linux 约 2 分钟，Windows 约 21 秒）：
// 系统核心依赖数据库，宁可等待连接成功，也不因超时提前放弃，所以不设连接超时。
// 连接池建立后，database/sql 内置自动重连机制：
// 断网期间查询返回错误，网络恢复后连接池自动重建连接，无需人工干预。
//
// 连不上（例如口令不对）返回带库标签的错误，由调用方记日志、决定是否终止启动。
func (con *DBConn) DBOpen() (*sqlx.DB, error) {
	conf := con.DBConf
	var dsn string
	switch conf.DBDriver {
	case "postgres":
		dsn = conf.postgresDSN()
	case "clickhouse":
		dsn = conf.clickhouseDSN()
	default:
		return nil, fmt.Errorf("数据库 [%s] 的 DBDriver 是 %q：只支持 postgres、clickhouse", conf.DBTag, conf.DBDriver)
	}
	db, err := sqlx.Connect(conf.DBDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("连接数据库 [%s] 失败: %w", conf.DBTag, err)
	}

	maxConn, maxIdle := conf.MaxConnNum, conf.MaxIdleNum
	if maxConn <= 0 {
		maxConn = defaultMaxConnNum
	}
	if maxIdle <= 0 {
		maxIdle = defaultMaxIdleNum
	}
	db.SetMaxOpenConns(maxConn) // 超出后新请求排队等待
	db.SetMaxIdleConns(maxIdle) // 低峰期保留的热连接数；比最大连接数大时按最大连接数算
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)
	return db, nil
}

// DBClose 关闭数据库连接池
func (con *DBConn) DBClose() error {
	if con.DB != nil {
		return con.DB.Close()
	}
	return nil
}

// clickhouseDSN 拼出 ClickHouse 的连接串：clickhouse://用户:口令@主机:端口/库名。
// 用户名、口令交给 url 包转义：里面有 @ : / # ? 这些字符也不会被解析错。连接串里有口令，不要写进日志。
func (c DBConfig) clickhouseDSN() string {
	u := url.URL{
		Scheme: "clickhouse",
		User:   url.UserPassword(c.DBUser, c.DBPass),
		Host:   net.JoinHostPort(c.DBHost, c.DBPort),
		Path:   "/" + c.DBName,
	}
	return u.String()
}

// postgresDSN 拼出 PostgreSQL 的连接串。每个值用单引号包住并转义：口令里有空格、引号、反斜杠都不会被解析错；空值不写。
// 连接串里有口令，不要写进日志。
func (c DBConfig) postgresDSN() string {
	items := [][2]string{
		{"host", c.DBHost},
		{"port", c.DBPort},
		{"user", c.DBUser},
		{"password", c.DBPass},
		{"dbname", c.DBName},
		{"sslmode", "disable"},
		// 在数据库那边（pg_stat_activity）认得出是谁的连接
		{"application_name", "PnWind"},
		// 带参数的语句一次网络往返发完。不开的话 lib/pq 先发 Parse、等回包，再发 Bind 和 Execute，是两次。
		// 开了之后 []byte 类型的参数按二进制格式发：传给 jsonb 列的值要先转成 string，
		// 不能直接传 []byte、json.RawMessage、json.Marshal 的结果
		{"binary_parameters", "yes"},
	}
	quote := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	var b strings.Builder
	for _, kv := range items {
		if kv[1] == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(kv[0] + "='" + quote.Replace(kv[1]) + "'")
	}
	return b.String()
}
