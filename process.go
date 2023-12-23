package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/message"
	"github.com/tencent-connect/botgo/openapi"
)

// Processor is a struct to process message
type Processor struct {
	api openapi.OpenAPI
}

// ProcessMessage is a function to process message
func (p Processor) ProcessMessage(input string, data *dto.WSATMessageData) error {
	ctx := context.Background()
	cmd := message.ParseCommand(input)
	toCreate := &dto.MessageToCreate{
		Content: "默认回复" + message.Emoji(307),
		MessageReference: &dto.MessageReference{
			MessageID:             data.ID,
			IgnoreGetMessageError: true,
		},
	}

	if cmd.Cmd == "图片" {
		toCreate.Content = genReplyContent(data)
		if _, err := p.api.PostMessage(ctx, data.ChannelID, toCreate); err != nil {
			log.Println(err)
		}
		return nil
	}

	return nil
}

func genReplyContent(data *dto.WSATMessageData) string {
	var tpl = `你好：
	当前本地时间为：%s

	消息来自：%s
	`

	return fmt.Sprintf(
		tpl, time.Now().Format(time.RFC3339),
		getIP(),
	)
}
