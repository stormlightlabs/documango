package tui

import (
	"slices"
	"testing"
)

// TestNewKeyBindings verifies key bindings are created correctly
func TestNewKeyBindings(t *testing.T) {
	kb := newKeyBindings()

	if kb.Quit.Keys() == nil {
		t.Error("expected Quit binding")
	}

	if kb.Search.Keys() == nil {
		t.Error("expected Search binding")
	}

	if kb.Navigate.Keys() == nil {
		t.Error("expected Navigate binding")
	}

	if kb.Open.Keys() == nil {
		t.Error("expected Open binding")
	}

	if kb.Back.Keys() == nil {
		t.Error("expected Back binding")
	}

	if kb.Scroll.Keys() == nil {
		t.Error("expected Scroll binding")
	}

	if kb.Help.Keys() == nil {
		t.Error("expected Help binding")
	}

	if kb.Link.Keys() == nil {
		t.Error("expected Link binding")
	}
}

// TestKeyBindings_Quit verifies quit binding
func TestKeyBindings_Quit(t *testing.T) {
	kb := newKeyBindings()

	keys := kb.Quit.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 quit keys, got %d", len(keys))
	}

	hasQ := false
	hasCtrlC := false
	for _, k := range keys {
		if k == "q" {
			hasQ = true
		}
		if k == "ctrl+c" {
			hasCtrlC = true
		}
	}

	if !hasQ {
		t.Error("expected quit to include 'q'")
	}
	if !hasCtrlC {
		t.Error("expected quit to include 'ctrl+c'")
	}

	help := kb.Quit.Help()
	if help.Key != "q" {
		t.Errorf("expected help key 'q', got %q", help.Key)
	}
	if help.Desc != "quit" {
		t.Errorf("expected help desc 'quit', got %q", help.Desc)
	}
}

// TestKeyBindings_Search verifies search binding
func TestKeyBindings_Search(t *testing.T) {
	kb := newKeyBindings()

	keys := kb.Search.Keys()
	if len(keys) != 1 || keys[0] != "/" {
		t.Errorf("expected search key '/', got %v", keys)
	}

	help := kb.Search.Help()
	if help.Key != "/" {
		t.Errorf("expected help key '/', got %q", help.Key)
	}
	if help.Desc != "search" {
		t.Errorf("expected help desc 'search', got %q", help.Desc)
	}
}

// TestKeyBindings_Navigate verifies navigate binding
func TestKeyBindings_Navigate(t *testing.T) {
	kb := newKeyBindings()

	keys := kb.Navigate.Keys()
	if len(keys) != 6 {
		t.Errorf("expected 6 navigate keys, got %d", len(keys))
	}

	expected := map[string]bool{"j": false, "k": false, "up": false, "down": false, "g": false, "G": false}
	for _, k := range keys {
		if _, exists := expected[k]; exists {
			expected[k] = true
		}
	}

	for k, found := range expected {
		if !found {
			t.Errorf("expected navigate to include '%s'", k)
		}
	}
}

// TestKeyBindings_Open verifies open binding
func TestKeyBindings_Open(t *testing.T) {
	kb := newKeyBindings()
	keys := kb.Open.Keys()
	if len(keys) != 1 || keys[0] != "enter" {
		t.Errorf("expected open key 'enter', got %v", keys)
	}
}

// TestKeyBindings_Back verifies back binding
func TestKeyBindings_Back(t *testing.T) {
	kb := newKeyBindings()
	keys := kb.Back.Keys()
	if len(keys) != 1 || keys[0] != "esc" {
		t.Errorf("expected back key 'esc', got %v", keys)
	}
}

// TestKeyBindings_Scroll verifies scroll binding
func TestKeyBindings_Scroll(t *testing.T) {
	kb := newKeyBindings()
	keys := kb.Scroll.Keys()
	if len(keys) != 6 {
		t.Errorf("expected 6 scroll keys, got %d", len(keys))
	}

	expected := map[string]bool{"j": false, "k": false, "d": false, "u": false, "g": false, "G": false}
	for _, k := range keys {
		if _, exists := expected[k]; exists {
			expected[k] = true
		}
	}

	for k, found := range expected {
		if !found {
			t.Errorf("expected scroll to include '%s'", k)
		}
	}
}

// TestKeyBindings_Help verifies help binding
func TestKeyBindings_Help(t *testing.T) {
	kb := newKeyBindings()
	keys := kb.Help.Keys()
	if len(keys) != 1 || keys[0] != "?" {
		t.Errorf("expected help key '?', got %v", keys)
	}
}

// TestKeyBindings_Link verifies link binding
func TestKeyBindings_Link(t *testing.T) {
	kb := newKeyBindings()
	keys := kb.Link.Keys()
	if len(keys) != 9 {
		t.Errorf("expected 9 link keys, got %d", len(keys))
	}

	for i := 1; i <= 9; i++ {
		keyStr := string('0' + byte(i))
		if found := slices.Contains(keys, keyStr); !found {
			t.Errorf("expected link keys to include '%s'", keyStr)
		}
	}
}

// TestKeyBindings_ShortHelp verifies short help returns correct bindings
func TestKeyBindings_ShortHelp(t *testing.T) {
	kb := newKeyBindings()
	bindings := kb.ShortHelp()
	if len(bindings) != 6 {
		t.Errorf("expected 6 short help bindings, got %d", len(bindings))
	}

	expectedKeys := map[string]bool{"/": false, "?": false, "q": false}
	for _, b := range bindings {
		for _, k := range b.Keys() {
			if _, exists := expectedKeys[k]; exists {
				expectedKeys[k] = true
			}
		}
	}

	for k, found := range expectedKeys {
		if !found {
			t.Errorf("expected short help to include key '%s'", k)
		}
	}
}

// TestKeyBindings_FullHelp verifies full help returns correct bindings
func TestKeyBindings_FullHelp(t *testing.T) {
	kb := newKeyBindings()
	bindings := kb.FullHelp()
	if len(bindings) != 2 {
		t.Errorf("expected 2 full help rows, got %d", len(bindings))
	}

	if len(bindings[0]) != 4 {
		t.Errorf("expected 4 bindings in first row, got %d", len(bindings[0]))
	}

	if len(bindings[1]) != 4 {
		t.Errorf("expected 4 bindings in second row, got %d", len(bindings[1]))
	}
}

// TestKeyBindings_Type verifies the keyBindings type can be created and used
func TestKeyBindings_Type(t *testing.T) {
	kb := newKeyBindings()
	if kb.Quit.Keys() == nil {
		t.Error("expected Quit binding to be initialized")
	}
}
