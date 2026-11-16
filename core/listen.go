package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"net/netip"
	"sync"
	"time"

	"github.com/pires/go-proxyproto"

	log "github.com/pony5551/PnWind/core/pnlogrus"
)

// -------------------------------------------------------------------------
// 监听（Boot 第 7 步）
// -------------------------------------------------------------------------

// listen 监听 addr。useProxy 为 true 时解析 PROXY 协议头：服务前面有负载均衡时，用它取得真实的客户端地址；
// 但只采用可信代理发来的，trusted 里每一项是一个 IP 或一个网段。
//
// 对任何来源的 PROXY 头都照单全收的话，任何人都能伪造来源地址；
// 所以默认不解析，开启时必须写明可信代理。清单写错、端口被占用都返回错误，由 Boot 决定不启动。
func listen(addr string, useProxy bool, trusted []string) (*readyListener, error) {
	var policy proxyproto.ConnPolicyFunc
	if useProxy {
		var err error
		if policy, err = trustedPolicy(trusted); err != nil {
			return nil, err
		}
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("监听 %s 失败: %w", addr, err)
	}
	if useProxy {
		ln = &proxyproto.Listener{Listener: ln, ConnPolicy: policy}
		log.Printf("PROXY 协议已开启，可信代理: %v", trusted)
	}
	return &readyListener{Listener: ln, ready: make(chan struct{})}, nil
}

// trustedPolicy 按可信代理的清单生成 PROXY 协议的连接策略：
// 来自清单里的地址的连接，采用 PROXY 头里的客户端地址（没带头的照常按普通连接处理）；
// 其他连接忽略 PROXY 头，按连接本身的地址处理。
// 清单是空的、或者有一项既不是 IP 也不是网段，返回错误。
func trustedPolicy(trusted []string) (proxyproto.ConnPolicyFunc, error) {
	if len(trusted) == 0 {
		return nil, errors.New("开启了 ProxyProtocol，但没有配置 ProxyTrusted（可信代理的 IP 或网段）")
	}
	prefixes := make([]netip.Prefix, len(trusted))
	for i, s := range trusted {
		p, err := netip.ParsePrefix(s)
		if err != nil { // 不是网段，按单个 IP 算：只含它自己的网段
			ip, ipErr := netip.ParseAddr(s)
			if ipErr != nil {
				return nil, fmt.Errorf("ProxyTrusted 里的 %q 不是合法的 IP 或网段，应当形如 10.0.0.8 或 10.0.0.0/8", s)
			}
			ip = ip.Unmap()
			p = netip.PrefixFrom(ip, ip.BitLen())
		}
		prefixes[i] = p
	}
	return func(opts proxyproto.ConnPolicyOptions) (proxyproto.Policy, error) {
		if tcp, isOk := opts.Upstream.(*net.TCPAddr); isOk {
			if ip, isOk := netip.AddrFromSlice(tcp.IP); isOk {
				ip = ip.Unmap() // 连接的地址可能是 ::ffff:10.0.0.8 这种写法
				for _, p := range prefixes {
					if p.Contains(ip) {
						return proxyproto.USE, nil
					}
				}
			}
		}
		return proxyproto.IGNORE, nil
	}, nil
}

// readyListener 在第一次 Accept 时关闭 Ready 返回的通道。
//
// fasthttp 先登记监听、再开始 Accept；在登记之前调用 Shutdown，监听不会被关闭，服务会一直挂着。
// 所以 Boot 要等到 Ready 之后才开始等退出信号。
type readyListener struct {
	net.Listener
	once  sync.Once
	ready chan struct{}
}

// Ready 返回一个通道，服务开始接收连接时关闭。
func (l *readyListener) Ready() <-chan struct{} {
	return l.ready
}

// Accept 实现 net.Listener。
func (l *readyListener) Accept() (net.Conn, error) {
	l.once.Do(func() { close(l.ready) })
	return l.Listener.Accept()
}

// -------------------------------------------------------------------------
// 性能剖析（pprof）
//
// pprof 挂在业务端口上、又排在鉴权之前的话，谁都能访问。
// 所以默认不开；开启时单独一个端口，只能监听本机地址。
// -------------------------------------------------------------------------

// pprofServer 性能剖析服务。没有开启时是 nil，stop 对 nil 什么也不做。
type pprofServer struct {
	srv *http.Server
}

// startPprof 在 addr 上启动性能剖析服务。addr 为空表示不开启，返回 nil。
func startPprof(addr string) (*pprofServer, error) {
	if addr == "" {
		return nil, nil
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("PprofAddr %q 不合法，应当形如 127.0.0.1:6060: %w", addr, err)
	}
	if ip, ipErr := netip.ParseAddr(host); host != "localhost" && (ipErr != nil || !ip.IsLoopback()) {
		return nil, fmt.Errorf("PprofAddr %q 不是本机地址：性能剖析只能监听 127.0.0.1 这样的本机地址", addr)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("pprof 监听 %s 失败: %w", addr, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	p := &pprofServer{srv: &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}}
	go func() {
		if err := p.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorln("pprof 服务出错:", err)
		}
	}()
	log.Printf("性能剖析已开启: http://%s/debug/pprof/", ln.Addr())
	return p, nil
}

// stop 关闭性能剖析服务，等正在进行的剖析结束。
func (p *pprofServer) stop() error {
	if p == nil {
		return nil
	}
	return p.srv.Shutdown(context.Background())
}
