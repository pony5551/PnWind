package util

import (
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
)

var httpclientobject *fasthttp.Client

func init() {
	/*
		proxy:=httpproxySetting{} // 判断当前环境的机器是否需要支持代理
		proxy.Load()
		if proxy.Enabled==false {
			httpclientobject=&fasthttp.Client{}
		}else{
			if proxy.IsLocal==false {
				url:=proxy.UserName+":"+proxy.Password+"@"+proxy.Server+":"+proxy.Port
				httpclientobject=&fasthttp.Client{
					Dial:fasthttpproxy.FasthttpHTTPDialerTimeout(url,time.Second*30),
				}
			}else{
				httpclientobject=&fasthttp.Client{}
			}
		}
	*/
	defaultDialer := &fasthttp.TCPDialer{Concurrency: 0}
	httpclientobject = &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			idx := 1 // 重试三次
			for {
				idx--
				conn, err := defaultDialer.DialTimeout(addr, 180*time.Second) // tcp连接超时时间30s
				if err != fasthttp.ErrDialTimeout || idx == 0 {
					return conn, err
				}
			}
		},
	}
	httpclientobject.ReadTimeout = time.Second * 30
	httpclientobject.MaxConnsPerHost = 10000
}

type PnHttpRespData struct {
	StatusCode int
	RespHead   []byte
	RespBody   []byte
}

// func PnHttpPost(Url string, ContentType string, ReqBody []byte, respData *PnHttpRespData) (int, []byte, []byte, error) {
func PnHttpPost(Url string, ContentType string, ReqHeaders map[string]string, ReqBody []byte, respData *PnHttpRespData) error {
	//var StatusCode = -1
	//var RespHead []byte
	//var RespBody []byte
	var err error
	if ContentType == "" {
		ContentType = "application/json"
	}
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	// 默认是application/x-www-form-urlencoded
	req.Header.SetContentType(ContentType)
	req.Header.SetMethod("POST")
	req.SetRequestURI(Url)
	req.SetBody(ReqBody)
	for k, v := range ReqHeaders {
		req.Header.Add(k, v)
	}

	for k, v := range ReqHeaders {
		req.Header.Set(k, v)
	}

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	//myUrl, _ := url.Parse(Url)
	//client := fasthttp.HostClient{
	//	Addr: myUrl.Host,
	//}
	//client.MaxConns = 10000
	if err = httpclientobject.DoTimeout(req, resp, time.Second*30); err != nil {
		//fmt.Println("请求失败:", err.Error())
		return err
	}
	httpclientobject.CloseIdleConnections()

	respData.StatusCode = resp.StatusCode()
	respData.RespHead = resp.Header.Header()
	ContentEncoding := string(resp.Header.ContentEncoding())
	switch ContentEncoding {
	case "gzip":
		unGZData := UnGZipBytes(resp.Body())
		respData.RespBody = make([]byte, len(unGZData))
		copy(respData.RespBody, unGZData)
	default:
		respData.RespBody = make([]byte, len(resp.Body()))
		copy(respData.RespBody, resp.Body())
	}
	return nil
}

// func PnHttpGet(Url string, respData *PnHttpRespData) (int, []byte, []byte, error) {
func PnHttpGet(Url string, ReqHeaders map[string]string, respData *PnHttpRespData) error {
	var err error
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	// 默认是application/x-www-form-urlencoded
	//req.Header.SetContentType(ContentType)
	req.Header.SetMethod("GET")
	req.SetRequestURI(Url)

	for k, v := range ReqHeaders {
		req.Header.Set(k, v)
	}

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	//myUrl, _ := url.Parse(Url)
	//client := fasthttp.HostClient{
	//	Addr: myUrl.Host,
	//}
	//client.MaxConns = 10000
	if err = httpclientobject.DoTimeout(req, resp, time.Second*30); err != nil {
		//fmt.Println("请求失败:", err.Error())
		return err
	}
	httpclientobject.CloseIdleConnections()

	respData.StatusCode = resp.StatusCode()
	respData.RespHead = resp.Header.Header()
	ContentEncoding := string(resp.Header.ContentEncoding())
	switch ContentEncoding {
	case "gzip":
		unGZData := UnGZipBytes(resp.Body())
		respData.RespBody = make([]byte, len(unGZData))
		copy(respData.RespBody, unGZData)
	default:
		respData.RespBody = make([]byte, len(resp.Body()))
		copy(respData.RespBody, resp.Body())
	}
	return nil
}

func HttpPost(url string, requestBody []byte) ([]byte, error) {
	ReqHeads := make(map[string]string)
	respData := PnHttpRespData{}

	err := PnHttpPost(url, "application/json", ReqHeads, requestBody, &respData)
	if err != nil {
		log.Errorln("HttpPost Error1:", url, err)
		return nil, err
	}

	if respData.StatusCode != 200 {
		err = errors.Errorf("响应失败,%d", respData.StatusCode)
		log.Errorln("HttpPost Error2:", url, err)
		return nil, err
	}

	return respData.RespBody, nil
}

func HttpPostEx(url string, ContentType string, ReqHeads map[string]string, requestBody []byte) ([]byte, error) {
	respData := PnHttpRespData{}

	err := PnHttpPost(url, ContentType, ReqHeads, requestBody, &respData)
	if err != nil {
		log.Errorln("HttpPost Error1:", url, err)
		return nil, err
	}

	if respData.StatusCode != 200 {
		err = errors.Errorf("响应失败,%d", respData.StatusCode)
		log.Errorln("HttpPost Error2:", url, err)
		return nil, err
	}

	return respData.RespBody, nil
}
