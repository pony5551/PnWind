package PnWebSocket

import (
	"github.com/pony5551/PnWind/core"
	"github.com/pony5551/PnWind/core/pnlibs"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"github.com/pony5551/PnWind/core/util"
	authmodels "github.com/pony5551/PnWind/sysauth/models"
	"github.com/pony5551/PnWind/syscore/utils"
	"github.com/pony5551/PnWind/sysmgr/models"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/golang-jwt/jwt/v5"
)

const SecWebsocketProtocol = "Sec-Websocket-Protocol"

type client struct {
	Key  string
	Conn *websocket.Conn
} // Add more data to this type if needed

type PnChatCenter struct {
	//var clients = make(map[*websocket.Conn]client) // Note: although large maps with pointer-like types (e.g. strings) as keys are slow, using pointers themselves as keys is acceptable and fast
	//var clientsLock sync.RWMutex = sync.RWMutex{}
	//var clients = make(map[string]client)
	//var register = make(chan *websocket.Conn)
	//var broadcast = make(chan string)
	//var unregister = make(chan *websocket.Conn)

	//clientsLock sync.RWMutex
	clients    *pnlibs.PnObjectsEx[string, interface{}]
	register   chan *websocket.Conn
	broadcast  chan string
	unregister chan *websocket.Conn
}

func NewPnChatCenter() *PnChatCenter {
	chatCenter := PnChatCenter{
		//clientsLock: sync.RWMutex{},
		clients:    pnlibs.NewPnObjectsEx[string, interface{}](),
		register:   make(chan *websocket.Conn),
		broadcast:  make(chan string),
		unregister: make(chan *websocket.Conn),
	}
	return &chatCenter
}

func (c *PnChatCenter) Reg(sKey string, connection *websocket.Conn) {
	log.Println("Reg:", sKey)
	cli := client{
		Key:  sKey,
		Conn: connection,
	}
	c.clients.Store(sKey, &cli)
}

func (c *PnChatCenter) UnReg(sKey string) {
	log.Println("UnReg:", sKey)
	c.clients.Delete(sKey)
}

//MsgInfo["wstype"] = 2
//MsgInfo["msgtype"] = "sysnotice"
//MsgInfo["msgevent"] = "login"
//MsgInfo["msgdata"] = userInfo
//MsgInfo["sender"] = "system"
//MsgInfo["target"] = WsConn.Key
/*
wstype:
1:点对点
2:广播

msgtype:
sysnotice
sysmail
usermsg 用户消息

msgevent:
login 登录
logout 登出
chatmsg 聊天消息
onlinelist 在线列表

sender: 发送人
target: 接受人
target=all 所有人

msgdata: 消息内容
*/

func (c *PnChatCenter) WsSendMsg(msgtype, msgevent, sender, target string, msgData interface{}) {
	var message string
	if target == "all" {
		//发送用户登录信息给所有人
		c.clients.Range(func(key string, value interface{}) bool {
			//log.Println("WsConn: ", WsConn)
			//log.Println("WsConn: ", WsConn)
			cli := value.(*client)
			userInfo := make(map[string]interface{})
			if msgevent == "login" {
				userInfo = msgData.(map[string]interface{})
				usersid := userInfo["usersid"].(string)
				userInfo["self"] = cli.Key == usersid
				msgData = userInfo
			}
			MsgInfo := make(map[string]interface{})
			MsgInfo["msgtype"] = msgtype
			MsgInfo["msgevent"] = msgevent
			MsgInfo["msgdata"] = msgData
			MsgInfo["sender"] = sender
			MsgInfo["target"] = target
			message, _ = util.ToJSON(MsgInfo)
			if err := cli.Conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				log.Println("write error:", err)
				cli.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				cli.Conn.Close()
				go func() {
					c.clients.Delete(cli.Key)
				}()
			}
			return true
		})

		//c.clients.Lock.RLock()
		//LClients := c.clients.Objs
		//c.clients.Lock.RUnlock()
		//for _, WsConn := range LClients {
		//	//log.Println("WsConn: ", WsConn)
		//	cli := WsConn.(*client)
		//	userInfo := make(map[string]interface{})
		//	if msgevent == "login" {
		//		userInfo = msgData.(map[string]interface{})
		//		usersid := userInfo["usersid"].(string)
		//		userInfo["self"] = cli.Key == usersid
		//		msgData = userInfo
		//	}
		//	MsgInfo := make(map[string]interface{})
		//	MsgInfo["msgtype"] = msgtype
		//	MsgInfo["msgevent"] = msgevent
		//	MsgInfo["msgdata"] = msgData
		//	MsgInfo["sender"] = sender
		//	MsgInfo["target"] = target
		//	message, _ = util.ToJSON(MsgInfo)
		//	if err := cli.Conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		//		log.Println("write error:", err)
		//		cli.Conn.WriteMessage(websocket.CloseMessage, []byte{})
		//		cli.Conn.Close()
		//		c.clients.Del(cli.Key)
		//	}
		//}
	} else {
		v, ok := c.clients.Load(target)
		if !ok {
			return
		}
		cli := v.(*client)
		MsgInfo := make(map[string]interface{})
		MsgInfo["msgtype"] = msgtype
		MsgInfo["msgevent"] = msgevent
		MsgInfo["msgdata"] = msgData
		MsgInfo["sender"] = sender
		MsgInfo["target"] = target
		message, _ = util.ToJSON(MsgInfo)
		if err := cli.Conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
			log.Println("write error:", err)
			cli.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			cli.Conn.Close()
			c.clients.Delete(cli.Key)
		}
		//给发送人推消息
		v, ok = c.clients.Load(sender)
		if ok && v != nil {
			cli2 := v.(*client)
			if err := cli2.Conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				log.Println("write error:", err)
				cli2.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				cli2.Conn.Close()
				c.clients.Delete(cli2.Key)
			}
		}
	}
}

