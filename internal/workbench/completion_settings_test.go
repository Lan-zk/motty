package workbench

import (
	"encoding/json"
	"testing"

	"github.com/vivaui/viva/ui"
	"motty/internal/configstore"
	"motty/internal/i18n"
)

func TestInlineSettingDisablesGhostButKeepsManualCompletion(t *testing.T) {
	w := newCompletionWorld(t)
	w.type_("git statu")
	if _, ok := w.state.commandGhost(w.session); !ok {
		t.Fatal("the catalog did not offer a suggestion before disabling inline suggestions")
	}
	var saved savedState
	if err := json.Unmarshal([]byte(`{"command_inline_suggest_off":true}`), &saved); err != nil {
		t.Fatal(err)
	}
	w.state.restoreShared(saved)
	if rest, ok := w.state.commandGhost(w.session); ok {
		t.Fatalf("inline suggestions are disabled but %q is still offered", rest)
	}
	w.state.openCompletions()
	if !w.state.completion.list.active || len(w.state.completion.list.candidates) == 0 {
		t.Fatal("disabling inline suggestions also disabled the manual completion list")
	}
	// The command completion switch still controls both modes.
	w.state.toggleSuggestions()
	if w.state.completion.list.active {
		t.Fatal("turning command completion off left the list open")
	}
	w.state.openCompletions()
	if w.state.completion.list.active {
		t.Fatal("command completion is disabled but the list reopened")
	}
}

func TestCompletionControlsAndKeysHaveSeparateSettingsHomes(t *testing.T) {
	for _, language := range []i18n.Language{i18n.English, i18n.Chinese} {
		t.Run(language.String(), func(t *testing.T) {
			state := newTestState(t, testOptions(t))
			state.setLanguage(language)
			state.Click(hitSettings)
			state.selectSettingsCategory(settingsTerminal)
			var canvas ui.Canvas
			state.Paint(&canvas, ui.Size{Width: 1400, Height: 2400})
			if !canvasHasText(&canvas, state.messages.Completion.InlineRow) {
				t.Fatal("terminal settings have no separate inline suggestion switch")
			}
			switchOn := false
			for _, hit := range canvas.Hits() {
				if hit.ID == hitSettingsInlineSuggestToggle && hit.Role == ui.RoleSwitch && hit.Checked {
					switchOn = true
				}
			}
			if !switchOn {
				t.Fatal("the inline suggestion switch is absent or disabled by default")
			}
			state.selectSettingsCategory(settingsShortcuts)
			canvas.Reset()
			state.Paint(&canvas, ui.Size{Width: 1400, Height: 2400})
			if _, ok := hitByID(&canvas, hitShortcutRowList.At(int(shortcutCompletion-1))); !ok {
				t.Fatal("the shortcut for opening completions is not visible")
			}
			if !canvasHasText(&canvas, state.messages.Completion.Open) {
				t.Fatal("the configurable shortcut does not explain that it opens completions")
			}
			for _, help := range state.completionKeys() {
				if !canvasHasText(&canvas, help.label) || !canvasHasText(&canvas, help.keys) {
					t.Errorf("keyboard shortcuts do not explain %q: %q", help.label, help.keys)
				}
				for _, hit := range canvas.Hits() {
					if hit.Label == help.label {
						t.Errorf("fixed completion behavior %q is presented as an interactive shortcut", help.label)
					}
				}
			}
		})
	}
}

func TestInlineSuggestionSwitchPersistsIndependently(t *testing.T) {
	options := testOptions(t)
	state := newTestState(t, options)
	state.Click(hitSettingsInlineSuggestToggle)
	if !state.history.inlineOff || state.history.suggestOff {
		t.Fatal("the inline switch did not change independently of command completion")
	}
	again := newTestState(t, options)
	if !again.history.inlineOff || again.history.suggestOff {
		t.Fatal("the independent inline preference did not survive restart")
	}
	again.Click(hitSettingsInlineSuggestToggle)
	if again.history.inlineOff {
		t.Fatal("the inline switch cannot be re-enabled")
	}
}

func TestInlinePreferenceUsesAnIndependentSharedRecord(t *testing.T) {
	state := newTestState(t, testOptions(t))
	state.history.inlineOff = true
	entries, local, err := splitState(state.snapshot(), newConfigBase(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var record *configstore.Entry
	for index := range entries {
		if entry := &entries[index]; entry.Kind == kindSetting && entry.ID == "command_inline_suggest_off" {
			record = entry
		}
	}
	if record == nil {
		t.Fatal("the inline preference is not its own shared setting")
	}
	saved, _, err := joinState(entries, local)
	if err != nil || !saved.CommandInlineSuggestOff || saved.CommandSuggestOff {
		t.Fatalf("shared record round trip: inlineOff=%v completionOff=%v err=%v", saved.CommandInlineSuggestOff, saved.CommandSuggestOff, err)
	}
	var own map[string]json.RawMessage
	if err := json.Unmarshal(local, &own); err != nil {
		t.Fatal(err)
	}
	if _, ok := own[record.ID]; ok {
		t.Fatal("a shared inline preference was also written as device-local state")
	}
}

func TestCompletionKeyHelpIsSearchableInShortcuts(t *testing.T) {
	state := newTestState(t, testOptions(t))
	searchSettings(t, state, "Alt+→")
	if state.settings.count != 1 || state.settings.results[0].category != settingsShortcuts || state.settings.results[0].label != state.messages.Completion.KeysAcceptWord {
		t.Fatalf("word acceptance search results: %+v", state.settings.results[:state.settings.count])
	}
}

func canvasHasText(canvas *ui.Canvas, value string) bool {
	for _, text := range canvas.Texts() {
		if text.Value == value {
			return true
		}
	}
	return false
}
