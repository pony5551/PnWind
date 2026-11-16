// Package middleware 登录校验、权限检查、操作日志（IsAuthenticated），以及登录令牌的签发和解析。
package middleware

import (
	"database/sql"
	"errors"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/pony5551/PnWind/core"
	"github.com/pony5551/PnWind/core/pncontext"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	authmodels "github.com/pony5551/PnWind/sysauth/models"
	"github.com/pony5551/PnWind/syscore/logutils"
	"github.com/pony5551/PnWind/syscore/utils"
	"github.com/pony5551/PnWind/sysmgr/models"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// TokenLife 登录令牌的有效期。登录接口回给前端的 expires 用的是同一个值
	TokenLife = 72 * time.Hour

	superAdmin  = "superadmin"             // 超级管理员的 user_id：不查个人权限
	mainMenuAct = "m_sysmenus:getmainmenu" // 取主菜单：登录了就能调，不查权限
)

// -------------------------------------------------------------------------
// 登录令牌（JWT）
// -------------------------------------------------------------------------

// GenerateJWT 签发登录令牌。tokenid 是令牌的编号，登记在 db_sys_tokenlist 的 token_id 列里；token 是回给前端的令牌本身。
func GenerateJWT(userid string, userroles []string, sRemoteAddr string) (tokenid, token string, err error) {
	tokenid, err = utils.GenerateSnowFlakeId()
	if err != nil {
		log.Println(err)
		return "", "", err
	}
	payload := authmodels.LoginClaims{UserId: userid, UserRoles: userroles, RemoteAddr: sRemoteAddr}
	payload.ID = tokenid
	payload.Issuer = authmodels.JwtIssuer
	payload.ExpiresAt = jwt.NewNumericDate(time.Now().Add(TokenLife)) // 定义过期时间
	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, payload).SignedString([]byte(authmodels.SecretKey))
	return tokenid, token, err
}

// ParseJWT 验签并取出令牌里的内容：签名要对，算法是签发时用的 HS256。
// allowExpired 为 false 时还要求是本系统签发的、没有过期；
// 退出登录时传 true：已经过期的令牌也要能把它的在线记录清掉。
func ParseJWT(token string, allowExpired bool) (*authmodels.LoginClaims, error) {
	options := []jwt.ParserOption{jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})}
	if allowExpired {
		options = append(options, jwt.WithoutClaimsValidation())
	} else {
		options = append(options, jwt.WithExpirationRequired(), jwt.WithIssuer(authmodels.JwtIssuer))
	}
	payload := &authmodels.LoginClaims{}
	_, err := jwt.ParseWithClaims(token, payload, func(*jwt.Token) (any, error) {
		return []byte(authmodels.SecretKey), nil
	}, options...)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

// TokenFromRequest 从请求头里取登录令牌：先看 Authorization（可以带 Bearer 前缀），再看 Access-Token。
// 不从 Cookie 和地址参数里取：Cookie 浏览器会自动带上，别的网站的页面能借着登录状态发请求；地址会留在各处的日志里。
func TokenFromRequest(ctx fiber.Ctx) string {
	if token := strings.TrimPrefix(ctx.Get(fiber.HeaderAuthorization), "Bearer "); token != "" {
		return token
	}
	return ctx.Get("Access-Token")
}

// -------------------------------------------------------------------------
// IsAuthenticated 登录校验与权限检查
//
// 由 SysRouter 挂在每一组路由的每个页面上，见 main.go 里的 SysRouter.Auth。
// 注册成"不用登录"的页面（routers.RegisterPublicController，例如登录页）不经过这里。
//
// 一个请求依次过这几关：
//
//	1. 令牌：签名对、没有过期            不对：HTTP 401，令牌无效（-10010）
//	2. 令牌还登记在令牌表里              已经退出或被踢下线：HTTP 401，未找到令牌（-10009）
//	                                     同时记下这个令牌最近一次活动的时间和地址
//	3. 这个操作在菜单表里登记了按钮行    没有登记：未配置权限（-30001）。取主菜单不用登记，登录了就能调
//	4. 按钮行上要求记操作日志的          等请求处理完写一条操作日志（请求体怎么记见 logutils.Body）
//	5. 个人权限：令牌的菜单权限里有它    没有：没有权限（-30002）。超级管理员不查
//
// 权限键是 m_<页面名>:<act>，例如 m_sysdept:add，和菜单表里按钮行的 name 一致。
// -------------------------------------------------------------------------

