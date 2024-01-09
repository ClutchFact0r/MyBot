package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	_ "github.com/mattn/go-sqlite3"
)

// Processor is a struct to process message
type Processor struct {
	api OpenAPI
}

var IdiomUsers = map[string]IdiomUser{}

type IdiomUser struct {
	score    int
	lastward rune
}

const (
	start int = 1
	stop  int = 0
)

var size int //成语长度

// ProcessMessage is a function to process message
func (p Processor) ProcessMessage(input string, data *Message) error {
	fmt.Printf("data.ChannelID: %v\n", data.ChannelID)
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
	case "成语接龙玩家积分榜":
		toCreate.Content = ShowIdiomTable()
		if _, err := p.api.PostMessage(ctx, data.ChannelID, toCreate); err != nil {
			log.Println(err)
		}
	default:
		_, exist := IdiomUsers[data.Author.Username]
		if exist {
			toCreate.Content = genAnsweridiom(data, input)
			if _, err := p.api.PostMessage(ctx, data.ChannelID, toCreate); err != nil {
				log.Println(err)
			}
		} else {
			toCreate.Content = "抱歉，此指令未知！"
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
			RecipientID:   data.Author.Username,
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

func ShowIdiomTable() string {
	var str = `你好，成语接龙玩家积分榜如下：%s
	`
	info := PrintScoreTable()
	return fmt.Sprintf(str, info)
}

func genReplyidiom(data *Message, input string) string {

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

	last, size := utf8.DecodeLastRuneInString(randomValue)
	IdiomUsers[data.Author.Username] = IdiomUser{
		score:    0,
		lastward: last,
	}
	if IdiomUsers[data.Author.Username].lastward == utf8.RuneError && (size == 0 || size != len(randomValue)) {
		fmt.Println("字符串为空或不是有效的UTF-8编码")
	}
	return fmt.Sprintf(str, randomValue, IdiomUsers[data.Author.Username].lastward)
}

// 用户游戏独立，每日计算积分规则
// 推送每日用户积分
// sql
func genAnsweridiom(data *Message, input string) string {
	var context = ""
	firstword, size := utf8.DecodeRuneInString(input)
	if firstword == utf8.RuneError && (size == 0 || size != len(input)) {
		fmt.Println("字符串为空或不是有效的UTF-8编码")
	}
	if IdiomUsers[data.Author.Username].lastward != firstword {
		context = `你好：
		你给出的成语不符合成语接龙规则，请给出以"%c"开头的成语:
		`
		return fmt.Sprintf(context, IdiomUsers[data.Author.Username].lastward)
	} else {
		user := IdiomUsers[data.Author.Username]
		user.score++
		IdiomUsers[data.Author.Username] = user
		fmt.Printf("IdiomUsers[data.Author.Username].score: %v\n", IdiomUsers[data.Author.Username].score)
	}

	inputFirstWord, size := utf8.DecodeLastRuneInString(input)
	inputLastWord, size := utf8.DecodeLastRuneInString(input)
	if inputLastWord == utf8.RuneError && (size == 0 || size != len(input)) {
		fmt.Println("字符串为空或不是有效的UTF-8编码")
	}

	ans, exists := idiomsMap[string(inputFirstWord)]
	if exists {
		context = `你好：
		我给出的成语是：%s
		请继续接龙，给出一个"%c"开头的成语:
		`
		user := IdiomUsers[data.Author.Username]
		user.lastward, size = utf8.DecodeLastRuneInString(ans)
		if user.lastward == utf8.RuneError && (size == 0 || size != len(ans)) {
			fmt.Println("字符串为空或不是有效的UTF-8编码")
		}
		IdiomUsers[data.Author.Username] = user
		return fmt.Sprintf(
			context, ans, IdiomUsers[data.Author.Username].lastward,
		)
	} else {
		context = `词库中未找到一个以"%c"开头的成语，游戏结束！`
		fmt.Printf("user.score: %v\n", IdiomUsers[data.Author.Username].score)
		err := UpdateScore(data.Author.Username, IdiomUsers[data.Author.Username].score)
		delete(idiomsMap, data.Author.Username)
		if err != nil {
			fmt.Println("Error inserting user:", err)
		}
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

func (p Processor) DailyPush() {
	var printed18 int32
	checkAndPrint18 := func() {
		now := time.Now()
		hour, minute := now.Hour(), now.Minute()
		if hour == 16 && minute == 11 && atomic.CompareAndSwapInt32(&printed18, 0, 1) {
			fmt.Println("打印！！！")
			ctx := context.Background()
			toCreate := &MessageToCreate{
				Content: "默认回复"}
			toCreate.Content = ShowIdiomTable()
			if _, err := p.api.PostMessage(ctx, "634993940", toCreate); err != nil {
				log.Println(err)
			}
			DeleteTable()
		}
	}

	// 每天重置标志位的函数
	resetFlags := func() {
		now := time.Now()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		sleepDuration := nextMidnight.Sub(now)
		time.Sleep(sleepDuration)
		atomic.StoreInt32(&printed18, 0)
	}

	// 启动定时任务1：每隔1秒检查一次是否是18点整
	go func() {
		for range time.Tick(1 * time.Second) {
			checkAndPrint18()
		}
	}()

	// 启动定时任务以每天重置标志位
	go func() {
		for {
			resetFlags()
		}
	}()
}

// 			ctx := context.Background()
// 			toCreate := &MessageToCreate{
// 				Content: "默认回复"}
// 			toCreate.Content = ShowIdiomTable()
// 			if _, err := p.api.PostMessage(ctx, "sx1cb4f6b7", toCreate); err != nil {
// 				log.Println(err)
// 			}
// 			DeleteTable()
