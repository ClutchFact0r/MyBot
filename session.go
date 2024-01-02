package main

import (
	"math"
	"runtime"
	"syscall"
	"time"
)

type SessionManager interface {
	// Start 启动连接，默认使用 apInfo 中的 shards 作为 shard 数量，如果有需要自己指定 shard 数，请修 apInfo 中的信息
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
	// defer log.Sync()
	startInterval := CalcInterval(apInfo.SessionStartLimit.MaxConcurrency)
	// log.Infof("[ws/session/local] will start %d sessions and per session start interval is %s",
	// 	apInfo.Shards, startInterval)

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
// 如果能够 resume，则往 sessionChan 中放入带有 sessionID 的 session
// 如果不能，则清理掉 sessionID，将 session 放入 sessionChan 中
// session 的启动，交给 start 中的 for 循环执行，session 不自己递归进行重连，避免递归深度过深
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
		// log.Error(err)
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
		// log.Errorf("[ws/session] Identify/Resume err %+v", err)
		return
	}
	if err := wsClient.Listening(); err != nil {
		// log.Errorf("[ws/session] Listening err %+v", err)
		currentSession := wsClient.Session()
		// 对于不能够进行重连的session，需要清空 session id 与 seq
		// if CanNotResume(err) {
		// 	currentSession.ID = ""
		// 	currentSession.LastSeq = 0
		// }
		// // 一些错误不能够鉴权，比如机器人被封禁，这里就直接退出了
		// if CanNotIdentify(err) {
		// 	msg := fmt.Sprintf("can not identify because server return %+v, so process exit", err)
		// 	// log.Errorf(msg)
		// 	panic(msg) // 当机器人被下架，或者封禁，将不能再连接，所以 panic
		// }
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
	// log.Errorf("[PANIC]%s\n%v\n%s\n", session, e, buf)
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
