package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/pony5551/PnWind/core"

	"github.com/bwmarrin/snowflake"
)

var (
	node *snowflake.Node
)

//// 创建订单id,Twitter 雪花算法
//func GenerateSnowFlakeId(nodeid int64) (string, error) {
//	if node == nil {
//		// Create a new Node with a Node number of 1
//		var err error
//		node, err = snowflake.NewNode(nodeid)
//		if err != nil {
//			return "", err
//		}
//	}
//
//	//timeUnixNano := time.Now().UnixNano() //单位纳秒,打印结果：1491888244752784461
//	//timeUnixNanoStr := fmt.Sprintf("%d", timeUnixNano)
//	////log.Println(timeUnixNanoStr)
//	//nlen := len(timeUnixNanoStr)
//	//if nlen > 10 {
//	//	timeUnixNanoStr = timeUnixNanoStr[nlen-10:]
//	//}
//	////log.Println(timeUnixNanoStr)
//
//	//sRandNum := fmt.Sprintf("%08v", rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(10000000000))
//
//	var nMin int64 = 100000
//	var nMax int64 = 999999
//	nRandInt, _ := rand.Int(rand.Reader, big.NewInt(nMax-nMin))
//	//log.Println(nRandInt.Int64())
//	nRandInt64 := nRandInt.Int64() + nMin
//	//log.Println(nRandInt64)
//
//	sRandNum := fmt.Sprintf("%d", nRandInt64)
//
//	// Generate a snowflake ID.
//	id := fmt.Sprintf("%s%s", node.Generate().String(), sRandNum)
//	return id, nil
//
//	//return id, nil
//}

// 雪花id+6位随机数
func GenerateSnowFlakeId() (string, error) {
	var nMin int64 = 100000
	var nMax int64 = 999999
	nRandInt, _ := rand.Int(rand.Reader, big.NewInt(nMax-nMin))
	nRandInt64 := nRandInt.Int64() + nMin
	sRandNum := fmt.Sprintf("%d", nRandInt64)

	// Generate a snowflake ID.
	id, err := core.GConfig.IdGen.Generate()
	if err != nil {
		return "", err
	}
	idStr := fmt.Sprintf("%d%s", id, sRandNum)
	return idStr, nil
}
