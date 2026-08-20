package localization

import (
	"io/fs"
	"reflect"
	"testing"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func TestLanguageSelectionAndFallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		preferences []string
		want        language.Base
	}{
		{name: "German", preferences: []string{"de-DE"}, want: language.MustParseBase("de")},
		{name: "regional German", preferences: []string{"de-AT"}, want: language.MustParseBase("de")},
		{name: "English", preferences: []string{"en-GB"}, want: language.MustParseBase("en")},
		{name: "unknown language", preferences: []string{"fr-FR"}, want: language.MustParseBase("en")},
		{name: "empty preferences", want: language.MustParseBase("en")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			texts := MustNew(test.preferences...)
			base, _ := texts.Language().Base()
			if base != test.want {
				t.Fatalf("selected language = %s, want %s", base, test.want)
			}
		})
	}
}

func TestLocalizedMessages(t *testing.T) {
	t.Parallel()

	if got := MustNew("en").ProfileRunning("work"); got != "Colima is running (work)" {
		t.Fatalf("English profile status = %q", got)
	}
	if got := MustNew("de").ProfileRunning("work"); got != "Colima läuft (work)" {
		t.Fatalf("German profile status = %q", got)
	}
}

func TestLocalizedTimeFormatting(t *testing.T) {
	t.Parallel()

	checkedAt := time.Date(2026, time.August, 15, 14, 30, 5, 0, time.UTC)
	if got := MustNew("de").LastChecked(checkedAt); got != "Zuletzt geprüft: 14:30:05" {
		t.Fatalf("German time = %q", got)
	}
	if got := MustNew("en").LastChecked(checkedAt); got != "Last checked: 2:30:05 PM" {
		t.Fatalf("English time = %q", got)
	}
}

func TestNormalizePreferences(t *testing.T) {
	t.Parallel()

	got := normalizePreferences([]string{" de_DE.UTF-8 ", "de-DE", "C", "en_US@calendar"})
	want := []string{"de-DE", "en-US"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizePreferences() = %#v, want %#v", got, want)
	}
}

func TestExplicitLanguageOverride(t *testing.T) {
	t.Setenv(languageOverrideEnvironment, "de_DE.UTF-8")
	got := DetectedLanguages()
	if !reflect.DeepEqual(got, []string{"de-DE"}) {
		t.Fatalf("DetectedLanguages() = %#v, want German override", got)
	}
}

func TestEmbeddedCatalogsAreComplete(t *testing.T) {
	t.Parallel()

	english := catalogMessages(t, "locales/active.en.json")
	german := catalogMessages(t, "locales/active.de.json")
	if len(english) == 0 {
		t.Fatal("English catalog contains no messages")
	}
	if len(german) != len(english) {
		t.Fatalf("German catalog contains %d messages, want %d", len(german), len(english))
	}
	for id, source := range english {
		translation, exists := german[id]
		if !exists {
			t.Errorf("German catalog is missing %q", id)
			continue
		}
		if source.Other != "" && translation.Other == "" {
			t.Errorf("German catalog has no other form for %q", id)
		}
	}
}

func catalogMessages(t *testing.T, path string) map[string]*i18n.Message {
	t.Helper()
	content, err := fs.ReadFile(localeFiles, path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	messageFile, err := i18n.ParseMessageFileBytes(content, path, nil)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	messages := make(map[string]*i18n.Message, len(messageFile.Messages))
	for _, message := range messageFile.Messages {
		messages[message.ID] = message
	}
	return messages
}
