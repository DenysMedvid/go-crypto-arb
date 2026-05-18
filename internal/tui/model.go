package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"go-crypto-arb/internal/config"
	"go-crypto-arb/internal/exchange"
	"go-crypto-arb/internal/marketdata"
)

type Model struct {
	client         *Client
	refreshEvery   time.Duration
	snapshot       marketdata.Snapshot
	err            error
	width          int
	height         int
	paused         bool
	profitableOnly bool
	showHelp       bool
	detailOpen     bool
	selectedRow    int
	lastRefresh    time.Time
	latency        time.Duration
	currentView    string
	icons          IconSet
	tabTitles      map[string]string
	priceSymbols   map[string]int
}

type snapshotMsg struct {
	snapshot marketdata.Snapshot
	latency  time.Duration
	err      error
}

type tickMsg time.Time

func NewModel(cfg config.Config, env config.Env) Model {
	interval := cfg.TUI.RefreshInterval.Duration
	if interval <= 0 {
		interval = 2 * time.Second
	}
	defaultView := cfg.TUI.DefaultView
	if defaultView == "" {
		defaultView = "crypto_dashboard"
	}
	return Model{
		client:       NewClient(cfg.TUI.BackendURL, env.APIKey),
		refreshEvery: interval,
		width:        100,
		currentView:  defaultView,
		icons:        NewIconSet(cfg.TUI.UseEmoji),
		tabTitles:    tabTitles(cfg),
		priceSymbols: configuredPriceSymbols(cfg),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fetchSnapshot(), m.nextTick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m, m.fetchSnapshot()
		case "p":
			m.paused = !m.paused
			return m, nil
		case "f":
			m.profitableOnly = !m.profitableOnly
			m.selectedRow = clampSelection(m.selectedRow, m.opportunityCount())
			return m, nil
		case "up":
			if m.selectedRow > 0 {
				m.selectedRow--
			}
			return m, nil
		case "down":
			count := m.opportunityCount()
			if count > 0 && m.selectedRow < count-1 {
				m.selectedRow++
			}
			return m, nil
		case "enter":
			if m.opportunityCount() > 0 {
				m.detailOpen = true
			}
			return m, nil
		case "esc":
			m.detailOpen = false
			return m, nil
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "1":
			m.currentView = "crypto_dashboard"
			m.selectedRow = clampSelection(m.selectedRow, m.opportunityCount())
			return m, nil
		case "2":
			m.currentView = "triangular"
			m.selectedRow = clampSelection(m.selectedRow, m.opportunityCount())
			return m, nil
		case "3":
			m.currentView = "cross_exchange"
			m.selectedRow = clampSelection(m.selectedRow, m.opportunityCount())
			return m, nil
		case "4":
			m.currentView = "spot_futures"
			m.selectedRow = clampSelection(m.selectedRow, m.opportunityCount())
			return m, nil
		case "5":
			m.currentView = "signals"
			m.selectedRow = clampSelection(m.selectedRow, m.opportunityCount())
			return m, nil
		case "6":
			m.currentView = "alerts"
			m.selectedRow = clampSelection(m.selectedRow, m.opportunityCount())
			return m, nil
		case "7":
			m.currentView = "health"
			m.selectedRow = clampSelection(m.selectedRow, m.opportunityCount())
			return m, nil
		case "8":
			m.currentView = "ibkr"
			m.selectedRow = clampSelection(m.selectedRow, m.opportunityCount())
			return m, nil
		}
	case snapshotMsg:
		m.err = msg.err
		if msg.err == nil {
			m.snapshot = msg.snapshot
			m.latency = msg.latency
			m.lastRefresh = time.Now()
			m.selectedRow = clampSelection(m.selectedRow, m.opportunityCount())
		}
		return m, nil
	case tickMsg:
		if m.paused {
			return m, m.nextTick()
		}
		return m, tea.Batch(m.fetchSnapshot(), m.nextTick())
	}
	return m, nil
}

func tabTitles(cfg config.Config) map[string]string {
	out := make(map[string]string)
	for name, tab := range cfg.TUI.Tabs {
		out[name] = tab.Title
	}
	return out
}

func configuredPriceSymbols(cfg config.Config) map[string]int {
	out := make(map[string]int)
	universeNames := []string{"crypto_spot"}
	if cfg.Strategies.CryptoTriangular.Universe != "" {
		universeNames = append([]string{cfg.Strategies.CryptoTriangular.Universe}, universeNames...)
	}
	for _, universeName := range universeNames {
		universe, ok := cfg.InstrumentUniverses[universeName]
		if !ok {
			continue
		}
		for _, instrument := range universe.Instruments {
			if instrument.Symbol == "" {
				continue
			}
			if instrument.MarketType != "" && instrument.MarketType != string(exchange.MarketSpot) {
				continue
			}
			symbol := exchange.NormalizeCanonicalSymbol(instrument.Symbol)
			if _, exists := out[symbol]; !exists {
				out[symbol] = len(out)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return out
}

func clampSelection(selected int, count int) int {
	if count <= 0 {
		return 0
	}
	if selected >= count {
		return count - 1
	}
	if selected < 0 {
		return 0
	}
	return selected
}

func (m Model) fetchSnapshot() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		snapshot, latency, err := m.client.Snapshot(ctx)
		return snapshotMsg{snapshot: snapshot, latency: latency, err: err}
	}
}

func (m Model) nextTick() tea.Cmd {
	return tea.Tick(m.refreshEvery, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
