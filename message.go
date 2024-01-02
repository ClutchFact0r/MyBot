package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
)

type User struct {
	ID               string `json:"id"`
	Username         string `json:"username"`
	Avatar           string `json:"avatar"`
	Bot              bool   `json:"bot"`
	UnionOpenID      string `json:"union_openid"`       // 特殊关联应用的 openid
	UnionUserAccount string `json:"union_user_account"` // 机器人关联的用户信息，与union_openid关联的应用是同一个
}

type Member struct {
	GuildID  string   `json:"guild_id"`
	JoinedAt string   `json:"joined_at"`
	Nick     string   `json:"nick"`
	User     *User    `json:"user"`
	Roles    []string `json:"roles"`
	OpUserID string   `json:"op_user_id,omitempty"`
}

// Message 消息结构体定义
type Message struct {
	// 消息ID
	ID string `json:"id"`
	// 子频道ID
	ChannelID string `json:"channel_id"`
	// 频道ID
	GuildID string `json:"guild_id"`
	// 内容
	Content string `json:"content"`
	// 发送时间
	Timestamp string `json:"timestamp"`
	// 消息编辑时间
	EditedTimestamp string `json:"edited_timestamp"`
	// 是否@all
	MentionEveryone bool `json:"mention_everyone"`
	// 消息发送方
	Author *User `json:"author"`
	// 消息发送方Author的member属性，只是部分属性
	Member *Member `json:"member"`
	// 附件
	// Attachments []*MessageAttachment `json:"attachments"`
	// 结构化消息-embeds
	// Embeds []*Embed `json:"embeds"`
	// 消息中的提醒信息(@)列表
	Mentions []*User `json:"mentions"`
	// ark 消息
	// Ark *Ark `json:"ark"`
	// 私信消息
	// DirectMessage bool `json:"direct_message"`
	// 子频道 seq，用于消息间的排序，seq 在同一子频道中按从先到后的顺序递增，不同的子频道之前消息无法排序
	// SeqInChannel string `json:"seq_in_channel"`
	// 引用的消息
	// MessageReference *MessageReference `json:"message_reference,omitempty"`
	// 私信场景下，该字段用来标识从哪个频道发起的私信
	// SrcGuildID string `json:"src_guild_id"`
}

// MessageToCreate 发送消息结构体定义
type MessageToCreate struct {
	Content string `json:"content,omitempty"`
	// Embed   *Embed `json:"embed,omitempty"`
	// Ark     *Ark   `json:"ark,omitempty"`
	Image string `json:"image,omitempty"`
	// 要回复的消息id，为空是主动消息，公域机器人会异步审核，不为空是被动消息，公域机器人会校验语料
	MsgID string `json:"msg_id,omitempty"`
	// MessageReference *MessageReference         `json:"message_reference,omitempty"`
	// Markdown         *Markdown                 `json:"markdown,omitempty"`
	// Keyboard         *keyboard.MessageKeyboard `json:"keyboard,omitempty"` // 消息按钮组件
	EventID string `json:"event_id,omitempty"` // 要回复的事件id, 逻辑同MsgID
}

const (
	messageURI  string = "/channels/{channel_id}/messages/{message_id}"
	messagesURI string = "/channels/{channel_id}/messages"
)

// Message 拉取单条消息
func (o *openAPI) Message(ctx context.Context, channelID string, messageID string) (*Message, error) {
	resp, err := o.request(ctx).
		SetResult(Message{}).
		SetPathParam("channel_id", channelID).
		SetPathParam("message_id", messageID).
		Get(o.getURL(messageURI))
	if err != nil {
		return nil, err
	}

	// 兼容处理
	result := resp.Result().(*Message)
	if result.ID == "" {
		body := gjson.Get(resp.String(), "message")
		if err := json.Unmarshal([]byte(body.String()), result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

const domain = "api.sgroup.qq.com"
const sandBoxDomain = "sandbox.api.sgroup.qq.com"

const scheme = "https"

func (o *openAPI) getURL(endpoint string) string {
	d := domain
	if o.sandbox {
		d = sandBoxDomain
	}
	return fmt.Sprintf("%s://%s%s", scheme, d, endpoint)
}

func (o *openAPI) PostMessage(ctx context.Context, channelID string, msg *MessageToCreate) (*Message, error) {
	resp, err := o.request(ctx).
		SetResult(Message{}).
		SetPathParam("channel_id", channelID).
		SetBody(msg).
		Post(o.getURL(messagesURI))
	if err != nil {
		return nil, err
	}

	return resp.Result().(*Message), nil
}

type WSPayload struct {
	WSPayloadBase
	Data       interface{} `json:"d,omitempty"`
	RawMessage []byte      `json:"-"` // 原始的 message 数据
}

// WSPayloadBase 基础消息结构，排除了 data
type WSPayloadBase struct {
	OPCode int       `json:"op"`
	Seq    uint32    `json:"s,omitempty"`
	Type   EventType `json:"t,omitempty"`
}
