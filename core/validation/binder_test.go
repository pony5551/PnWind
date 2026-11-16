package validation

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func newTestBinder() *binder {
	return newBinder()
}

type pageReq struct {
	Page     int `json:"page"     default:"1"`
	PageSize int `json:"pageSize" default:"20" label:"每页条数" validate:"lte=100"`
}

type defaultsReq struct {
	pageReq
	Name    string        `json:"name"  default:"匿名"`
	Ratio   float64       `json:"ratio" default:"0.5"`
	Enabled bool          `json:"on"    default:"true"`
	Limit   uint8         `json:"limit" default:"8"`
	Wait    time.Duration `json:"wait"  default:"1m"`
	Inner   struct {
		Level int `json:"level" default:"3"`
	} `json:"inner"`
	When    time.Time `json:"when"`
	Ptr     *int      `json:"ptr"`
	private int
}

func TestApplyDefaults(t *testing.T) {
	b := newTestBinder()
	var got defaultsReq
	if err := b.Bind(nil, &got); err != nil {
		t.Fatal(err)
	}
	if got.Page != 1 || got.PageSize != 20 || got.Name != "匿名" || got.Ratio != 0.5 || !got.Enabled ||
		got.Limit != 8 || got.Wait != time.Minute || got.Inner.Level != 3 {
		t.Errorf("默认值没有填好：%+v", got)
	}
	if got.Ptr != nil || got.private != 0 || !got.When.IsZero() {
		t.Errorf("指针、未导出字段和时间不应被填写：%+v", got)
	}

	var override defaultsReq
	if err := b.Bind([]byte(`{"page":3,"name":"小明","on":false}`), &override); err != nil {
		t.Fatal(err)
	}
	if override.Page != 3 || override.Name != "小明" || override.Enabled || override.PageSize != 20 {
		t.Errorf("请求体里的值应当覆盖默认值：%+v", override)
	}
}

func TestDefaultsInvalidTag(t *testing.T) {
	type badInt struct {
		N int `default:"abc"`
	}
	type badKind struct {
		Ptr *int `default:"9"`
	}
	b := newTestBinder()
	if err := b.Bind(nil, &badInt{}); err == nil || !strings.Contains(err.Error(), "badInt.N") {
		t.Errorf("非法的默认值应当指出字段，得到 %v", err)
	}
	if err := b.Bind(nil, &badKind{}); err == nil || !strings.Contains(err.Error(), "不支持") {
		t.Errorf("不支持的类型应当报错，得到 %v", err)
	}
	for _, tt := range []struct {
		typ reflect.Type
		raw string
	}{
		{reflect.TypeFor[bool](), "x"},
		{reflect.TypeFor[uint](), "-1"},
		{reflect.TypeFor[float64](), "x"},
		{reflect.TypeFor[time.Duration](), "x"},
	} {
		if _, err := parseDefault(tt.typ, tt.raw); err == nil {
			t.Errorf("%s 的默认值 %q 应当报错", tt.typ, tt.raw)
		}
	}
}

func TestBindNonStruct(t *testing.T) {
	var m map[string]any
	if err := newTestBinder().Bind([]byte(`{"a":1}`), &m); err != nil || m["a"] != float64(1) {
		t.Errorf("非结构体也应能解析，得到 %v，%v", m, err)
	}
}

func TestBindBadJSON(t *testing.T) {
	var req pageReq
	err := newTestBinder().Bind([]byte(`{"page":`), &req)
	var perr *ParseError
	if !errors.As(err, &perr) || !strings.HasPrefix(perr.Error(), "请求体不是合法的 JSON") {
		t.Errorf("非法的 JSON 应当返回 ParseError，得到 %v", err)
	}
}

// trim 标签：先去掉首尾的空白再校验，所以全是空格的名称过不了 required
func TestBindTrim(t *testing.T) {
	type inner struct {
		Code string `json:"code" trim:"true"`
	}
	type nameReq struct {
		inner
		Name   string `json:"name"   label:"名称" trim:"true" validate:"required,max=3"`
		Remark string `json:"remark"` // 没有 trim 标签的字段原样保留
	}
	var req nameReq
	if err := newTestBinder().Bind([]byte(`{"name":"  总公司\t\n","code":" A1 ","remark":" 备注 "}`), &req); err != nil {
		t.Fatal(err)
	}
	if req.Name != "总公司" || req.Code != "A1" || req.Remark != " 备注 " {
		t.Errorf("带 trim 标签的字段应当去掉首尾空白，其余的不动：%+v", req)
	}

	err := newTestBinder().Bind([]byte(`{"name":"   "}`), &nameReq{})
	var verr *Error
	if !errors.As(err, &verr) || verr.Error() != "名称不能为空" {
		t.Errorf("全是空格的名称应当提示不能为空，得到 %v", err)
	}

	type badValue struct {
		Name string `trim:"yes"`
	}
	type badKind struct {
		N int `trim:"true"`
	}
	for _, out := range []any{&badValue{}, &badKind{}} {
		if err := newTestBinder().Bind(nil, out); err == nil || !strings.Contains(err.Error(), "trim 标签") {
			t.Errorf("写错的 trim 标签应当报错，得到 %v", err)
		}
	}

	// 标签写在切片、指针、map 里的结构体上不会生效：第一次用就报出来，不让它悄悄不起作用
	type item struct {
		Name string `json:"name" trim:"true"`
	}
	type plain struct {
		Name string `json:"name"`
	}
	type tree struct {
		Name     string `json:"name" default:"根"`
		Children []tree `json:"children"`
	}
	for name, out := range map[string]any{
		"切片":    &struct{ Items []item }{},
		"指针":    &struct{ Item *item }{},
		"map":   &struct{ Items map[string]item }{},
		"套了两层":  &struct{ Rows [][]*item }{},
		"自己套自己": &tree{},
	} {
		if err := newTestBinder().Bind(nil, out); err == nil || !strings.Contains(err.Error(), "不会生效") {
			t.Errorf("%s里的结构体带标签应当报错，得到 %v", name, err)
		}
	}
	// 切片里的结构体没有这两个标签，照常能用
	var ok struct {
		Items []plain     `json:"items"`
		Times []time.Time `json:"times"`
	}
	if err := newTestBinder().Bind([]byte(`{"items":[{"name":" a "}]}`), &ok); err != nil || ok.Items[0].Name != " a " {
		t.Errorf("切片里没有标签的结构体应当照常解析，得到 %+v，%v", ok, err)
	}
}

