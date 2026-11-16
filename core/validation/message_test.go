package validation

import (
	"errors"
	"reflect"
	"testing"
)

func TestFieldMessages(t *testing.T) {
	type item struct {
		Score int `json:"score" label:"分数" validate:"gte=0"`
	}
	type req struct {
		pageReq
		Email string   `json:"email" validate:"omitempty,email"`
		Code  string   `json:"code"  msg:"请填写六位验证码" validate:"len=6"`
		IDs   []int64  `json:"ids"   label:"编号" validate:"min=1,dive,gt=0"`
		Items []item   `json:"items" validate:"dive"`
		Kind  string   `json:"kind"  label:"类型" validate:"oneof=a b c"`
		Tags  []string `json:"tags"  label:"标签" validate:"max=2"`
	}
	in := req{
		pageReq: pageReq{PageSize: 500},
		Email:   "不是邮箱",
		Code:    "12",
		IDs:     []int64{0},
		Items:   []item{{Score: -1}},
		Kind:    "x",
		Tags:    []string{"1", "2", "3"},
	}
	err := newTestBinder().check(&in)
	var verr *Error
	if !errors.As(err, &verr) {
		t.Fatalf("应当返回 ValidationError，得到 %v", err)
	}
	got := map[string]string{}
	for _, f := range verr.Fields {
		got[f.Field] = f.Message
	}
	want := map[string]string{
		"pageSize": "每页条数不能大于100",
		"email":    "email不是有效的邮箱地址",
		"code":     "请填写六位验证码",
		"ids[0]":   "编号必须大于0",
		"score":    "分数不能小于0",
		"kind":     "类型必须是以下值之一：a、b、c",
		"tags":     "标签最多2项",
	}
	for field, msg := range want {
		if got[field] != msg {
			t.Errorf("字段 %s 的提示 = %q，期望 %q", field, got[field], msg)
		}
	}
}

func TestRuleMessage(t *testing.T) {
	tests := []struct {
		rule, param string
		kind        reflect.Kind
		want        string
	}{
		{"required", "", reflect.String, "名称不能为空"},
		{"required_if", "Kind a", reflect.Int, "名称不能为空"},
		{"min", "2", reflect.String, "名称至少2个字符"},
		{"min", "1", reflect.Slice, "名称至少1项"},
		{"min", "0", reflect.Int, "名称不能小于0"},
		{"max", "9", reflect.Int, "名称不能大于9"},
		{"len", "6", reflect.String, "名称必须是6个字符"},
		{"len", "3", reflect.Int, "名称必须等于3"},
		{"eq", "1", reflect.Int, "名称必须等于1"},
		{"ne", "1", reflect.Int, "名称不能等于1"},
		{"gt", "0", reflect.Int, "名称必须大于0"},
		{"lt", "9", reflect.Int, "名称必须小于9"},
		{"lte", "9", reflect.Int, "名称不能大于9"},
		{"url", "", reflect.String, "名称不是有效的网址"},
		{"numeric", "", reflect.String, "名称必须是数字"},
		{"alphanum", "", reflect.String, "名称只能包含字母和数字"},
		{"uuid", "", reflect.String, "名称不符合规则 uuid"},
		{"startswith", "a", reflect.String, "名称不符合规则 startswith=a"},
	}
	for _, tt := range tests {
		if got := ruleMessage("名称", tt.rule, tt.param, tt.kind); got != tt.want {
			t.Errorf("ruleMessage(%s=%s, %s) = %q，期望 %q", tt.rule, tt.param, tt.kind, got, tt.want)
		}
	}
}

func TestStructFieldNotFound(t *testing.T) {
	root := reflect.TypeFor[pageReq]()
	for _, ns := range []string{"pageReq", "pageReq.Missing", "pageReq.Page.X"} {
		if _, ok := structField(root, ns); ok {
			t.Errorf("命名空间 %q 应当找不到字段", ns)
		}
	}
}

func TestJSONName(t *testing.T) {
	type s struct {
		A int `json:"a,omitempty"`
		B int `json:"-"`
		C int
	}
	typ := reflect.TypeFor[s]()
	for i, want := range []string{"a", "", ""} {
		if got := jsonName(typ.Field(i)); got != want {
			t.Errorf("字段 %d 的 JSON 名 = %q，期望 %q", i, got, want)
		}
	}
}

// 处理器里临时定义的匿名结构体：label 和 msg 标签同样要生效。
func TestAnonymousStructLabels(t *testing.T) {
	var form struct {
		Name string `json:"name" label:"名称" validate:"required"`
		Code string `json:"code" msg:"请填写六位验证码" validate:"len=6"`
		Page struct {
			Size int `json:"size" label:"每页条数" validate:"lte=100"`
		} `json:"page"`
	}
	form.Code = "12"
	form.Page.Size = 500
	err := newTestBinder().check(&form)
	var verr *Error
	if !errors.As(err, &verr) {
		t.Fatalf("应当返回 *Error，得到 %v", err)
	}
	got := map[string]string{}
	for _, f := range verr.Fields {
		got[f.Field] = f.Message
	}
	want := map[string]string{"name": "名称不能为空", "code": "请填写六位验证码", "size": "每页条数不能大于100"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("提示 = %v，期望 %v", got, want)
	}
}