func IsAuthenticated(ctx fiber.Ctx) error {
	// 路由登记的地址是 <页面地址>/:act?（见 SysRouter.Setup），例如 /web/sysmgr/sysset/sysdept/:act?：
	// 去掉最后一段是页面地址，act 是请求地址里的最后一段
	pagePath := strings.ToLower(path.Dir(ctx.Route().Path))
	act := ctx.Params("act")
	pageAct := "m_" + path.Base(pagePath) + ":" + act

	// 1. 令牌
	payload, err := ParseJWT(TokenFromRequest(ctx), false)
	if err != nil {
		return respUnauthorized(ctx, utils.Resp_code_loginFail10010)
	}

	// 2. 令牌表：一条语句更新最近活动的时间和地址，同时把这一行读回来
	db, err := core.GConfig.GetMainDB()
	if err != nil {
		return err
	}
	ip := ctx.IP() // 只记地址，不带端口；令牌表和操作日志记的都是它
	tokenList := models.SysTokenList{}
	err = db.Get(&tokenList, `update db_sys_tokenlist set updated_at=$1, ip=$2 where token_id=$3
		returning id, token_id, user_type, user_id, roles, menuset, ip, created_at, updated_at`,
		time.Now(), ip, payload.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return respUnauthorized(ctx, utils.Resp_code_loginFail10009)
	}
	if err != nil {
		return err
	}
	pncontext.SetToken(ctx, &tokenList) // 存入页面对象：页面里的 pnctx.TokenList

	// 3. 按钮行
	if pageAct == mainMenuAct {
		return ctx.Next()
	}
	menuAuth := authmodels.SysMenuAuth{}
	err = db.Get(&menuAuth, `select p.id, p.title, p.name as pageact, m.title as menutitle, m.path as menupath, p.loged
		from db_sys_menus p inner join db_sys_menus m on (p.parent_id = m.id)
		where p.menu_type = 3 and p.name = $1`, pageAct)
	if errors.Is(err, sql.ErrNoRows) {
		return pncontext.NewError(utils.Resp_code_authFail30001, utils.RespCodeEnum(utils.Resp_code_authFail30001).CodeParse())
	}
	if err != nil {
		return err
	}
	title := menuAuth.MenuTitle.String + "-" + menuAuth.Title.String // 例如 部门管理-添加

	// 4. 操作日志：等这个请求处理完再写，被第 5 步拒绝的请求也记
	if menuAuth.Loged {
		logItem := logutils.SysOptLogItem{
			UserType:  tokenList.UserType,
			UserId:    tokenList.UserId,
			Type:      "用户操作",
			Events:    act,
			Details:   title,
			PagePath:  pagePath,
			PageQuery: string(ctx.Request().URI().QueryString()),
			PageForm:  logutils.Body(ctx.BodyRaw()), // 请求体的原样，和页面里 pnctx.Bind 读的是同一份；不管 Content-Type 是什么都记
			Ip:        ip,
		}
		defer logutils.DbInsertLogOpt(db, &logItem) // 写失败时它自己记日志
	}

	// 5. 个人权限：超级管理员不查；其他人的菜单权限里要有这个操作，精确匹配
	if tokenList.UserId != superAdmin && !slices.Contains(tokenList.UserMenuSet, pageAct) {
		return pncontext.NewError(utils.Resp_code_authFail30002, "您没有【"+title+"】的操作权限")
	}
	return ctx.Next()
}

// respUnauthorized 令牌有问题时的响应：HTTP 状态码 401，响应体仍然是 {code, message, data}。
func respUnauthorized(ctx fiber.Ctx, code int) error {
	return ctx.Status(fiber.StatusUnauthorized).JSON(pncontext.Resp{Code: code, Message: utils.RespCodeEnum(code).CodeParse()})
}
