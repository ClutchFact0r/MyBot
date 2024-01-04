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
	DirectMessageAPI
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
	PostDirectMessage(ctx context.Context, dm *DirectMessage, msg *MessageToCreate) (*Message, error)
}

// DirectMessageAPI 信息相关接口
type DirectMessageAPI interface {
	// CreateDirectMessage 创建私信频道
	CreateDirectMessage(ctx context.Context, dm *DirectMessageToCreate) (*DirectMessage, error)
	// PostDirectMessage 在私信频道内发消息
	PostDirectMessage(ctx context.Context, dm *DirectMessage, msg *MessageToCreate) (*Message, error)
}
