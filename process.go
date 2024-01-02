package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Processor is a struct to process message
type Processor struct {
	api OpenAPI
}

// ProcessMessage is a function to process message
func (p Processor) ProcessMessage(input string, data *Message) error {
	ctx := context.Background()
	cmd := ParseCommand(input)
	toCreate := &MessageToCreate{
		Content: "默认回复"}

	if cmd.Cmd == "加法运算" {
		toCreate.Content = genReplyContent(data, input)
		if _, err := p.api.PostMessage(ctx, data.ChannelID, toCreate); err != nil {
			log.Println(err)
		}
		return nil
	}

	return nil
}

func genReplyContent(data *Message, input string) string {
	text := strings.Split(input, " ")
	formula := text[1]

	var str = `你好：
	当前本地时间为：%s
	输入格式不正确，请重新输入！
	消息来自：%s
	`
	parts := strings.Split(formula, "+")
	if len(parts) != 2 {
		return fmt.Sprintf(
			str, time.Now().Format(time.RFC3339),
			getIP(),
		)
	}
	num1, err1 := strconv.Atoi(parts[0])
	num2, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return fmt.Sprintf(
			str, time.Now().Format(time.RFC3339),
			getIP(),
		)
	}
	sum := num1 + num2

	var tpl = `你好：
	当前本地时间为：%s
	加法算式为：%s
	计算结果为：%s
	消息来自：%s
	`
	return fmt.Sprintf(
		tpl, time.Now().Format(time.RFC3339), formula, strconv.Itoa(sum),
		getIP(),
	)
}

type CMD struct {
	Cmd     string
	Content string
}

var atRE = regexp.MustCompile(`<@!\d+>`)

const spaceCharSet = " \u00A0"

func ETLInput(input string) string {
	etlData := string(atRE.ReplaceAll([]byte(input), []byte("")))
	etlData = strings.Trim(etlData, spaceCharSet)
	return etlData
}

func ParseCommand(input string) *CMD {
	input = ETLInput(input)
	s := strings.Split(input, " ")
	if len(s) < 2 {
		return &CMD{
			Cmd:     strings.Trim(input, spaceCharSet),
			Content: "",
		}
	}
	return &CMD{
		Cmd:     strings.Trim(s[0], spaceCharSet),
		Content: strings.Join(s[1:], " "),
	}
}
