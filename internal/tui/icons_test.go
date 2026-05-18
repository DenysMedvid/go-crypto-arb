package tui

import (
	"strings"
	"testing"

	"go-crypto-arb/internal/config"
)

func TestIconHelperEmojiSet(t *testing.T) {
	icons := NewIconSet(true)
	if icons.IBKR == "" || icons.OK == "" || icons.Locked == "" {
		t.Fatalf("expected emoji icons, got %#v", icons)
	}
}

func TestIconHelperASCIISet(t *testing.T) {
	icons := NewIconSet(false)
	if icons.IBKR != "IBKR" || icons.OK != "OK" || icons.Locked != "LOCK" {
		t.Fatalf("expected ASCII icons, got %#v", icons)
	}
}

func TestRenderingDoesNotPanicWithEmojiDisabled(t *testing.T) {
	model := NewModel(config.Config{TUI: config.TUIConfig{UseEmoji: false}}, config.Env{})
	got := model.View()
	if !strings.Contains(got, "go-crypto-arb") {
		t.Fatalf("expected rendered app title, got %q", got)
	}
}
