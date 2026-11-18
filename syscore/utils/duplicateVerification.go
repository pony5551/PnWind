package utils

import (
	"github.com/pkg/errors"
	"strings"

	"github.com/pony5551/PnWind/core/pncontext"

	pg "github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

// GetVerfication 校验是否是重复插入导致的报错
func GetVerfication(err error) error {
	if err != nil && strings.Contains(err.Error(), "unique") {
		err = errors.Errorf("请勿重复提交!")
	}
	return err
}

// DupErr 违反了唯一约束时，换成说得出原因的提示（-20002，不记日志）。
// msgs 的键是约束名（唯一索引就是索引名），值是回给前端的提示；
// 不是唯一冲突、约束不在 msgs 里的，原样返回，由统一的错误出口记日志、回"操作失败"。
func DupErr(err error, msgs map[string]string) error {
	if pqErr := pg.As(err, pqerror.UniqueViolation); pqErr != nil {
		if msg, isOk := msgs[pqErr.Constraint]; isOk {
			return pncontext.NewError(Resp_code_paramParsel20002, msg)
		}
	}
	return err
}
