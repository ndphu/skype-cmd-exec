package bot

import (
	"errors"
	"github.com/ndphu/message-handler-lib/bot"
	"github.com/ndphu/message-handler-lib/broker"
)

type Bot struct {
	BotId string `json:"botId"`
}

func (b *Bot) SendText(target, message string) error {
	_, err := b.action(bot.ActionSendText)(target, message)
	return err
}

func (b *Bot) SendImage(target, imageUrl string) error {
	_, err := b.action(bot.ActionSendImage)(target, imageUrl)
	return err
}

func exec(botId, action string, args ...string) (interface{}, error) {
	req := broker.RpcRequest{
		Method: action,
		Args:   args,
	}
	rc, err := broker.NewRpcClient(botId)
	if err != nil {
		return "", err
	}
	receive, err := rc.SendAndReceive(&req)
	if err != nil {
		return "", err
	}
	if receive.Success {
		return receive.Response, nil
	}
	return nil, errors.New(receive.Error)
}

func (b *Bot) action(action string) func(...string) (interface{}, error) {
	return func(args ...string) (interface{}, error) {
		return exec(b.BotId, action, args...)
	}
}
