package main

// DefaultHandlers 默认的 handler 结构，管理所有支持的 handler 类型
var DefaultHandlers struct {
	Ready       ReadyHandlers
	ErrorNotify ErrorNotifyHandlers

	Message   MessageEventHandler
	ATMessage ATMessageEventHandlers
}

// MessageEventHandler 消息事件 handler
type MessageEventHandler func(event *WSPayload, data *Message) error