func (c *PnChatCenter) WsGetOnlineList() map[string]interface{} {
	userOnlineList := make(map[string]interface{})
	var onlinelist []string

	c.clients.Range(func(key string, value interface{}) bool {
		cli := value.(*client)
		onlinelist = append(onlinelist, cli.Key)
		return true
	})

	//c.clients.Lock.RLock()
	//defer c.clients.Lock.RUnlock()
	//for _, WsConn := range c.clients.Objs {
	//	cli := WsConn.(*client)
	//	onlinelist = append(onlinelist, cli.Key)
	//}

	userOnlineList["onlinelist"] = onlinelist
	return userOnlineList
}

func (c *PnChatCenter) runHub() {
	for {
		select {
		//赋值 register <- c
		//读取 <-register
		case connection := <-c.register:
			//log.Println("connection registered")
			tokenList := connection.Locals("tokenList").(models.SysTokenList)
			sKey := fmt.Sprintf("%s%d", tokenList.UserId, tokenList.Id)
			c.Reg(sKey, connection)

			//发送用户登录信息给所有人
			userInfo := make(map[string]interface{})
			userInfo["usersid"] = sKey
			userInfo["UserId"] = tokenList.UserId
			userInfo["Roles"] = tokenList.Roles
			//userInfo["ParentRoleCode"] = tokenList.ParentRoleCode
			userInfo["self"] = false
			c.WsSendMsg("sysnotice", "login", "system", "all", userInfo)

			userOnlineList := c.WsGetOnlineList()
			c.WsSendMsg("sysnotice", "onlinelist", "system", "all", userOnlineList)

		case message := <-c.broadcast:
			log.Println("message received:", message)

			MsgInfo := make(map[string]interface{})
			err := json.Unmarshal([]byte(message), &MsgInfo)
			if err != nil {
				log.Errorln(err)
			}

			if _, ok := MsgInfo["msgtype"]; !ok {
				log.Errorln("未找到 msgtype")
				continue
			}
			if _, ok := MsgInfo["msgevent"]; !ok {
				log.Errorln("未找到 msgevent")
				continue
			}
			if _, ok := MsgInfo["sender"]; !ok {
				log.Errorln("未找到 sender")
				continue
			}
			if _, ok := MsgInfo["target"]; !ok {
				log.Errorln("未找到 target")
				continue
			}
			if _, ok := MsgInfo["msgdata"]; !ok {
				log.Errorln("未找到 msgdata")
				continue
			}

			msgtype := MsgInfo["msgtype"].(string)
			msgevent := MsgInfo["msgevent"].(string)
			sender := MsgInfo["sender"].(string)
			target := MsgInfo["target"].(string)
			msgdata := MsgInfo["msgdata"].(string)
			c.WsSendMsg(msgtype, msgevent, sender, target, msgdata)

		case connection := <-c.unregister:
			//log.Println("connection unregistered")

			// Remove the client from the hub
			tokenList := connection.Locals("tokenList").(models.SysTokenList)
			sKey := fmt.Sprintf("%s%d", tokenList.UserId, tokenList.Id)
			c.UnReg(sKey)

			//发送用户登出信息给所有人
			userInfo := make(map[string]interface{})
			userInfo["usersid"] = sKey
			userInfo["UserId"] = tokenList.UserId
			userInfo["Roles"] = tokenList.Roles
			//userInfo["ParentRoleCode"] = tokenList.ParentRoleCode
			c.WsSendMsg("sysnotice", "logout", "system", "all", userInfo)

			userOnlineList := c.WsGetOnlineList()
			c.WsSendMsg("sysnotice", "onlinelist", "system", "all", userOnlineList)
		}
	}
}

