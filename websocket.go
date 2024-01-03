package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	wss "github.com/gorilla/websocket" // 是一个流行的 websocket 客户端，服务端实现
	"github.com/tencent-connect/botgo/log"
	"github.com/tidwall/gjson"
)

// WebsocketAP wss 接入点信息
type WebsocketAP struct {
	URL               string            `json:"url"`
	Shards            uint32            `json:"shards"`
	SessionStartLimit SessionStartLimit `json:"session_start_limit"`
}

// SessionStartLimit 链接频控信息
type SessionStartLimit struct {
	Total          uint32 `json:"total"`
	Remaining      uint32 `json:"remaining"`
	ResetAfter     uint32 `json:"reset_after"`
	MaxConcurrency uint32 `json:"max_concurrency"`
}

// ShardConfig 连接的 shard 配置，ShardID 从 0 开始，ShardCount 最小为 1
type ShardConfig struct {
	ShardID    uint32
	ShardCount uint32
}

// WSResumeData 重连数据
type WSResumeData struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Seq       uint32 `json:"seq"`
}

// Session 连接的 session 结构，包括链接的所有必要字段
type Session struct {
	ID      string
	URL     string
	Token   Token
	Intent  Intent
	LastSeq uint32
	Shards  ShardConfig
}

// String 输出session字符串
func (s *Session) String() string {
	return fmt.Sprintf("[ws][ID:%s][Shard:(%d/%d)][Intent:%d]",
		s.ID, s.Shards.ShardID, s.Shards.ShardCount, s.Intent)
}

// WSUser 当前连接的用户信息
type WSUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Bot      bool   `json:"bot"`
}

const (
	gatewayURI    string = "/gateway" // nolint
	gatewayBotURI string = "/gateway/bot"
)

func (o *openAPI) WS(ctx context.Context, _ map[string]string, _ string) (*WebsocketAP, error) {
	resp, err := o.request(ctx).
		SetResult(WebsocketAP{}).
		Get(o.getURL(gatewayBotURI))
	if err != nil {
		return nil, err
	}

	return resp.Result().(*WebsocketAP), nil
}

func RegisterHandlers(handlers ...interface{}) Intent {
	return RRegisterHandlers(handlers...)
}

type WebSocket interface {
	// New 创建一个新的ws实例，需要传递 session 对象
	Newss(session Session) WebSocket
	// Connect 连接到 wss 地址
	Connect() error
	// Identify 鉴权连接
	Identify() error
	// Session 拉取 session 信息，包括 token，shard，seq 等
	Session() *Session
	// Resume 重连
	Resume() error
	// Listening 监听websocket事件
	Listening() error
	// Write 发送数据
	Write(message *WSPayload) error
	// Close 关闭连接
	Close()
}

// WSHelloData hello 返回
type WSHelloData struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

type WSReadyData struct {
	SessionID string `json:"session_id"`
	User      struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Bot      bool   `json:"bot"`
	} `json:"user"`
	Shard []uint32 `json:"shard"`
}

// WSIdentityData 鉴权数据
type WSIdentityData struct {
	Token   string   `json:"token"`
	Intents Intent   `json:"intents"`
	Shard   []uint32 `json:"shard"` // array of two integers (shard_id, num_shards)
}

type messageChan chan *WSPayload
type closeErrorChan chan error

// Client websocket 连接客户端
type Client struct {
	conn            *wss.Conn
	messageQueue    messageChan
	session         *Session
	user            *WSUser
	closeChan       closeErrorChan
	heartBeatTicker *time.Ticker // 用于维持定时心跳
}

// Setup 依赖注册
func Setup() {
	Register(&Client{})
}

// DefaultQueueSize 监听队列的缓冲长度
const DefaultQueueSize = 10000

func (c *Client) Newss(session Session) WebSocket {
	return &Client{
		messageQueue:    make(messageChan, DefaultQueueSize),
		session:         &session,
		closeChan:       make(closeErrorChan, 10),
		heartBeatTicker: time.NewTicker(60 * time.Second), // 先给一个默认 ticker，在收到 hello 包之后，会 resets
	}
}

// WS OPCode
const (
	WSDispatchEvent int = iota
	WSHeartbeat
	WSIdentity
	_ // Presence Update
	_ // Voice State Update
	_
	WSResume
	WSReconnect
	_ // Request Guild Members
	WSInvalidSession
	WSHello
	WSHeartbeatAck
	HTTPCallbackAck
)

// Connect 连接到 websocket
func (c *Client) Connect() error {
	if c.session.URL == "" {
		return errors.New("url invaild")
	}

	var err error
	c.conn, _, err = wss.DefaultDialer.Dial(c.session.URL, nil)
	if err != nil {
		return err
	}

	return nil
}

