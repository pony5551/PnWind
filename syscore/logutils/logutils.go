// Package logutils 写操作日志（db_sys_logopt 表）。
package logutils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pony5551/PnWind/core"
	log "github.com/pony5551/PnWind/core/pnlogrus"

	"github.com/jmoiron/sqlx"
)

// maxBody 操作日志里请求体最多记这么多字节（64KB），再长就截断
const maxBody = 64 << 10

// secretField 找出请求体里名字带 pass、pwd 的字段（password、pay_password、new_pwd 等）和它的值；
// 值被截断、没有收尾的引号时也算
var secretField = regexp.MustCompile(`(?i)("[^"]*(?:pass|pwd)[^"]*"\s*:\s*)"(?:[^"\\]|\\.)*(?:"|\\?$)`)

// SysOptLogItem 一条操作日志
type SysOptLogItem struct {
	DeviceId  int64  `db:"device_id"`  // 节点号，DbInsertLogOpt 按配置里的 DeviceId 填
	UserType  int16  `db:"user_type"`  // 1 管理员，2 用户
	UserId    string `db:"user_id"`    // 管理员表或用户表的 user_id
	Type      string `db:"type"`       // 类别，例如 用户操作
	Events    string `db:"events"`     // 页面事件，也就是 act
	Details   string `db:"details"`    // 页面事件详情，例如 部门管理-添加
	PagePath  string `db:"page_path"`  // 页面地址，例如 /web/sysmgr/sysset/sysdept
	PageQuery string `db:"page_query"` // 地址参数
	PageForm  string `db:"page_form"`  // 请求体，用 Body 整理过的
	Ip        string `db:"ip"`
}

// Body 整理要记进操作日志的请求体：只记前 maxBody 个字节，密码一类字段的值换成 ***。
func Body(body []byte) string {
	total := len(body)
	if total > maxBody {
		body = body[:maxBody]
	}
	s := secretField.ReplaceAllString(string(body), `${1}"***"`)
	if total > maxBody {
		s += fmt.Sprintf("…（请求体共 %d 字节，只记了前 %d 字节）", total, maxBody)
	}
	return s
}

// DbInsertLogOpt 写一条操作日志，记录时间由表上的默认值填。写失败时在这里记一条错误日志。
func DbInsertLogOpt(db *sqlx.DB, LogItem *SysOptLogItem) error {
	LogItem.DeviceId = core.GConfig.DeviceId
	// 请求里带来的内容进库之前，先把数据库的文本列存不了的字节换掉：
	// 不然请求里塞一个这样的字节，操作照常成功，日志却写不进去
	LogItem.PageQuery, LogItem.PageForm = cleanText(LogItem.PageQuery), cleanText(LogItem.PageForm)
	_, err := db.NamedExec(`INSERT INTO db_sys_logopt
		(device_id, user_type, user_id, type, events, details, page_path, page_query, page_form, ip)
		VALUES (:device_id, :user_type, :user_id, :type, :events, :details, :page_path, :page_query, :page_form, :ip)`, LogItem)
	if err != nil {
		log.Errorln("写操作日志失败:", err)
	}
	return err
}

// cleanText 把数据库的文本列存不了的字节换掉：不是合法 UTF-8 的换成 ?，空字符去掉。
func cleanText(s string) string {
	return strings.ReplaceAll(strings.ToValidUTF8(s, "?"), "\x00", "")
}
