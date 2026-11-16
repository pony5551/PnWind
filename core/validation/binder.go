// Package validation 处理请求参数：填写 default 标签的默认值，解析 JSON 请求体，按 validate 标签校验，
// 校验失败时生成中文提示。pnctx.Bind 用的就是这里的 Bind。
package validation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/pony5551/PnWind/core/pnlibs"
)

// std 是全包共用的一个 binder，可以并发使用。
var std = newBinder()

// Bind 依次填写默认值、解析 JSON 请求体、去掉首尾空白、校验参数。out 必须是指针，通常指向一个表单结构体。
//
// 结构体字段上的五个标签：
//
//	default   没传这个字段时的默认值
//	trim      写 trim:"true" 表示先去掉首尾的空白再校验，只能用在字符串字段上。名称、编码这类字段都应当加
//	validate  校验规则，写法见 go-playground/validator
//	label     字段的中文名，用来生成校验提示，例如"部门名称不能为空"
//	msg       直接指定这个字段校验失败时的提示，优先于 label
//
// 请求体不是合法的 JSON 时返回 *ParseError。其余的问题都返回 *Error，里面是没通过的字段：
// 值的类型和字段对不上（例如数字的位置上传了字符串）、字符串里有空字符 \u0000、没通过 validate 的校验。
func Bind(body []byte, out any) error {
	return std.Bind(body, out)
}

// binder 负责填写默认值、解析请求体和校验参数，按类型缓存标签的解析结果。
type binder struct {
	validate *validator.Validate
	plans    *pnlibs.PnObjects[reflect.Type, *plan]
}

// plan 一种结构体上 default、trim 两个标签的解析结果。每种类型只解析一次。
type plan struct {
	defaults []defaultField
	trims    [][]int // 带 trim 标签的字段在结构体里的位置
}

// defaultField 是一个带 default 标签的字段。
type defaultField struct {
	index []int         // 字段在结构体里的位置，包括嵌套结构体的路径
	value reflect.Value // 解析好的默认值
}

func newBinder() *binder {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(jsonName)
	return &binder{
		validate: v,
		plans:    pnlibs.NewPnObjects[reflect.Type, *plan](typeHash),
	}
}

// typeHash 给 PnObjects 分片用：同一个类型的 reflect.Type 背后是同一个指针。
func typeHash(t reflect.Type) uint64 {
	return uint64(reflect.ValueOf(t).Pointer())
}

// Bind 见包级函数 Bind。
func (b *binder) Bind(body []byte, out any) error {
	if v := reflect.ValueOf(out); v.Kind() != reflect.Pointer || v.IsNil() {
		return errors.New("Bind 的参数必须是非空指针，例如 pnctx.Bind(&form)")
	}
	v := reflect.ValueOf(out).Elem()
	p := &plan{}
	if v.Kind() == reflect.Struct {
		var err error
		if p, err = b.planOf(v.Type()); err != nil {
			return err
		}
	}

	// 1. 默认值
	for _, f := range p.defaults {
		v.FieldByIndex(f.index).Set(f.value)
	}

	// 2. 解析
	if len(body) > 0 {
		if hasNUL(body) {
			// PostgreSQL 的文本列存不了空字符，放过去只会在数据库那里报错
			return &Error{Fields: []FieldError{{Rule: "nul", Message: `参数里不能有空字符（\u0000）`}}}
		}
		if err := json.Unmarshal(body, out); err != nil {
			var typeErr *json.UnmarshalTypeError
			if errors.As(err, &typeErr) && typeErr.Field != "" {
				return typeMismatch(v.Type(), typeErr)
			}
			return &ParseError{Err: err}
		}
	}

	// 3. 去掉首尾空白
	for _, index := range p.trims {
		f := v.FieldByIndex(index)
		f.SetString(strings.TrimSpace(f.String()))
	}

	// 4. 校验
	return b.check(out)
}

// planOf 返回 t 上 default、trim 标签的解析结果。每个类型只解析一次，之后走缓存。
func (b *binder) planOf(t reflect.Type) (*plan, error) {
	if p, isOk := b.plans.Load(t); isOk {
		return p, nil
	}
	p := &plan{}
	if err := collectTags(t, nil, p); err != nil {
		return nil, err
	}
	b.plans.Store(t, p)
	return p, nil
}

// collectTags 收集 t 里所有带 default、trim 标签的字段。嵌套和嵌入的结构体会递归处理。
// default 标签只能用在字符串、布尔、整数、浮点数和时长字段上，trim 标签只能用在字符串字段上，
// 用在其他类型上会报错，以便尽早发现写错的标签。
func collectTags(t reflect.Type, prefix []int, p *plan) error {
	for i := range t.NumField() {
		f := t.Field(i)
		index := append(slices.Clone(prefix), i)
		isStruct := f.Type.Kind() == reflect.Struct && f.Type != reflect.TypeFor[time.Time]()
		switch {
		case isStruct && (f.IsExported() || f.Anonymous):
			if err := collectTags(f.Type, index, p); err != nil {
				return err
			}
			continue
		case !f.IsExported():
			continue
		}
		// 这两个标签只在直接嵌套的结构体里生效。写在切片、指针、map 里的结构体上不会生效，报出来免得白写
		if elem := elemStruct(f.Type); elem != nil && hasBindTags(elem, map[reflect.Type]bool{}) {
			return fmt.Errorf("%s.%s 里的结构体带 default 或 trim 标签，但它在切片、指针或 map 里，这两个标签不会生效：改成直接嵌套的结构体，或者去掉标签", t.Name(), f.Name)
		}
		if tag, isOk := f.Tag.Lookup("default"); isOk {
			value, err := parseDefault(f.Type, tag)
			if err != nil {
				return fmt.Errorf("%s.%s 的 default 标签 %q 不合法: %w", t.Name(), f.Name, tag, err)
			}
			p.defaults = append(p.defaults, defaultField{index: index, value: value})
		}
		if tag, isOk := f.Tag.Lookup("trim"); isOk {
			if tag != "true" || f.Type.Kind() != reflect.String {
				return fmt.Errorf(`%s.%s 的 trim 标签 %q 不合法: 只能写 trim:"true"，并且只能用在字符串字段上`, t.Name(), f.Name, tag)
			}
			p.trims = append(p.trims, index)
		}
	}
	return nil
}

