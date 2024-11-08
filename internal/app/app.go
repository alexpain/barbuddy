package app

import (
	"context"
	"github.com/alexpain/barbuddy/internal/config"
	"github.com/alexpain/barbuddy/internal/telegram"
	"golang.org/x/sync/errgroup"
	"log"
)

type Application struct {
	conf *config.Config
	bot  *telegram.Bot
}

func New(ctx context.Context, conf *config.Config) (*Application, error) {
	bot, err := telegram.NewBot(conf.Bot.Token)
	if err != nil {
		return nil, err
	}

	return &Application{
		conf: conf,
		bot:  bot,
	}, nil
}

func (a *Application) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Printf("starting bot update...")
		if err := a.bot.Update(); err != nil {
			return err
		}
		return nil
	})

	return nil
}

func (a *Application) Stop(ctx context.Context) error {
	a.bot.StopReceivingUpdates()

	return nil
}
