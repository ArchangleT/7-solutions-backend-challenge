package user

import (
	"strings"
	"unicode"
)

// Relevance weights for the "Top Hits" ranking. A match on the name (the
// analogue of a note title) outranks a match on the email, and a whole-word
// match outranks a mere word-prefix match.
const (
	WeightName  = int64(10)
	WeightEmail = int64(5)
	WeightExact = int64(4)
)

// SearchQuery describes a search request using word-prefix semantics: a user
// matches when a word in the name or email starts with one of Tokens.
// ExactTokens must instead match a whole word. Matching is always
// case-insensitive, mirroring how Apple Notes matches its notes.
type SearchQuery struct {
	Tokens      []string
	ExactTokens []string
}

// Empty reports whether the query carries no tokens to match.
func (q SearchQuery) Empty() bool {
	return len(q.Tokens) == 0 && len(q.ExactTokens) == 0
}

// ParseSearchQuery turns raw search input into a SearchQuery. A token wrapped
// in double quotes (e.g. `"alice"`) becomes an exact-match token; everything
// else is a word-prefix token. Input is lower-cased and split on anything that
// is not a letter or digit, so "mary-jane" yields "mary" and "jane".
func ParseSearchQuery(raw string) SearchQuery {
	var q SearchQuery
	var current strings.Builder
	inQuotes := false

	flush := func() {
		word := current.String()
		current.Reset()
		if word == "" {
			return
		}
		if inQuotes {
			q.ExactTokens = append(q.ExactTokens, word)
		} else {
			q.Tokens = append(q.Tokens, word)
		}
	}

	for _, r := range strings.ToLower(raw) {
		switch {
		case r == '"':
			flush()
			inQuotes = !inQuotes
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			current.WriteRune(r)
		default:
			flush()
		}
	}
	flush()

	return q
}

// Score returns the relevance of a user for this query and whether the user
// matched at all. Prefix and exact tokens are weighted so that name matches
// and whole-word matches rise to the top (see the Weight* constants).
func (q SearchQuery) Score(u *User) (int64, bool) {
	nameWords := SplitWords(u.Name)
	emailWords := SplitWords(u.Email)

	var score int64
	matched := false
	add := func(weight int64) {
		score += weight
		matched = true
	}

	for _, t := range q.Tokens {
		if HasWordPrefix(nameWords, t) {
			add(WeightName)
		}
		if HasWordPrefix(emailWords, t) {
			add(WeightEmail)
		}
	}
	for _, t := range q.ExactTokens {
		if HasWord(nameWords, t) {
			add(WeightName + WeightExact)
		}
		if HasWord(emailWords, t) {
			add(WeightEmail + WeightExact)
		}
	}

	return score, matched
}

// SplitWords lower-cases text and splits it on any non letter/digit boundary.
func SplitWords(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// HasWordPrefix reports whether any word starts with the given token.
func HasWordPrefix(words []string, token string) bool {
	for _, w := range words {
		if strings.HasPrefix(w, token) {
			return true
		}
	}
	return false
}

// HasWord reports whether the token equals a whole word.
func HasWord(words []string, token string) bool {
	for _, w := range words {
		if w == token {
			return true
		}
	}
	return false
}
