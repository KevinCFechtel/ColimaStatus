package localization

import (
	"os"
	"strings"

	"golang.org/x/text/language"
)

const languageOverrideEnvironment = "COLIMASTATUS_LANGUAGE"

// DetectedLanguages returns normalized language preferences in priority order.
func DetectedLanguages() []string {
	if override := os.Getenv(languageOverrideEnvironment); override != "" {
		if preferences := normalizePreferences(splitLanguageList(override)); len(preferences) > 0 {
			return preferences
		}
	}

	preferences := normalizePreferences(platformPreferredLanguages())
	if len(preferences) == 0 {
		preferences = normalizePreferences(environmentPreferredLanguages(os.Getenv))
	}
	if len(preferences) == 0 {
		return []string{language.English.String()}
	}
	return preferences
}

func environmentPreferredLanguages(getenv func(string) string) []string {
	if languages := getenv("LANGUAGE"); languages != "" {
		return splitLanguageList(languages)
	}
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := getenv(name); value != "" {
			return []string{value}
		}
	}
	return nil
}

func splitLanguageList(value string) []string {
	return strings.FieldsFunc(value, func(character rune) bool {
		switch character {
		case ':', ',', ';':
			return true
		default:
			return false
		}
	})
}

func normalizePreferences(preferences []string) []string {
	normalized := make([]string, 0, len(preferences))
	seen := make(map[string]struct{}, len(preferences))
	for _, preference := range preferences {
		value := normalizePreference(preference)
		if value == "" {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func normalizePreference(preference string) string {
	value := strings.TrimSpace(preference)
	if value == "" || strings.EqualFold(value, "C") || strings.EqualFold(value, "POSIX") {
		return ""
	}
	if separator := strings.IndexAny(value, ".@"); separator >= 0 {
		value = value[:separator]
	}
	value = strings.ReplaceAll(value, "_", "-")
	tag, err := language.Parse(value)
	if err != nil {
		return ""
	}
	return tag.String()
}
