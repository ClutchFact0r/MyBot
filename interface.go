package main

import (
	"context"
	"time"
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
	// Transport 透传请求
	Transport(ctx context.Context, method, url string, body interface{}) ([]byte, error)
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
