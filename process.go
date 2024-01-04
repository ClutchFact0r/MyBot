package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Processor is a struct to process message
type Processor struct {
	api OpenAPI
}

const (
	start int = 1
	stop  int = 0
)

var flag int = stop //是否正在成语接龙
var lastword rune   //用户需要给出的成语最后一个字
var size int        //成语长度

// ProcessMessage is a function to process message
func (p Processor) ProcessMessage(input string, data *Message) error {
	ctx := context.Background()
	cmd := ParseCommand(input)
	toCreate := &MessageToCreate{
		Content: "默认回复"}

	switch cmd.Cmd {
	case "hi":
		toCreate.Content = hiReply()
		if _, err := p.api.PostMessage(ctx, data.ChannelID, toCreate); err != nil {
			log.Println(err)
		}
	case "加法运算":
		toCreate.Content = genReplyContent(data, input)
		if _, err := p.api.PostMessage(ctx, data.ChannelID, toCreate); err != nil {
			log.Println(err)
		}
	case "成语接龙":
		toCreate.Content = genReplyidiom(data, input)
		if _, err := p.api.PostMessage(ctx, data.ChannelID, toCreate); err != nil {
			log.Println(err)
		}
	case "提示":
		toCreate.Content = hintIdiom(data, input)
		if _, err := p.api.PostMessage(ctx, data.ChannelID, toCreate); err != nil {
			log.Println(err)
		}
	case "私信":
		p.dmHandler(data)
		return nil
	default:
		if flag == start {
			toCreate.Content = genAnsweridiom(data, input)
			if _, err := p.api.PostMessage(ctx, data.ChannelID, toCreate); err != nil {
				log.Println(err)
			}
		}
	}
	return nil
}

func (p Processor) dmHandler(data *Message) {
	dm, err := p.api.CreateDirectMessage(
		context.Background(), &DirectMessageToCreate{
			SourceGuildID: data.GuildID,
			RecipientID:   data.Author.ID,
		},
	)
	if err != nil {
		log.Println(err)
		return
	}

	toCreate := &MessageToCreate{
		Content: "默认私信回复",
	}
	_, err = p.api.PostDirectMessage(
		context.Background(), dm, toCreate,
	)
	if err != nil {
		log.Println(err)
		return
	}
}

func hiReply() string {
	var str = `你好：
	欢迎使用问答机器人！
	本机器人可以为你提供加法运算和成语接龙服务！
	`
	return fmt.Sprintf(str)
}

func genReplyidiom(data *Message, input string) string {
	flag = start

	var str = `你好：
	即将开始成语接龙，我给出的成语是：%s
	请继续接龙，给出一个"%c"开头的成语。
	提示：可以@机器人使用“提示”指令+空格+汉字获取答案！
	例：@问答机器人提示 足，会给出“足”字开头的四字成语。 
	`

	// 初始化随机数种子
	rand.Seed(time.Now().UnixNano())
	var keys []string
	for k := range idiomsMap {
		keys = append(keys, k)
	}

	randomIndex := rand.Intn(len(keys))
	randomKey := keys[randomIndex]
	randomValue := idiomsMap[randomKey]

	lastword, size = utf8.DecodeLastRuneInString(randomValue)
	if lastword == utf8.RuneError && (size == 0 || size != len(randomValue)) {
		fmt.Println("字符串为空或不是有效的UTF-8编码")
	}
	return fmt.Sprintf(str, randomValue, lastword)
}

func genAnsweridiom(data *Message, input string) string {
	var context = ""
	firstword, size := utf8.DecodeRuneInString(input)
	if firstword == utf8.RuneError && (size == 0 || size != len(input)) {
		fmt.Println("字符串为空或不是有效的UTF-8编码")
	}
	if lastword != firstword {
		context = `你好：
		你给出的成语不符合成语接龙规则，请给出以"%c"开头的成语:
		`
		return fmt.Sprintf(context, lastword)
	}

	inputFirstWord := string(input[3])
	inputLastWord, size := utf8.DecodeLastRuneInString(input)
	if inputLastWord == utf8.RuneError && (size == 0 || size != len(input)) {
		fmt.Println("字符串为空或不是有效的UTF-8编码")
	}

	ans, exists := idiomsMap[inputFirstWord]
	if exists {
		context = `你好：
		我给出的成语是：%s
		请继续接龙，给出一个"%c"开头的成语:
		`
		lastword, size = utf8.DecodeLastRuneInString(ans)
		if lastword == utf8.RuneError && (size == 0 || size != len(ans)) {
			fmt.Println("字符串为空或不是有效的UTF-8编码")
		}

		return fmt.Sprintf(
			context, ans, lastword,
		)
	} else {
		flag = stop
		context = `词库中未找到一个以"%c"开头的成语，游戏结束！`
		return fmt.Sprintf(context, inputLastWord)
	}

}

func hintIdiom(data *Message, input string) string {
	text := strings.Split(input, " ")
	word := text[1]

	idiom, exists := idiomsMap[word]
	if exists {
		var str = `你好,以"%s"开头的成语有：%s
		`
		return fmt.Sprintf(
			str, word, idiom,
		)
	} else {
		var str = `抱歉，词库中未找到以"%s"开头的成语。`
		return fmt.Sprintf(str, word)
	}
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

const userMeDMURI string = "/users/@me/dms"

// CreateDirectMessage 创建私信频道
func (o *openAPI) CreateDirectMessage(ctx context.Context, dm *DirectMessageToCreate) (*DirectMessage, error) {
	resp, err := o.request(ctx).
		SetResult(DirectMessage{}).
		SetBody(dm).
		Post(o.getURL(userMeDMURI))
	if err != nil {
		return nil, err
	}
	return resp.Result().(*DirectMessage), nil
}
