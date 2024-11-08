package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

type Bot struct {
	*tgbotapi.BotAPI
}

func NewBot(token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
		return nil, err
	}
	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	return &Bot{bot}, nil
}

func (b *Bot) Update() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.GetUpdatesChan(u)

	for update := range updates {
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Welcome!")
				_, err := b.Send(msg)
				if err != nil {
					return err
				}
			case "help":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "List of available commands: /start, /help")
				_, err := b.Send(msg)
				if err != nil {
					return err
				}
			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Unknown command")
				_, err := b.Send(msg)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
