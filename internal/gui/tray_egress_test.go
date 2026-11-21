package gui

import (
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/egress"
)

// The tray's tooltip begins with a warning while the Claude accounts' way
// out is in a region Anthropic doesn't list, in magpie's language; one in
// a supported region, or one not known, says nothing.
func TestTrayEgressTip(t *testing.T) {
	cn := egress.Result{State: egress.Unsupported, Loc: "CN"}
	for lang, want := range map[string]string{
		"en":    "Claude API route check: CN is outside Anthropic's supported regions",
		"zh":    "Claude API 出口检测：CN 不在 Anthropic 支持的地区内",
		"zh-TW": "Claude API 網路出口檢測：CN 不在 Anthropic 支援的地區內",
		"ja":    "Claude API の接続元チェック：CN は Anthropic のサポート対象地域外です",
		"de":    "Claude-API-Verbindungsprüfung: CN liegt außerhalb der von Anthropic unterstützten Regionen",
	} {
		got := trayEgressTip(lang, cn)
		if !strings.Contains(got, want) {
			t.Errorf("%s: %q", lang, got)
		}
		for _, w := range []string{"ban", "封", "安全", "sicher"} {
			if strings.Contains(got, w) {
				t.Errorf("%s: the warning says %q: %s", lang, w, got)
			}
		}
	}
	for _, r := range []egress.Result{{State: egress.OK, Loc: "US"}, {State: egress.Unknown}, {State: egress.Unknown, Loc: "XX"}} {
		if got := trayEgressTip("en", r); got != "" {
			t.Errorf("%+v warned %q", r, got)
		}
	}
}
