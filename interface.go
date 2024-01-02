package main

import (
	"context"
	"time"
)

// RetractMessageOption 撤回消息可选参数
type RetractMessageOption int

const (
	// RetractMessageOptionHidetip 撤回消息隐藏小灰条可选参数
	RetractMessageOptionHidetip RetractMessageOption = 1
)

// OpenAPI openapi 完整实现
type OpenAPI interface {
	Base
	WebsocketAPI
	MessageAPI
}

// Base 基础能力接口
type Base interface {
	Setup(token *Token, inSandbox bool) OpenAPI
	// WithTimeout 设置请求接口超时时间
	WithTimeout(duration time.Duration) OpenAPI
	// Transport 透传请求，如果 sdk 没有及时跟进新的接口的变更，可以使用该方法进行透传，openapi 实现时可以按需选择是否实现该接口
	Transport(ctx context.Context, method, url string, body interface{}) ([]byte, error)
	// TraceID 返回上一次请求的 trace id
	// TraceID() string
}

// WebsocketAPI websocket 接入地址
type WebsocketAPI interface {
	WS(ctx context.Context, params map[string]string, body string) (*WebsocketAP, error)
}

// MessageAPI 消息相关接口
type MessageAPI interface {
	Message(ctx context.Context, channelID string, messageID string) (*Message, error)
	PostMessage(ctx context.Context, channelID string, msg *MessageToCreate) (*Message, error)
}
