package routers

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/pony5551/PnWind/core/pncontext"
	"github.com/pony5551/PnWind/syscore/utils"
)

// ActFunc 一个操作的处理函数：控制器上的 Page_<act> 方法和 PageMain 方法都是这个写法。
type ActFunc = func(pnctx *pncontext.PnContext) error

// Page 一个页面，对应一个控制器。请求交给控制器有两种方式：
//
//	自动分发：控制器上每个 Page_<act> 方法是一个操作，按地址里的 act 调用。管理页面都这样写。
//	显式调用：控制器自己写了入口 PageMain，这个页面的请求全部交给它。不按 act 分的接口这样写。
//
// 控制器只在注册时扫描一次；请求到来时查表、直接调用方法，不经过反射。
type Page struct {
	Public bool               // 不用登录就能访问，例如登录页（见 RegisterPublicController）
	main   ActFunc            // 控制器的 PageMain，没写就是 nil
	acts   map[string]ActFunc // act → Page_<act> 方法
}

// NewPage 扫描控制器 ctrl 上的方法：PageMain 是入口；Page_list 这样的是 act "list"（去掉 Page_ 再转成小写）。
func NewPage[T any](ctrl *T) (*Page, error) {
	page := &Page{acts: map[string]ActFunc{}}
	t := reflect.TypeOf(ctrl)
	for i := range t.NumMethod() {
		m := t.Method(i)
		if m.Name != "PageMain" && !strings.HasPrefix(m.Name, "Page_") {
			continue
		}
		// m.Func 就是这个方法本身（接收者是第一个参数），取出来之后调用它是普通的函数调用
		fn, isOk := m.Func.Interface().(func(*T, *pncontext.PnContext) error)
		if !isOk {
			return nil, fmt.Errorf("%s.%s 的写法不对，要写成 func (c *XxxCont) %s(pnctx *pncontext.PnContext) error", t, m.Name, m.Name)
		}
		act := func(pnctx *pncontext.PnContext) error { return fn(ctrl, pnctx) }
		if m.Name == "PageMain" {
			page.main = act
			continue
		}
		name := strings.ToLower(strings.TrimPrefix(m.Name, "Page_"))
		if _, exists := page.acts[name]; exists {
			return nil, fmt.Errorf("%s 上有两个方法都对应 act %q，方法名转成小写后不能相同", t, name)
		}
		page.acts[name] = act
	}
	if page.main == nil && len(page.acts) == 0 {
		return nil, fmt.Errorf("%s 上没有 Page_ 开头的方法，也没有 PageMain", t)
	}
	return page, nil
}

// Dispatch 把请求交给控制器：有 PageMain 就交给它，没有就按 pnctx.Act 调用对应的 Page_ 方法。
func (p *Page) Dispatch(pnctx *pncontext.PnContext) error {
	if p.main != nil {
		return p.main(pnctx)
	}
	if fn, isOk := p.acts[pnctx.Act]; isOk {
		return fn(pnctx)
	}
	return pnctx.RespErr(utils.Resp_code_OptFail99999, "unknown act: "+pnctx.Act)
}

// String 这个页面的入口，启动时打印用：显式调用是 PageMain，自动分发是全部 act（按字母顺序）；不用登录的页面带上标记。
func (p *Page) String() string {
	s := "PageMain"
	if p.main == nil {
		s = strings.Join(slices.Sorted(maps.Keys(p.acts)), " ")
	}
	if p.Public {
		s += "（不用登录）"
	}
	return s
}
