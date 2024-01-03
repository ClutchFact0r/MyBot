package main

import (
	"math"
	"runtime"
	"syscall"
	"time"
)

type SessionManager interface {
	// Start 启动连接，默认使用 apInfo 中的 shards 作为 shard 数量
	Start(apInfo *WebsocketAP, token *Token, intents *Intent) error
}

// New 创建本地session管理器
func NewSession() *ChanManager {
	return &ChanManager{}
}

type ChanManager struct {
	sessionChan chan Session
}

var defaultSessionManager SessionManager = NewSession()

func NewSessionManager() SessionManager {
	return defaultSessionManager
}

const concurrencyTimeWindowSec = 2

func CalcInterval(maxConcurrency uint32) time.Duration {
	if maxConcurrency == 0 {
		maxConcurrency = 1
	}
	f := math.Round(concurrencyTimeWindowSec / float64(maxConcurrency))
	if f == 0 {
		f = 1
	}
	return time.Duration(f) * time.Second
}

// Start 启动本地 session manager
func (l *ChanManager) Start(apInfo *WebsocketAP, token *Token, intents *Intent) error {
	startInterval := CalcInterval(apInfo.SessionStartLimit.MaxConcurrency)
	// 按照shards数量初始化，用于启动连接的管理
	l.sessionChan = make(chan Session, apInfo.Shards)
	for i := uint32(0); i < apInfo.Shards; i++ {
		session := Session{
			URL:     apInfo.URL,
			Token:   *token,
			Intent:  *intents,
			LastSeq: 0,
			Shards: ShardConfig{
				ShardID:    i,
				ShardCount: apInfo.Shards,
			},
		}
		l.sessionChan <- session
	}

	for session := range l.sessionChan {
		// MaxConcurrency 代表的是每 5s 可以连多少个请求
		time.Sleep(startInterval)
		go l.newConnect(session)
	}
	return nil
}

var (
	// ClientImpl websocket 实现
	ClientImpl WebSocket
	// ResumeSignal 用于强制 resume 连接的信号量
	ResumeSignal syscall.Signal
)

// Register 注册 websocket 实现
func Register(ws WebSocket) {
	ClientImpl = ws
}

// SetWebsocketClient 替换 websocket 实现
func SetWebsocketClient(c WebSocket) {
	Register(c)
}

// newConnect 启动一个新的连接，如果连接在监听过程中报错了，或者被远端关闭了链接，需要识别关闭的原因，能否继续 resume
func (l *ChanManager) newConnect(session Session) {
	defer func() {
		// panic 留下日志，放回 session
		if err := recover(); err != nil {
			PanicHandler(err, &session)
			l.sessionChan <- session
		}
	}()
	Setup()
	wsClient := ClientImpl.Newss(session)
	if err := wsClient.Connect(); err != nil {
		l.sessionChan <- session // 连接失败，丢回去队列排队重连
		return
	}
	var err error
	// 如果 session id 不为空，则执行的是 resume 操作，如果为空，则执行的是 identify 操作
	if session.ID != "" {
		err = wsClient.Resume()
	} else {
		// 初次鉴权
		err = wsClient.Identify()
	}
	if err != nil {
		return
	}
	if err := wsClient.Listening(); err != nil {
		currentSession := wsClient.Session()
		// 将 session 放到 session chan 中，用于启动新的连接，当前连接退出
		l.sessionChan <- *currentSession
		return
	}
}

var PanicBufLen = 1024

// PanicHandler 处理websocket场景的 panic ，打印堆栈
func PanicHandler(e interface{}, session *Session) {
	buf := make([]byte, PanicBufLen)
	buf = buf[:runtime.Stack(buf, false)]
}

const (
	CodeNeedReConnect = 9000 + iota
	// CodeInvalidSession 无效的的 session id 请重新连接
	CodeInvalidSession
	CodeURLInvalid
	CodeNotFoundOpenAPI
	CodeSessionLimit
	// CodeConnCloseCantResume 关闭连接错误码，收拢 websocket close error，不允许 resume
	CodeConnCloseCantResume
	// CodeConnCloseCantIdentify 不允许连接的关闭连接错误，比如机器人被封禁
	CodeConnCloseCantIdentify
	// CodePagerIsNil 分页器为空
	CodePagerIsNil
)

var CanNotResumeErrSet = map[int]bool{
	CodeConnCloseCantResume: true,
}