// elemStruct 返回切片、数组、指针、map 最里面的元素类型；元素不是结构体，或者 t 本身不是这几种类型时返回 nil。
func elemStruct(t reflect.Type) reflect.Type {
	isContainer := false
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Map {
		t, isContainer = t.Elem(), true
	}
	if !isContainer || t.Kind() != reflect.Struct || t == reflect.TypeFor[time.Time]() {
		return nil
	}
	return t
}

// hasBindTags 判断结构体 t 里（包括它嵌套的结构体）有没有 default 或 trim 标签。seen 用来防止自己套自己的类型查不完。
func hasBindTags(t reflect.Type, seen map[reflect.Type]bool) bool {
	if seen[t] {
		return false
	}
	seen[t] = true
	for i := range t.NumField() {
		f := t.Field(i)
		if _, isOk := f.Tag.Lookup("default"); isOk {
			return true
		}
		if _, isOk := f.Tag.Lookup("trim"); isOk {
			return true
		}
		inner := f.Type
		if elem := elemStruct(inner); elem != nil {
			inner = elem
		}
		if inner.Kind() == reflect.Struct && inner != reflect.TypeFor[time.Time]() && hasBindTags(inner, seen) {
			return true
		}
	}
	return false
}

// hasNUL 判断 JSON 请求体的字符串里有没有空字符。空字符在 JSON 里只能写成转义 \u0000。
func hasNUL(body []byte) bool {
	for i := 0; ; i++ {
		n := bytes.Index(body[i:], []byte("u0000"))
		if n < 0 {
			return false
		}
		i += n
		// u 前面连续的反斜杠是奇数个，才是转义 \u0000；偶数个是"转义过的反斜杠后面跟着字面的 u0000"
		slashes := 0
		for j := i - 1; j >= 0 && body[j] == '\\'; j-- {
			slashes++
		}
		if slashes%2 == 1 {
			return true
		}
	}
}

// typeMismatch 把"请求里某个值的类型和字段对不上"转成按字段的提示，
// 例如排序号的位置上传了 1.5 或者字符串，提示"排序的格式不对"。
func typeMismatch(root reflect.Type, e *json.UnmarshalTypeError) error {
	field, message := e.Field, e.Field+"的格式不对"
	if f, jsonPath, isOk := fieldByJSONPath(root, e.Field); isOk {
		field, message = jsonPath, jsonPath+"的格式不对"
		if msg := f.Tag.Get("msg"); msg != "" {
			message = msg
		} else if label := f.Tag.Get("label"); label != "" {
			message = label + "的格式不对"
		}
	}
	return &Error{Fields: []FieldError{{Field: field, Rule: "type", Message: message}}}
}

// fieldByJSONPath 按 encoding/json 报错里的字段路径（例如 inner.level）找到结构体上对应的字段。
//
// encoding/json 给的路径里带着嵌入结构体的名字（例如 PageReq.pageSize），而 JSON 里并没有这一层，
// 所以另外返回去掉这些名字之后的路径，回给前端用。
func fieldByJSONPath(root reflect.Type, path string) (f reflect.StructField, jsonPath string, isOk bool) {
	t := root
	var segs []string
	for seg := range strings.SplitSeq(path, ".") {
		for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Map {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return f, path, false
		}
		if f, isOk = fieldByJSONName(t, seg); !isOk {
			return f, path, false
		}
		if !f.Anonymous || jsonName(f) != "" {
			segs = append(segs, seg)
		}
		t = f.Type
	}
	return f, strings.Join(segs, "."), true
}

// fieldByJSONName 在结构体 t 上找 JSON 名字是 name 的字段；没有 json 标签的字段按 Go 的字段名找。
func fieldByJSONName(t reflect.Type, name string) (reflect.StructField, bool) {
	for i := range t.NumField() {
		f := t.Field(i)
		if tagName := jsonName(f); tagName == name || (tagName == "" && f.Name == name) {
			return f, true
		}
	}
	return reflect.StructField{}, false
}

// parseDefault 把 default 标签解析成字段类型的值。
func parseDefault(t reflect.Type, s string) (reflect.Value, error) {
	v := reflect.New(t).Elem()
	switch t.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return v, err
		}
		v.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if t == reflect.TypeFor[time.Duration]() {
			d, err := time.ParseDuration(s)
			if err != nil {
				return v, err
			}
			v.SetInt(int64(d))
			return v, nil
		}
		n, err := strconv.ParseInt(s, 10, t.Bits())
		if err != nil {
			return v, err
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, t.Bits())
		if err != nil {
			return v, err
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, t.Bits())
		if err != nil {
			return v, err
		}
		v.SetFloat(n)
	default:
		return v, fmt.Errorf("不支持 %s 类型的字段", t)
	}
	return v, nil
}
