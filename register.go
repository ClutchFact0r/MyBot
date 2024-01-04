package main

// DefaultHandlers 默认的 handler 结构，管理所有支持的 handler 类型
var DefaultHandlers struct {
	Ready       ReadyHandlers
	ErrorNotify ErrorNotifyHandlers

	Message       MessageEventHandler
	ATMessage     ATMessageEventHandlers
	DirectMessage DirectMessageEventHandler
}

// MessageEventHandler 消息事件 handler
type MessageEventHandler func(event *WSPayload, data *Message) error

// DirectMessageEventHandler 私信消息事件 handler
type DirectMessageEventHandler func(event *WSPayload, data *DirectMessage) error
