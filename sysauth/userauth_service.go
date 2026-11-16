package sysauth

import (
	"github.com/pony5551/PnWind/core/pncontext"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"github.com/pony5551/PnWind/core/service"
	"github.com/pony5551/PnWind/syscore/utils"

	pg "github.com/lib/pq"
)

type UserAuthService struct {
	service.Service
}

// InsertTokenList 登记一个新令牌，返回这一行的 id。
// 一个账号在令牌表里最多一行（唯一约束 user_type + user_id），这个账号已经有令牌时：
//
//	takeover 为 false：什么都不写，返回 0（登录据此回"已在别处登录"，要二次密码）；
//	takeover 为 true ：把那一行换成新令牌，原来的令牌随即失效，也就是把别处的登录踢掉。
//
// 有没有、换不换都在这一条语句里由库判断，两处同时登录也只会留下一行。
func (c *UserAuthService) InsertTokenList(pnctx *pncontext.PnContext, params map[string]interface{}, takeover bool) (nRet int64, err error) {
	//log.Println(params)
	params["menusets"] = pg.Array(params["menuset"])
	onConflict := "DO NOTHING"
	if takeover {
		onConflict = "DO UPDATE SET token_id=EXCLUDED.token_id, roles=EXCLUDED.roles, menuset=EXCLUDED.menuset, " +
			"ip=EXCLUDED.ip, created_at=EXCLUDED.created_at, updated_at=EXCLUDED.updated_at"
	}
	sqlStr :=
		"INSERT INTO db_sys_tokenlist (" +
			"token_id," +
			"user_type," +
			"user_id," +
			"roles," +
			"menuset," +
			"ip," +
			"created_at" +
			") VALUES (" +
			":token_id," +
			":user_type," +
			":user_id," +
			":roles," +
			":menusets," +
			":ip," +
			":created_at" +
			") ON CONFLICT (user_type, user_id) " + onConflict + " RETURNING id"
	rows, err := pnctx.DB.NamedQuery(sqlStr, params)
	if err != nil {
		//原样记下来:库里没有那个唯一约束时报的错里也带着 unique,下面会把它换成"请勿重复提交",不记的话查不出原因
		log.Println(err)
	}
	//校验是否是重复报错
	err = utils.GetVerfication(err)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var nLastInsertId int64
	if rows.Next() {
		err = rows.Scan(&nLastInsertId)
		if err != nil {
			return 0, err
		}
	}
	return nLastInsertId, err
}
