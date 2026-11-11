package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"docker-tg-bot/internal/bot"
	"docker-tg-bot/internal/config"
	"docker-tg-bot/internal/docker"
)

const dockerTimeout = 15 * time.Second

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svc := bot.NewService(docker.NewClient(dockerTimeout), cfg.AllowedUserID)
	if err := bot.Run(ctx, cfg.Token, svc); err != nil {
		log.Fatal(err)
	}
}
