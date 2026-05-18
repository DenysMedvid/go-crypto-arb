package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go-crypto-arb/internal/app"
	"go-crypto-arb/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if len(os.Args) > 1 && os.Args[1] == "validate-config" {
		runValidateConfig(logger)
		return
	}
	env, err := config.LoadEnv()
	if err != nil {
		logger.Error("load env", "error", err)
		os.Exit(1)
	}
	cfg, err := config.Load(env.ConfigPath)
	if err != nil {
		logger.Error("load config", "path", env.ConfigPath, "error", err)
		os.Exit(1)
	}
	cfg.ApplyEnv(env)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.New(cfg, env, logger).Run(ctx); err != nil {
		logger.Error("run api", "error", err)
		os.Exit(1)
	}
}

func runValidateConfig(logger *slog.Logger) {
	flags := flag.NewFlagSet("validate-config", flag.ExitOnError)
	configPath := flags.String("config", "", "path to config.yaml")
	_ = flags.Parse(os.Args[2:])
	env, err := config.LoadEnv()
	if err != nil {
		logger.Error("load env", "error", err)
		os.Exit(1)
	}
	if *configPath != "" {
		env.ConfigPath = *configPath
	}
	cfg, err := config.Load(env.ConfigPath)
	if err != nil {
		logger.Error("load config", "path", env.ConfigPath, "error", err)
		os.Exit(1)
	}
	cfg.ApplyEnv(env)
	messages := config.Validate(cfg, env, nil)
	for _, message := range messages {
		fmt.Printf("%s: %s\n", message.Level, message.Message)
	}
	if config.HasValidationErrors(messages) {
		os.Exit(1)
	}
}
