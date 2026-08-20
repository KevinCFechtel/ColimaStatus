package localization

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/active.*.json
var localeFiles embed.FS

var supportedLanguages = []language.Tag{
	language.English,
	language.German,
}

// Strings provides all user-facing ColimaStatus messages in one language.
type Strings struct {
	localizer *i18n.Localizer
	language  language.Tag
}

// New loads the embedded catalogs and selects the best supported language.
// English is the deterministic fallback.
func New(preferences ...string) (*Strings, error) {
	bundle := i18n.NewBundle(language.English)

	paths, err := fs.Glob(localeFiles, "locales/active.*.json")
	if err != nil {
		return nil, fmt.Errorf("find embedded localization catalogs: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no embedded localization catalogs found")
	}
	sort.Strings(paths)
	for _, path := range paths {
		if _, err := bundle.LoadMessageFileFS(localeFiles, path); err != nil {
			return nil, fmt.Errorf("load embedded localization catalog %q: %w", path, err)
		}
	}

	normalized := normalizePreferences(preferences)
	if len(normalized) == 0 {
		normalized = []string{language.English.String()}
	}

	matcher := language.NewMatcher(supportedLanguages)
	matched, _ := language.MatchStrings(matcher, normalized...)
	return &Strings{
		localizer: i18n.NewLocalizer(bundle, normalized...),
		language:  matched,
	}, nil
}

func MustNew(preferences ...string) *Strings {
	strings, err := New(preferences...)
	if err != nil {
		panic(err)
	}
	return strings
}

func NewDetected() (*Strings, error) {
	return New(DetectedLanguages()...)
}

func (strings *Strings) Language() language.Tag {
	return strings.language
}

func (strings *Strings) localize(message *i18n.Message, data any) string {
	value, err := strings.localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: message,
		TemplateData:   data,
	})
	if err != nil {
		panic(fmt.Sprintf("localize %q: %v", message.ID, err))
	}
	return value
}
