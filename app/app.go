package app

import (
	"context"
	"fmt"
	"github.com/alexpain/barbuddy/bots"
	"github.com/alexpain/barbuddy/config"
	"github.com/alexpain/barbuddy/database"
	"golang.org/x/sync/errgroup"
	"log"
)

type Application struct {
	conf *config.Config
	bot  *bots.Bot
	db   *database.Database
}

func New(ctx context.Context, conf *config.Config) (*Application, error) {
	db, err := database.New("./cocktails.db")
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}

	bot, err := bots.NewTelegramBot(conf.Bot.Token, db)
	if err != nil {
		return nil, err
	}

	if err := db.CreateTable(); err != nil {
		log.Fatalf("Error creating table: %v", err)
	}
	fmt.Println("Table created or already exists.")

	return &Application{
		conf: conf,
		bot:  bot,
		db:   db,
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
	a.db.Stop()

	return nil
}
