package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/tui"
)

func main() {
	env, err := config.LoadEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load env: %v\n", err)
		os.Exit(1)
	}
	cfg, err := config.Load(env.ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config %s: %v\n", env.ConfigPath, err)
		os.Exit(1)
	}
	cfg.ApplyEnv(env)
	program := tea.NewProgram(tui.NewModel(cfg, env), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "run tui: %v\n", err)
		os.Exit(1)
	}
}
