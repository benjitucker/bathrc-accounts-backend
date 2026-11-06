package email

import (
	"reflect"
	"strings"
	"unicode"
)

// VenueDisplayName turns a venue value such as "StockleyFarm" into "Stockley Farm".
func VenueDisplayName(venue string) string {
	var b strings.Builder
	runes := []rune(venue)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && (unicode.IsLower(runes[i-1]) ||
			(i+1 < len(runes) && unicode.IsLower(runes[i+1]) && unicode.IsUpper(runes[i-1]))) {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// DisplayName tidies a name that is all in capitals or all in lower case, such as "OLIVIA" or
// "mary-jane", into "Olivia" or "Mary-Jane". Names already in mixed case, such as "McDonald",
// are left alone.
func DisplayName(name string) string {
	hasUpper, hasLower := false, false
	for _, r := range name {
		hasUpper = hasUpper || unicode.IsUpper(r)
		hasLower = hasLower || unicode.IsLower(r)
	}
	if hasUpper && hasLower {
		return name
	}
	runes := []rune(strings.ToLower(name))
	for i, r := range runes {
		if i == 0 || runes[i-1] == ' ' || runes[i-1] == '-' || runes[i-1] == '\'' || runes[i-1] == '’' {
			runes[i] = unicode.ToUpper(r)
		}
	}
	return string(runes)
}

// tidyForDisplay tidies the names and venues in an email's data before it is rendered, so every
// email shows "Olivia" rather than "OLIVIA" and "West Wilts" rather than "WestWilts".
func tidyForDisplay(data any) {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return
	}
	s := v.Elem()
	set := func(field string, tidy func(string) string) {
		f := s.FieldByName(field)
		if f.IsValid() && f.Kind() == reflect.String && f.CanSet() {
			f.SetString(tidy(f.String()))
		}
	}
	set("FirstName", func(names string) string {
		// "BENJAMIN and OLIVIA": tidy each name, keeping the "and"
		parts := strings.Split(names, " and ")
		for i := range parts {
			parts[i] = DisplayName(parts[i])
		}
		return strings.Join(parts, " and ")
	})
	set("Venue", VenueDisplayName)
	set("Venue2", VenueDisplayName)
}
