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
	// 消息中的提醒信息(@)列表
	Mentions []*User `json:"mentions"`
}

// DirectMessage 私信结构定义，一个 DirectMessage 为两个用户之间的一个私信频道，简写为 DM
type DirectMessage struct {
	// 频道ID
	GuildID string `json:"guild_id"`
	// 子频道id
	ChannelID string `json:"channel_id"`
	// 私信频道创建的时间戳
	CreateTime string `json:"create_time"`
}

// DirectMessageToCreate 创建私信频道的结构体定义
type DirectMessageToCreate struct {
	// 频道ID
	SourceGuildID string `json:"source_guild_id"`
	// 用户ID
	RecipientID string `json:"recipient_id"`
}

// MessageToCreate 发送消息结构体定义
type MessageToCreate struct {
	Content string `json:"content,omitempty"`
	Image   string `json:"image,omitempty"`
	MsgID   string `json:"msg_id,omitempty"`
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
const dmsURI string = "/dms/{guild_id}/messages"
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

// PostDirectMessage 在私信频道内发消息
func (o *openAPI) PostDirectMessage(ctx context.Context,
	dm *DirectMessage, msg *MessageToCreate) (*Message, error) {
	resp, err := o.request(ctx).
		SetResult(Message{}).
		SetPathParam("guild_id", dm.GuildID).
		SetBody(msg).
		Post(o.getURL(dmsURI))
	if err != nil {
		return nil, err
	}
	return resp.Result().(*Message), nil
}
