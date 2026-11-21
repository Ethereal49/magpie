package agent

import (
	"path/filepath"
	"testing"

	"github.com/yetone/magpie/internal/edit"
)

func TestClaudePickerAfterDisconnect(t *testing.T) {
	const magpie = `{"options":[{"model":"test-relay/model","description":"Test via magpie"}],"replaceBuiltInOptions":true}`
	const user = `{"options":[{"model":"test-relay/model","description":"Test via magpie"},{"model":"mine","description":"My model"}]}`
	for _, action := range []string{"disconnect", "sync"} {
		for _, tc := range []struct {
			name, picker string
			endpoint     string
			marked, keep bool
		}{
			{"marked", magpie, "", true, false},
			{"unmarked", magpie, "", false, false},
			{"user-edited", user, "", true, true},
			{"other-endpoint", magpie, "http://127.0.0.1:1", true, true},
		} {
			t.Run(action+"/"+tc.name, func(t *testing.T) {
				home := syncHome(t)
				path := filepath.Join(home, ".claude", "settings.json")
				before := `{"theme":"dark","modelPicker":` + tc.picker
				if tc.endpoint != "" {
					before += `,"env":{"ANTHROPIC_BASE_URL":"` + tc.endpoint + `"}`
				}
				before += "}"
				writeFile(t, path, before)
				marks := map[string]string{"claude.disconnected": "1"}
				if tc.marked {
					marks["claude.model_picker"] = "1"
				}
				stash(marks)
				a := claude(home)
				if a.Wired() {
					t.Fatal("fixture must already be disconnected")
				}

				run := a.Disconnect
				if action == "sync" {
					run = a.Sync
				}
				if err := run(); err != nil {
					t.Fatal(err)
				}
				if got, found := edit.GetJSON(path, "modelPicker"); found != tc.keep || found && got != tc.picker {
					t.Errorf("modelPicker = %q (present: %t), want present: %t", got, found, tc.keep)
				}
				if got := stashLoad()["claude.model_picker"]; !tc.keep && got != "" {
					t.Errorf("stale picker marker = %q", got)
				}
				if got, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL"); got != tc.endpoint {
					t.Errorf("endpoint = %q, want %q", got, tc.endpoint)
				}
				if theme, _ := edit.GetJSON(path, "theme"); theme != "dark" {
					t.Errorf("theme = %q, want dark", theme)
				}
			})
		}
	}
}
