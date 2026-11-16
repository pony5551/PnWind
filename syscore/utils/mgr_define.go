package utils

type RespCodeEnum int

const (
	Resp_code_loginFail10001 = -10001 //验证码未通过
	Resp_code_loginFail10002 = -10002 //用户不存在
	Resp_code_loginFail10030 = -10030 //用户密码错误
	Resp_code_loginFail10031 = -10031 //用户被禁用
	Resp_code_loginFail10032 = -10032 //用户被冻结
	Resp_code_loginFail10033 = -10033 //用户已停用
	Resp_code_loginFail10034 = -10034 //该账号已在别处登录，要带二次密码再登录
	Resp_code_loginFail10035 = -10035 //二次密码错误
	Resp_code_loginFail10004 = -10004 //生成token错误
	Resp_code_loginFail10005 = -10005 //角色不存在
	Resp_code_loginFail10006 = -10006 //数据库未连接
	Resp_code_loginFail10007 = -10007 //数据库读取出错
	Resp_code_loginFail10008 = -10008 //数据库查询错误

	Resp_code_loginFail10009 = -10009 //未找到令牌
	Resp_code_loginFail10010 = -10010 //令牌无效

	Resp_code_paramParsel20001 = -20001 //参数解析错误
	Resp_code_paramParsel20002 = -20002 //参数验证错误

	Resp_code_authFail30001 = -30001 //未配置权限
	Resp_code_authFail30002 = -30002 //没有权限

	Resp_code_OptFail90001 = -90001 //系统未开放删除数据
	Resp_code_OptFail90002 = -90002 //操作失败
	Resp_code_OptFail99999 = -99999 //未知的请求
)

func (c RespCodeEnum) CodeParse() (msg string) {
	switch c {
	case Resp_code_loginFail10001:
		return "验证码未通过"
	case Resp_code_loginFail10002:
		return "用户不存在"
	case Resp_code_loginFail10030:
		return "用户密码错误"
	case Resp_code_loginFail10031:
		return "用户被禁用"
	case Resp_code_loginFail10032:
		return "用户被冻结"
	case Resp_code_loginFail10033:
		return "用户已停用"
	case Resp_code_loginFail10034:
		return "该账号已在别处登录，请输入二次密码"
	case Resp_code_loginFail10035:
		return "二次密码错误"
	case Resp_code_loginFail10004:
		return "生成token错误"
	case Resp_code_loginFail10005:
		return "角色不存在"
	case Resp_code_loginFail10006:
		return "数据库未连接"
	case Resp_code_loginFail10007:
		return "数据库读取出错"
	case Resp_code_loginFail10008:
		return "数据库查询错误"
	case Resp_code_loginFail10009:
		return "未找到令牌"
	case Resp_code_loginFail10010:
		return "令牌无效"

	case Resp_code_paramParsel20001:
		return "参数解析错误"
	case Resp_code_paramParsel20002:
		return "参数验证错误"

	case Resp_code_authFail30001:
		return "未配置权限"
	case Resp_code_authFail30002:
		return "没有权限"

	case Resp_code_OptFail90001:
		return "系统未开放删除数据"
	case Resp_code_OptFail90002:
		return "操作失败"
	case Resp_code_OptFail99999:
		return "未知的请求"

	default:
		return "UnKnow"
	}
}