// Listening 开始监听，会阻塞进程，内部会从事件队列不断的读取事件，解析后投递到注册的 event handler，如果读取消息过程中发生错误，会循环
// 定时心跳也在这里维护
func (c *Client) Listening() error {
	defer c.Close()
	// reading message
	go c.readMessageToQueue()
	// read message from queue and handle,in goroutine to avoid business logic block closeChan and heartBeatTicker
	go c.listenMessageAndHandle()

	// 接收 resume signal
	resumeSignal := make(chan os.Signal, 1)
	if ResumeSignal >= syscall.SIGHUP {
		signal.Notify(resumeSignal, ResumeSignal)
	}

	// handler message
	for {
		select {
		case <-resumeSignal: // 使用信号量控制连接立即重连
			return errors.New("received resumeSignal signal")
		case <-c.heartBeatTicker.C:
			heartBeatEvent := &WSPayload{
				WSPayloadBase: WSPayloadBase{
					OPCode: WSHeartbeat,
				},
				Data: c.session.LastSeq,
			}
			// 不处理错误，Write 内部会处理，如果发生发包异常，会通知主协程退出
			_ = c.Write(heartBeatEvent)
		}
	}
}

// Write 往 ws 写入数据
func (c *Client) Write(message *WSPayload) error {
	m, _ := json.Marshal(message)

	if err := c.conn.WriteMessage(wss.TextMessage, m); err != nil {
		c.closeChan <- err
		return err
	}
	return nil
}

// Identify 对一个连接进行鉴权，并声明监听的 shard 信息
func (c *Client) Identify() error {
	// 避免传错 intent
	if c.session.Intent == 0 {
		c.session.Intent = IntentGuilds
	}
	payload := &WSPayload{
		Data: &WSIdentityData{
			Token:   c.session.Token.GetString(),
			Intents: c.session.Intent,
			Shard: []uint32{
				c.session.Shards.ShardID,
				c.session.Shards.ShardCount,
			},
		},
	}
	payload.OPCode = WSIdentity
	return c.Write(payload)
}

// Close 关闭连接
func (c *Client) Close() {
	if err := c.conn.Close(); err != nil {
	}
	c.heartBeatTicker.Stop()
}

func (c *Client) readMessageToQueue() {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			close(c.messageQueue)
			c.closeChan <- err
			return
		}
		payload := &WSPayload{}
		if err := json.Unmarshal(message, payload); err != nil {
			continue
		}
		payload.RawMessage = message
		c.messageQueue <- payload
	}
}

func (c *Client) listenMessageAndHandle() {
	defer func() {
		// 打印日志后，关闭这个连接，进入重连流程
		if err := recover(); err != nil {
			PanicHandler(err, c.session)
			c.closeChan <- fmt.Errorf("panic: %v", err)
		}
	}()
	for payload := range c.messageQueue {
		c.saveSeq(payload.Seq)
		// ready 事件需要特殊处理
		if payload.Type == "READY" {
			c.readyHandler(payload)
			continue
		}
		// 解析具体事件，并投递给业务注册的 handler
		if err := ParseAndHandle(payload); err != nil {
			log.Errorf("%s parseAndHandle failed, %v", c.session, err)
		}
	}
	log.Infof("%s message queue is closed", c.session)
}

func (c *Client) saveSeq(seq uint32) {
	if seq > 0 {
		c.session.LastSeq = seq
	}
}

// startHeartBeatTicker 启动定时心跳
func (c *Client) startHeartBeatTicker(message []byte) {
	helloData := &WSHelloData{}
	if err := ParseData(message, helloData); err != nil {
	}
	// 根据 hello 的回包，重新设置心跳的定时器时间
	c.heartBeatTicker.Reset(time.Duration(helloData.HeartbeatInterval) * time.Millisecond)
}

// ParseData 解析数据
func ParseData(message []byte, target interface{}) error {
	data := gjson.Get(string(message), "d")
	return json.Unmarshal([]byte(data.String()), target)
}

// readyHandler 针对ready返回的处理，需要记录 sessionID 等相关信息
func (c *Client) readyHandler(payload *WSPayload) {
	readyData := &WSReadyData{}
	if err := ParseData(payload.RawMessage, readyData); err != nil {
		log.Errorf("%s parseReadyData failed, %v, message %v", c.session, err, payload.RawMessage)
	}
	// 基于 ready 事件，更新 session 信息
	c.session.ID = readyData.SessionID
	c.session.Shards.ShardID = readyData.Shard[0]
	c.session.Shards.ShardCount = readyData.Shard[1]
	c.user = &WSUser{
		ID:       readyData.User.ID,
		Username: readyData.User.Username,
		Bot:      readyData.User.Bot,
	}
	// 调用自定义的 ready 回调
	if DefaultHandlers.Ready != nil {
		DefaultHandlers.Ready(payload, readyData)
	}
}

// ParseAndHandle 处理回调事件
func ParseAndHandle(payload *WSPayload) error {
	// 指定类型的 handler
	if h, ok := eventParseFuncMap[payload.OPCode][payload.Type]; ok {
		return h(payload, payload.RawMessage)
	}
	return nil
}

// Resume 重连
func (c *Client) Resume() error {
	payload := &WSPayload{
		Data: &WSResumeData{
			Token:     c.session.Token.GetString(),
			SessionID: c.session.ID,
			Seq:       c.session.LastSeq,
		},
	}
	payload.OPCode = WSResume // 内嵌结构体字段，单独赋值
	return c.Write(payload)
}

// Session 获取client的session信息
func (c *Client) Session() *Session {
	return c.session
}
