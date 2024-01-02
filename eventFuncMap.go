package main

var eventParseFuncMap = map[int]map[EventType]eventParseFunc{
	WSDispatchEvent: {
		EventAtMessageCreate: atMessageHandler,
	},
}

type eventParseFunc func(event *WSPayload, message []byte) error

func atMessageHandler(payload *WSPayload, message []byte) error {
	data := &Message{}
	if err := ParseData(message, data); err != nil {
		return err
	}
	if DefaultHandlers.ATMessage != nil {
		return DefaultHandlers.ATMessage(payload, data)
	}
	return nil
}