func (c *PnChatCenter) UseWebSocket(app *fiber.App) {
	app.Use("/ws", func(ctx fiber.Ctx) error {
		var AccessToken string
		ReqHeaders := ctx.GetReqHeaders()
		//log.Println("ReqHeaders: ", ReqHeaders)
		if _, ok := ReqHeaders[SecWebsocketProtocol]; ok {
			AccessToken = ReqHeaders[SecWebsocketProtocol][0]
		}
		//log.Println("AccessToken: ", AccessToken)
		token, err := jwt.ParseWithClaims(AccessToken, &authmodels.LoginClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(authmodels.SecretKey), nil
		})
		if err != nil || !token.Valid {
			pageResp := fiber.Map{
				"code":    utils.Resp_code_loginFail10010,
				"message": utils.RespCodeEnum(utils.Resp_code_loginFail10010).CodeParse(),
			}
			return ctx.Status(fiber.StatusUnauthorized).JSON(pageResp)
		}

		payload := token.Claims.(*authmodels.LoginClaims)
		//log.Println("payload:================", payload)
		//读取tokenlist
		db, err := core.GConfig.GetMainDB()
		if err != nil {
			return err
		}
		tokenList := models.SysTokenList{}
		err = db.Get(&tokenList, "select * from db_sys_tokenlist where token_id=$1", payload.ID)
		if err != nil {
			//log.Println(err)
			pageResp := fiber.Map{
				"code":    utils.Resp_code_loginFail10009,
				"message": utils.RespCodeEnum(utils.Resp_code_loginFail10009).CodeParse(),
			}
			return ctx.Status(fiber.StatusUnauthorized).JSON(pageResp)
		}

		//更新tokenList
		tokenList.UpdatedAt = time.Now()
		tokenList.Ip = ctx.IP() //只记地址,不带端口;和鉴权中间件是同一个取法
		_, err = db.Exec("update db_sys_tokenlist set updated_at=$1,ip=$2 where token_id=$3", time.Now(), tokenList.Ip, payload.ID)
		if err != nil {
			//log.Println(err)
			pageResp := fiber.Map{
				"code":    utils.Resp_code_loginFail10009,
				"message": utils.RespCodeEnum(utils.Resp_code_loginFail10009).CodeParse(),
			}
			return ctx.Status(fiber.StatusUnauthorized).JSON(pageResp)
		}

		//ws不需要页面权限判断
		tokenList.UserMenuSet = []string{}
		//tokenList.UserPowerSet = []string{}

		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(ctx) {
			//SecWebsocketProtocol
			ctx.Response().Header.Set(SecWebsocketProtocol, AccessToken)
			ctx.Locals("allowed", true)
			ctx.Locals("tokenList", tokenList)
			return ctx.Next()
		}

		//return fiber.ErrUpgradeRequired
		return ctx.SendStatus(fiber.StatusUpgradeRequired)
	})

	go c.runHub()

	app.Get("/ws", websocket.New(func(con *websocket.Conn) {
		// When the function returns, unregister the client and close the connection
		defer func() {
			c.unregister <- con
			con.Close()
		}()

		// Register the client
		c.register <- con

		for {
			messageType, message, err := con.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Println("read error:", err)
				}
				return // Calls the deferred function, i.e. closes the connection on error
			}

			if messageType == websocket.TextMessage {
				// Broadcast the received message
				c.broadcast <- string(message)
			} else {
				log.Println("websocket message received of type", messageType)
			}
		}
	}))

}
