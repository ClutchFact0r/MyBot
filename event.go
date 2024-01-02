package main

type ReadyHandlers func(event *WSPayload, data *WSReadyData)
type ErrorNotifyHandlers func(err error)
type ATMessageEventHandlers func(event *WSPayload, data *Message) error

type EventType string

// Intent 类型
type Intent int

// 事件类型
const (
	EventMessageCreate         EventType = "MESSAGE_CREATE"
	EventMessageReactionAdd    EventType = "MESSAGE_REACTION_ADD"
	EventMessageReactionRemove EventType = "MESSAGE_REACTION_REMOVE"
	EventAtMessageCreate       EventType = "AT_MESSAGE_CREATE"
	EventPublicMessageDelete   EventType = "PUBLIC_MESSAGE_DELETE"
)

// intentEventMap 不同 intent 对应的事件定义
var intentEventMap = map[Intent][]EventType{

	IntentGuildAtMessage: {EventAtMessageCreate, EventPublicMessageDelete},
}

// websocket intent 声明
const (
	IntentGuilds         Intent = 1 << iota
	IntentGuildAtMessage Intent = 1 << 30 // 只接收@消息事件

	IntentNone Intent = 0
)

var eventIntentMap = transposeIntentEventMap(intentEventMap)

// RegisterHandlers 注册事件回调，并返回 intent 用于 websocket 的鉴权
func RRegisterHandlers(handlers ...interface{}) Intent {
	var i Intent
	for _, h := range handlers {
		switch handle := h.(type) {
		case ReadyHandlers:
			DefaultHandlers.Ready = handle
		case ErrorNotifyHandlers:
			DefaultHandlers.ErrorNotify = handle
		default:
		}
	}
	i = i | registerMessageHandlers(i, handlers...)
	return i
}

// EventToIntent 事件转换对应的Intent
func EventToIntent(events ...EventType) Intent {
	var i Intent
	for _, event := range events {
		i = i | eventIntentMap[event]
	}
	return i
}

// registerMessageHandlers 注册消息相关的 handler
func registerMessageHandlers(i Intent, handlers ...interface{}) Intent {
	for _, h := range handlers {
		switch handle := h.(type) {
		case ATMessageEventHandlers:
			DefaultHandlers.ATMessage = handle
			i = i | EventToIntent(EventAtMessageCreate)
		default:
		}
	}
	return i
}

func transposeIntentEventMap(input map[Intent][]EventType) map[EventType]Intent {
	result := make(map[EventType]Intent)
	for i, eventTypes := range input {
		for _, s := range eventTypes {
			result[s] = i
		}
	}
	return result
}
