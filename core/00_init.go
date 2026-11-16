package core

import (
	"fmt"
	log "github.com/pony5551/PnWind/core/pnlogrus"
)

func init() {
	fmt.Println("init system")
	log.InitZapLog(false)
}
