package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func Run(ctx context.Context, token string, svc *Service) error {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return fmt.Errorf("failed to run bot: %w", err)
	}
	log.Printf("bot started as @%s", api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := api.GetUpdatesChan(u)

	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			api.StopReceivingUpdates()
			return nil
		case upd, ok := <-updates:
			if !ok {
				return nil
			}
			msg := upd.Message
			if msg == nil || msg.From == nil {
				continue
			}
			if !svc.Authorized(msg.From.ID) {
				log.Printf("user rejected id=%d", msg.From.ID)
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				handle(ctx, api, svc, msg)
			}()
		}
	}
}

func handle(ctx context.Context, api *tgbotapi.BotAPI, svc *Service, msg *tgbotapi.Message) {
	var reply string
	if msg.IsCommand() {
		reply = svc.HandleCommand(ctx, msg.Command(), strings.Fields(msg.CommandArguments()))
	} else {
		reply = svc.HandleText(ctx, msg.Text)
	}

	m := tgbotapi.NewMessage(msg.Chat.ID, reply)
	m.ParseMode = tgbotapi.ModeHTML
	if _, err := api.Send(m); err != nil {
		log.Printf("sending failed: %v", err)
	}
}
