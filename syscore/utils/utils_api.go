package utils

import (
	"database/sql"
	"github.com/pony5551/PnWind/core/pncontext"
)

type SysSysConfigValue struct {
	Value sql.NullString `db:"value"`
}

func GetWebApiStatus(pnctx *pncontext.PnContext) (string, error) {
	var configVal []SysSysConfigValue
	strSQL := "select value from db_sys_config where sort='web' and key='apistatus'"
	err := pnctx.DB.Select(&configVal, strSQL)
	if err != nil {
		return "", err
	}
	if len(configVal) < 1 {
		return "", nil
	}
	return configVal[0].Value.String, nil
}

func GetWebVar(pnctx *pncontext.PnContext, key string) (string, error) {
	var configVal []SysSysConfigValue
	strSQL := "select value from db_sys_config where sort='web' and key=$1"
	err := pnctx.DB.Select(&configVal, strSQL, key)
	if err != nil {
		return "", err
	}
	if len(configVal) < 1 {
		return "", nil
	}
	return configVal[0].Value.String, nil
}
