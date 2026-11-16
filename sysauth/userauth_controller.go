package sysauth

import (
	"strings"
	"time"

	"github.com/pony5551/PnWind/core"
	"github.com/pony5551/PnWind/core/controller"
	"github.com/pony5551/PnWind/core/pncontext"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"github.com/pony5551/PnWind/core/util"
	"github.com/pony5551/PnWind/syscore/middleware"
	"github.com/pony5551/PnWind/syscore/routers"
	"github.com/pony5551/PnWind/syscore/utils"

	"github.com/gofiber/fiber/v3"
	pg "github.com/lib/pq"
)

type UserAuthCont struct {
	controller.ControllerBase
	SVC UserAuthService
}

func (c *UserAuthCont) setMerge(set1, set2 string) string {
	set1List := strings.Split(set1, ",")
	set2List := strings.Split(set2, ",")
	set2List = append(set2List, set1List...)
	set2List = util.RemoveRepeatElement(set2List)
	return strings.Join(set2List, ",")
}

func (c *UserAuthCont) setMerge2(set1, set2 []string) []string {
	setList := []string{}
	setList = append(setList, set1...)
	setList = append(setList, set2...)
	setList = util.RemoveRepeatElement(setList)
	return setList
}

func (c *UserAuthCont) Page_Login(pnctx *pncontext.PnContext) error {
	db := pnctx.DB
	if pnctx.DB == nil {
		return pnctx.RespErr(utils.Resp_code_loginFail10006, utils.RespCodeEnum(utils.Resp_code_loginFail10006).CodeParse())
	}

	type UserLogindata struct {
		Username  string `json:"username" xml:"username" form:"username"`
		Password  string `json:"password" xml:"password" form:"password"`
		Password2 string `json:"password2"` //二次密码:这个账号已经在别处登录着才要带,和password一样是三重MD5之后的值
		Code      string `json:"code" xml:"code" form:"code"`
	}

	uLogindata := new(UserLogindata)
	if err := pnctx.Bind(uLogindata); err != nil {
		return err
	}

	//验证码检测
	var err error
	var user SysAdmin
	//已删除用户(deleted_at不为空)不允许登录,统一返回"用户不存在"
	err = db.Get(&user, "select user_id, username, password, password2, avatar, roles, status from db_sys_admin where ((user_id=$1) or (username=$1) or (phone=$1) or (email=$1)) and deleted_at is null", uLogindata.Username)
	if err != nil {
		log.Println(err)
		return pnctx.RespErr(utils.Resp_code_loginFail10002, utils.RespCodeEnum(utils.Resp_code_loginFail10002).CodeParse())
	}
	//log.Println(user)

	//状态检测
	if user.Status == -1 {
		return pnctx.RespErr(utils.Resp_code_loginFail10031, utils.RespCodeEnum(utils.Resp_code_loginFail10031).CodeParse())
	} else if user.Status == -2 {
		return pnctx.RespErr(utils.Resp_code_loginFail10032, utils.RespCodeEnum(utils.Resp_code_loginFail10032).CodeParse())
	} else if user.Status == 0 {
		//停用:管理页状态开关(1启用/0停用)关闭的用户不允许登录
		return pnctx.RespErr(utils.Resp_code_loginFail10033, utils.RespCodeEnum(utils.Resp_code_loginFail10033).CodeParse())
	}

	//密码检测
	if uLogindata.Password != user.Password {
		return pnctx.RespErr(utils.Resp_code_loginFail10030, utils.RespCodeEnum(utils.Resp_code_loginFail10030).CodeParse())
	}

	//二次密码:带了就必须对,对了这次登录才可以把别处的登录踢掉(排在登录密码之后:密码不对的人探不出别的)
	//没有单独设置过(空串)的账号,二次密码就是登录密码
	takeover := uLogindata.Password2 != ""
	if takeover {
		password2 := user.Password2
		if password2 == "" {
			password2 = user.Password
		}
		if uLogindata.Password2 != password2 {
			return pnctx.RespErr(utils.Resp_code_loginFail10035, utils.RespCodeEnum(utils.Resp_code_loginFail10035).CodeParse())
		}
	}

	//通过检测
	//读取菜单与权限:角色编码只作为参数传给数据库,不拼进SQL(编码里带引号也只是查不到这个角色)
	//没有角色的用户不能登录(回"角色不存在")
	if len(user.Roles) == 0 {
		return pnctx.RespErr(utils.Resp_code_loginFail10005, utils.RespCodeEnum(utils.Resp_code_loginFail10005).CodeParse())
	}
	var roleMenuSets []pg.StringArray
	err = db.Select(&roleMenuSets, "select menuset from db_sys_role where code = any($1) order by id", user.Roles)
	if err != nil {
		log.Println(err)
		return pnctx.RespErr(utils.Resp_code_loginFail10005, utils.RespCodeEnum(utils.Resp_code_loginFail10005).CodeParse())
	}

	//合并菜单
	sMenuSets := []string{}
	for _, menuSet := range roleMenuSets {
		sMenuSets = c.setMerge2(sMenuSets, menuSet)
	}

	//创建JWT
	tokenid, token, err := middleware.GenerateJWT(user.UserId, user.Roles, pnctx.Context().RemoteAddr().String())
	if err != nil {
		return pnctx.RespErr(utils.Resp_code_loginFail10004, utils.RespCodeEnum(utils.Resp_code_loginFail10004).CodeParse())
	}

	//顺手清掉已经过期的令牌行:令牌只管TokenLife这么久,登录超过这个时间的行已经没有用了;清不掉只记日志,不耽误登录
	//要排在登记新令牌之前:过期的行不算"已经登录着"
	if _, err := db.Exec("delete from db_sys_tokenlist where created_at < $1", time.Now().Add(-middleware.TokenLife)); err != nil {
		log.Println(err)
	}

	//写入数据库
	ip := pnctx.IP() //只记地址,不带端口;和鉴权中间件、created_ip是同一个取法
	tokenParams := make(map[string]interface{})
	tokenParams["token_id"] = tokenid
	tokenParams["user_type"] = 1 //1=管理员,2=用户;这里登录的都是管理员表里的人,操作日志的user_type取的就是它
	tokenParams["user_id"] = user.UserId
	tokenParams["roles"] = user.Roles
	tokenParams["menuset"] = sMenuSets
	tokenParams["ip"] = ip
	tokenParams["created_at"] = time.Now()

	retTokenid, err := c.SVC.InsertTokenList(pnctx, tokenParams, takeover)
	if err != nil {
		return pnctx.RespErr(utils.Resp_code_loginFail10004, utils.RespCodeEnum(utils.Resp_code_loginFail10004).CodeParse())
	}
	if retTokenid == 0 {
		//一个账号只能有一处登录:这个账号已经在别处登录着,没有登记新令牌。前端据此要二次密码,带上它再登录一次
		return pnctx.RespErr(utils.Resp_code_loginFail10034, utils.RespCodeEnum(utils.Resp_code_loginFail10034).CodeParse())
	}

	//更新登录:令牌这时已经登记好了(别处的登录也可能已经被换掉),这一步只是给页面看的,出错只记日志,不让这次登录失败。
	//不然会出现"原来的人被踢了、新的人也没登上",或者留下一行谁都没拿到的令牌、下次登录平白要二次密码
	strSQL := "update db_sys_admin set last_login_ip=$1,last_login_at=$2 where user_id=$3"
	if _, err = db.Exec(strSQL, ip, time.Now(), user.UserId); err != nil {
		log.Println(err)
	}

	////写入cookie
	//cookie := fiber.Cookie{
	//	Name:    "jwt",
	//	Value:   token,
	//	Expires: time.Now().Add(time.Hour * 24),
	//	//Expires:  time.Now().Add(time.Minute * 24),
	//	HTTPOnly: true,
	//}
	//ctx.Cookie(&cookie)

	roles := []string{}
	rolesPlus := []string{"admin", "r_Admin"}
	roles = c.setMerge2(user.Roles, rolesPlus)
	return pnctx.RespOK(fiber.Map{
		"avatar":       user.Avatar,   //头像
		"userid":       user.UserId,   //用户id
		"username":     user.Username, //用户名
		"nickname":     user.Username, //昵称
		"roles":        roles,
		"permissions":  []string{"*:*:*"},
		"accessToken":  token,
		"refreshToken": token,
		"expires":      time.Now().Add(middleware.TokenLife).Format("2006-01-02 15:04:05"), //和令牌本身的有效期是同一个值
		"tokenid":      retTokenid,
		"MenuSets":     sMenuSets,
	}, "登录成功")
}

func (c *UserAuthCont) Page_Logout(pnctx *pncontext.PnContext) error {
	//删除db_sys_tokenlist在线记录使令牌立即失效
	//取令牌、验签和IsAuthenticated中间件用的是同一套(middleware.TokenFromRequest、ParseJWT);
	//签名必须有效;已过期的令牌也能完成清理;解析失败不影响登出
	if payload, err := middleware.ParseJWT(middleware.TokenFromRequest(pnctx.Ctx), true); err == nil && payload.ID != "" && pnctx.DB != nil {
		if _, err := pnctx.DB.Exec("delete from db_sys_tokenlist where token_id=$1", payload.ID); err != nil {
			log.Println(err)
		}
	}

	pnctx.ClearCookie()
	return pnctx.RespOK(nil, "退出成功")
}

func init() {
	pagePath := "/sysauth/auth"
	LController := &UserAuthCont{}
	//fmt.Println(reflect.TypeOf(*LController).PkgPath() + " ---> init")
	//登录、退出不用登录就能调：注册成不用登录的页面
	routers.RegisterPublicController(core.GConfig, "web", pagePath, LController)
}
