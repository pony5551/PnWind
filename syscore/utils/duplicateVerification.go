package utils

import (
	"github.com/pkg/errors"
	"strings"
)

/**
校验是否是重复插入导致的报错
*/
func GetVerfication(err error) error {
	if err != nil && strings.Contains(err.Error(), "unique") {
		err = errors.Errorf("请勿重复提交!")
	}
	return err
}
