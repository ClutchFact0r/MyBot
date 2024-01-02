package main

import (
	"context"
	"fmt"
	"log"
	"path"
	"runtime"
	"strings"
	"time"
)

// 消息处理器，持有 openapi 对象
var processor Processor

func main() {
	ctx := context.Background()
	// 加载 appid 和 token
	botToken := New(TypeBot)
	if err := botToken.LoadFromConfig(getConfigPath("config.yaml")); err != nil {
		log.Fatalln(err)
	}

	// 初始化 openapi，正式环境
	api := NewOpenAPI(botToken).WithTimeout(3 * time.Second)

	// 获取 websocket 信息
	wsInfo, err := api.WS(ctx, nil, "")
	if err != nil {
		log.Fatalln(err)
	}

	processor = Processor{api: api}
	// 根据不同的回调，生成 intents
	intent := RegisterHandlers(
		// at 机器人事件，目前是在这个事件处理中有逻辑，会回消息，其他的回调处理都只把数据打印出来，不做任何处理
		ATMessageEventHandler(),
		// 如果想要捕获到连接成功的事件，可以实现这个回调
		ReadyHandler(),
		// 连接关闭回调
		ErrorNotifyHandler(),
	)
	
	if err = NewSessionManager().Start(wsInfo, botToken, &intent); err != nil {
		log.Fatalln(err)
	}
}

// ReadyHandler 自定义 ReadyHandler 感知连接成功事件
func ReadyHandler() ReadyHandlers {
	return func(event *WSPayload, data *WSReadyData) {
		log.Println("ready event receive: ", data)
	}
}

func ErrorNotifyHandler() ErrorNotifyHandlers {
	return func(err error) {
		log.Println("error notify receive: ", err)
	}
}

// ATMessageEventHandler 实现处理 at 消息的回调
func ATMessageEventHandler() ATMessageEventHandlers {
	return func(event *WSPayload, data *Message) error {
		input := strings.ToLower(ETLInput(data.Content))
		return processor.ProcessMessage(input, data)
	}
}

func getConfigPath(name string) string {
	_, filename, _, ok := runtime.Caller(1)
	if ok {
		return fmt.Sprintf("%s/%s", path.Dir(filename), name)
	}
	return ""
}
