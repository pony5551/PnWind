package validation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ParseError 请求体不是合法的 JSON。
type ParseError struct {
	Err error
}

func (e *ParseError) Error() string {
	return "请求体不是合法的 JSON: " + e.Err.Error()
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// FieldError 一个没通过校验的字段。
type FieldError struct {
	Field   string `json:"field"`   // 字段名，取自 json 标签
	Rule    string `json:"rule"`    // 没通过的校验规则，例如 required、max
	Param   string `json:"param"`   // 规则的参数，例如 max=20 里的 20
	Message string `json:"message"` // 中文提示
}

// Error 参数校验失败。Fields 是全部没通过的字段，前端可以据此在对应的表单项上显示提示。
type Error struct {
	Fields []FieldError
}

// Error 返回第一个字段的提示。
func (e *Error) Error() string {
	if len(e.Fields) == 0 {
		return "参数校验失败"
	}
	return e.Fields[0].Message
}

// check 按 validate 标签校验 out 指向的结构体，失败时返回 *Error。不是结构体时不校验。
func (b *binder) check(out any) error {
	t := reflect.TypeOf(out).Elem()
	if t.Kind() != reflect.Struct {
		return nil
	}
	err := b.validate.Struct(out)
	if err == nil {
		return nil
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return fmt.Errorf("参数校验: %w", err)
	}
	fields := make([]FieldError, 0, len(verrs))
	for _, fe := range verrs {
		fields = append(fields, FieldError{
			Field:   fe.Field(),
			Rule:    fe.Tag(),
			Param:   fe.Param(),
			Message: fieldMessage(t, fe),
		})
	}
	return &Error{Fields: fields}
}

// fieldMessage 生成字段的校验提示：优先用字段的 msg 标签；否则用 label 标签（没有时用 json 字段名）
// 加上规则对应的说明。
func fieldMessage(root reflect.Type, fe validator.FieldError) string {
	label := fe.Field()
	if f, ok := structField(root, fe.StructNamespace()); ok {
		if msg := f.Tag.Get("msg"); msg != "" {
			return msg
		}
		if l := f.Tag.Get("label"); l != "" {
			label = l
		}
	}
	return ruleMessage(label, fe.Tag(), fe.Param(), fe.Kind())
}

// structField 按校验器给出的结构体命名空间（例如 UserListReq.PageReq.PageSize）找到对应的字段。
//
// 命名空间的开头是根类型的名字，先把它去掉。处理器里临时定义的匿名结构体没有名字，命名空间里也就没有这一段。
func structField(root reflect.Type, namespace string) (reflect.StructField, bool) {
	if name := root.Name(); name != "" {
		rest, isOk := strings.CutPrefix(namespace, name+".")
		if !isOk {
			return reflect.StructField{}, false
		}
		namespace = rest
	}
	var f reflect.StructField
	t := root
	for _, seg := range strings.Split(namespace, ".") {
		if i := strings.IndexByte(seg, '['); i >= 0 {
			seg = seg[:i]
		}
		for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Map {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return reflect.StructField{}, false
		}
		var ok bool
		if f, ok = t.FieldByName(seg); !ok {
			return reflect.StructField{}, false
		}
		t = f.Type
	}
	return f, true
}

// ruleMessage 返回校验规则对应的中文提示。
func ruleMessage(label, rule, param string, kind reflect.Kind) string {
	sized := kind == reflect.String || kind == reflect.Slice || kind == reflect.Array || kind == reflect.Map
	unit := "项"
	if kind == reflect.String {
		unit = "个字符"
	}
	switch rule {
	case "required", "required_if", "required_unless", "required_with", "required_without":
		return label + "不能为空"
	case "min":
		if sized {
			return label + "至少" + param + unit
		}
		return label + "不能小于" + param
	case "max":
		if sized {
			return label + "最多" + param + unit
		}
		return label + "不能大于" + param
	case "len":
		if sized {
			return label + "必须是" + param + unit
		}
		return label + "必须等于" + param
	case "eq":
		return label + "必须等于" + param
	case "ne":
		return label + "不能等于" + param
	case "gt":
		return label + "必须大于" + param
	case "gte":
		return label + "不能小于" + param
	case "lt":
		return label + "必须小于" + param
	case "lte":
		return label + "不能大于" + param
	case "oneof":
		return label + "必须是以下值之一：" + strings.Join(strings.Fields(param), "、")
	case "email":
		return label + "不是有效的邮箱地址"
	case "url", "http_url":
		return label + "不是有效的网址"
	case "numeric", "number":
		return label + "必须是数字"
	case "alphanum":
		return label + "只能包含字母和数字"
	}
	if param != "" {
		return label + "不符合规则 " + rule + "=" + param
	}
	return label + "不符合规则 " + rule
}

// jsonName 返回字段在 JSON 里的名字，供校验器在错误里使用。
func jsonName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	if name == "-" {
		return ""
	}
	return name
}
