QQ频道机器人方案设计文档

1、通过加载yaml文件初始化botToken对象{appid，token}。

2、调用botgo.NewOpenAPI接口获取openapi实例。

3、通过接入地址调用api.WS获取一个websocket。

4、注册websocket事件，@机器人事件、连接成功回调、连接关闭回调。

5、设计信息处理函数，判断固定指令，计算相应算式，给出结果。



相关结构体与接口定义：

```golang
const (
	TypeBot    Type = "Bot"
	TypeNormal Type = "Bearer"
)

// Token 用于调用接口的 token 结构
type Token struct {
	AppID       uint64
	AccessToken string
	Type        Type
}

// New 创建一个新的 Token
func New(tokenType Type) *Token {
	return &Token{
		Type: tokenType,
	}
}

// BotToken 机器人身份的 token
func BotToken(appID uint64, accessToken string) *Token {
	return &Token{
		AppID:       appID,
		AccessToken: accessToken,
		Type:        TypeBot,
	}
}
```

```golang
// MessageToCreate 发送消息结构体定义
type MessageToCreate struct {
	Content string `json:"content,omitempty"`
	// 要回复的消息id，为空是主动消息，公域机器人会异步审核，不为空是被动消息，公域机器人会校验语料
	MsgID            string            `json:"msg_id,omitempty"`
	MessageReference *MessageReference `json:"message_reference,omitempty"`
	EventID string `json:"event_id,omitempty"` // 要回复的事件id, 逻辑同MsgID
}

// MessageReference 引用消息
type MessageReference struct {
	MessageID             string `json:"message_id"`               // 消息 id
	IgnoreGetMessageError bool   `json:"ignore_get_message_error"` // 是否忽律获取消息失败错误
}
```

```golang
type Processor struct {
	api openapi.OpenAPI
}
```

```golang
func ReadyHandler() event.ReadyHandler//感知连接成功事件
func ErrorNotifyHandler() event.ErrorNotifyHandler//连接关闭事件
func ATMessageEventHandler() event.ATMessageEventHandler//处理@机器人信息回调事件
func (p Processor) ProcessMessage(input string, data *dto.WSATMessageData)//信息处理函数
func genReplyContent(data *dto.WSATMessageData, input string)//加法功能实现函数
func getIP()//获取IP函数

```