// 值的类型和字段对不上：按字段给提示，不再笼统地说"参数解析错误"
func TestBindTypeMismatch(t *testing.T) {
	type item struct {
		Count int `json:"count" label:"数量"`
	}
	type embedded struct {
		Level int8 `json:"level" label:"级别"`
	}
	type sortReq struct {
		embedded
		Sort   int     `json:"sort"   label:"排序"`
		Status int16   `json:"status" msg:"状态只能是 0 或 1"`
		Ids    []int64 `json:"ids"`
		Items  []item  `json:"items"`
		NoTag  int
	}
	tests := []struct {
		body    string
		field   string
		message string
	}{
		{`{"sort":1.5}`, "sort", "排序的格式不对"},
		{`{"sort":"1"}`, "sort", "排序的格式不对"},
		{`{"level":300}`, "level", "级别的格式不对"},                   // 嵌入的结构体里的字段；超出 int8 的范围
		{`{"status":"1"}`, "status", "状态只能是 0 或 1"},             // msg 标签优先
		{`{"ids":["1"]}`, "ids", "ids的格式不对"},                    // 没有 label 时用 JSON 字段名
		{`{"items":[{"count":"x"}]}`, "items.count", "数量的格式不对"}, // 切片里的结构体
		{`{"NoTag":"x"}`, "NoTag", "NoTag的格式不对"},
	}
	for _, tt := range tests {
		err := newTestBinder().Bind([]byte(tt.body), &sortReq{})
		var verr *Error
		if !errors.As(err, &verr) || len(verr.Fields) != 1 {
			t.Errorf("%s：应当返回一个字段的校验错误，得到 %v", tt.body, err)
			continue
		}
		if f := verr.Fields[0]; f.Field != tt.field || f.Rule != "type" || f.Message != tt.message {
			t.Errorf("%s：得到 %+v，期望字段 %q、提示 %q", tt.body, f, tt.field, tt.message)
		}
	}

	// 整个请求体的类型就不对（要对象却给了数组）：还是解析错误
	var perr *ParseError
	if err := newTestBinder().Bind([]byte(`[1,2]`), &sortReq{}); !errors.As(err, &perr) {
		t.Errorf("请求体不是对象应当返回 ParseError，得到 %v", err)
	}
}

// 字符串里的空字符：数据库存不了，在这里挡住
func TestBindNUL(t *testing.T) {
	type nameReq struct {
		Name string `json:"name"`
	}
	for _, body := range []string{`{"name":"a\u0000b"}`, `{"name":"\\\u0000"}`, `{"a\u0000":1}`} {
		err := newTestBinder().Bind([]byte(body), &nameReq{})
		var verr *Error
		if !errors.As(err, &verr) || verr.Fields[0].Rule != "nul" {
			t.Errorf("%s：有空字符应当返回校验错误，得到 %v", body, err)
		}
	}
	// 字面的反斜杠加 u0000 不是空字符，要放行
	for body, want := range map[string]string{
		`{"name":"a\\u0000b"}`: `a\u0000b`,
		`{"name":"\\\\u0000"}`: `\\u0000`,
		`{"name":"u0000 A"}`:   "u0000 A",
	} {
		var req nameReq
		if err := newTestBinder().Bind([]byte(body), &req); err != nil || req.Name != want {
			t.Errorf("%s：应当解析成 %q，得到 %q，%v", body, want, req.Name, err)
		}
	}
}

func TestBindValidation(t *testing.T) {
	type greetReq struct {
		Name  string `json:"name"  label:"名字" validate:"required,max=5"`
		Times int    `json:"times" default:"1"  validate:"gte=1,lte=3"`
	}
	err := newTestBinder().Bind([]byte(`{"name":"一个很长的名字","times":9}`), &greetReq{})
	var verr *Error
	if !errors.As(err, &verr) {
		t.Fatalf("应当返回 ValidationError，得到 %v", err)
	}
	want := []FieldError{
		{Field: "name", Rule: "max", Param: "5", Message: "名字最多5个字符"},
		{Field: "times", Rule: "lte", Param: "3", Message: "times不能大于3"},
	}
	if !reflect.DeepEqual(verr.Fields, want) {
		t.Errorf("Fields = %+v，期望 %+v", verr.Fields, want)
	}
}
