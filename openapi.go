package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

var DefaultImpl OpenAPI

func NewOpenAPI(token *Token) OpenAPI {
	DefaultImpl = &openAPI{}
	return DefaultImpl.Setup(token, false)
}

// MaxIdleConns 默认指定空闲连接池大小
const MaxIdleConns = 3000

type openAPI struct {
	token   *Token
	timeout time.Duration

	sandbox bool // 请求沙箱环境
	debug   bool // debug 模式，调试sdk时候使用
	// lastTraceID string // lastTraceID id

	restyClient *resty.Client // resty client 复用
}

// VersionMapping openapi 版本管理
var VersionMapping = map[uint32]OpenAPI{}

var (
	versionMapLock = sync.RWMutex{}
	once           sync.Once
)

// 这些状态码不会当做错误处理
// 未排除 201,202 : 用于提示创建异步任务成功，所以不屏蔽错误
var successStatusSet = map[int]bool{
	http.StatusOK:        true,
	http.StatusNoContent: true,
}

// func Register(version uint32, api OpenAPI) {
// 	versionMapLock.Lock()
// 	VersionMapping[version] = api
// 	setDefaultOnce(api)
// 	versionMapLock.Unlock()
// }

// IsSuccessStatus 是否是成功的状态码
func IsSuccessStatus(code int) bool {
	if _, ok := successStatusSet[code]; ok {
		return true
	}
	return false
}

// func setDefaultOnce(api OpenAPI) {
// 	once.Do(func() {
// 		if DefaultImpl == nil {
// 			DefaultImpl = api
// 		}
// 	})
// }

const (
	APIv1 uint32 = 1 + iota
	_
)

// // Setup 注册
// func Setup() {
// 	Register(APIv1, &openAPI{})
// }

// Version 创建当前版本
func (o *openAPI) Version() uint32 {
	return APIv1
}

// TraceID 获取 lastTraceID id
// func (o *openAPI) TraceID() string {
// 	return o.lastTraceID
// }

// Setup 生成一个实例
func (o *openAPI) Setup(token *Token, inSandbox bool) OpenAPI {
	api := &openAPI{
		token:   token,
		timeout: 3 * time.Second,
		sandbox: inSandbox,
	}
	api.setupClient() // 初始化可复用的 client
	return api
}

// WithTimeout 设置请求接口超时时间
func (o *openAPI) WithTimeout(duration time.Duration) OpenAPI {
	o.restyClient.SetTimeout(duration)
	return o
}

// Transport 透传请求
func (o *openAPI) Transport(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	resp, err := o.request(ctx).SetBody(body).Execute(method, url)
	return resp.Body(), err
}

func DoReqFilterChains(req *http.Request, resp *http.Response) error {
	for _, name := range reqFilterChains {
		if _, ok := reqFilterChainSet[name]; !ok {
			continue
		}
		if err := reqFilterChainSet[name](req, resp); err != nil {
			return err
		}
	}
	return nil
}

// const TraceIDKey = "X-Tps-trace-ID"

// 初始化 client
func (o *openAPI) setupClient() {
	o.restyClient = resty.New().
		SetTransport(createTransport(nil, MaxIdleConns)). // 自定义 transport
		// SetLogger(log.DefaultLogger).
		SetDebug(o.debug).
		SetTimeout(o.timeout).
		SetAuthToken(o.token.GetString()).
		SetAuthScheme(string(o.token.Type)).
		SetHeader("User-Agent", "v1").
		SetPreRequestHook(
			func(client *resty.Client, request *http.Request) error {
				// 执行请求前过滤器
				// 由于在 `OnBeforeRequest` 的时候，request 还没生成，所以 filter 不能使用，所以放到 `PreRequestHook`
				return DoReqFilterChains(request, nil)
			},
		).
		// 设置请求之后的钩子，打印日志，判断状态码
		OnAfterResponse(
			func(client *resty.Client, resp *resty.Response) error {
				// log.Infof("%v", respInfo(resp))
				// 执行请求后过滤器
				if err := DoRespFilterChains(resp.Request.RawRequest, resp.RawResponse); err != nil {
					return err
				}
				// traceID := resp.Header().Get(TraceIDKey)
				// o.lastTraceID = traceID
				// 非成功含义的状态码，需要返回 error 供调用方识别
				// if !IsSuccessStatus(resp.StatusCode()) {
				// 	return errs.New(resp.StatusCode(), string(resp.Body()), traceID)
				// }
				return nil
			},
		)
}

// request 每个请求，都需要创建一个 request
func (o *openAPI) request(ctx context.Context) *resty.Request {
	return o.restyClient.R().SetContext(ctx)
}

// respInfo 用于输出日志的时候格式化数据
func respInfo(resp *resty.Response) string {
	bodyJSON, _ := json.Marshal(resp.Request.Body)
	return fmt.Sprintf(
		"[OPENAPI]%v %v, status:%v, elapsed:%v req: %v, resp: %v",
		resp.Request.Method,
		resp.Request.URL,
		// resp.Header().Get(TraceIDKey),
		resp.Status(),
		resp.Time(),
		string(bodyJSON),
		string(resp.Body()),
	)
}

func createTransport(localAddr net.Addr, idleConns int) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 60 * time.Second,
	}
	if localAddr != nil {
		dialer.LocalAddr = localAddr
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          idleConns,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   idleConns,
		MaxConnsPerHost:       idleConns,
	}
}

type HTTPFilter func(req *http.Request, response *http.Response) error

var (
	filterLock         = sync.RWMutex{}
	reqFilterChainSet  = map[string]HTTPFilter{}
	reqFilterChains    []string
	respFilterChainSet = map[string]HTTPFilter{}
	respFilterChains   []string
)

// DoRespFilterChains 按照注册顺序执行返回过滤器
func DoRespFilterChains(req *http.Request, resp *http.Response) error {
	for _, name := range respFilterChains {
		if _, ok := respFilterChainSet[name]; !ok {
			continue
		}
		if err := respFilterChainSet[name](req, resp); err != nil {
			return err
		}
	}
	return nil
}
