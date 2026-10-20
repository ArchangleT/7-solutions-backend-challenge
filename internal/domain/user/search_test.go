package user

import (
	"reflect"
	"testing"
)

func TestParseSearchQuery(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		want      SearchQuery
		wantEmpty bool
	}{
		{name: "empty", raw: "   ", wantEmpty: true},
		{
			name: "splits on non alphanumerics and lowercases",
			raw:  "Mary-Jane O'Neil",
			want: SearchQuery{Tokens: []string{"mary", "jane", "o", "neil"}},
		},
		{
			name: "quoted tokens become exact",
			raw:  `ali "bob" carol`,
			want: SearchQuery{Tokens: []string{"ali", "carol"}, ExactTokens: []string{"bob"}},
		},
		{
			name: "unterminated quote still flushes exact token",
			raw:  `"bob`,
			want: SearchQuery{ExactTokens: []string{"bob"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSearchQuery(tt.raw)
			if tt.wantEmpty {
				if !got.Empty() {
					t.Fatalf("ParseSearchQuery(%q) = %+v, want empty", tt.raw, got)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseSearchQuery(%q) = %+v, want %+v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSearchQueryScore(t *testing.T) {
	alice := &User{Name: "Alice Johnson", Email: "alice@example.com"}

	tests := []struct {
		name      string
		query     SearchQuery
		wantMatch bool
		wantScore int64
	}{
		{
			name:      "name prefix outranks email",
			query:     SearchQuery{Tokens: []string{"ali"}},
			wantMatch: true,
			wantScore: WeightName + WeightEmail, // matches "Alice" and "alice@"
		},
		{
			name:      "email-only prefix match",
			query:     SearchQuery{Tokens: []string{"example"}},
			wantMatch: true,
			wantScore: WeightEmail,
		},
		{
			name:      "exact token adds the exact bonus",
			query:     SearchQuery{ExactTokens: []string{"alice"}},
			wantMatch: true,
			wantScore: WeightName + WeightExact + WeightEmail + WeightExact,
		},
		{
			name:      "no match",
			query:     SearchQuery{Tokens: []string{"zzz"}},
			wantMatch: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, matched := tt.query.Score(alice)
			if matched != tt.wantMatch {
				t.Fatalf("matched = %v, want %v", matched, tt.wantMatch)
			}
			if matched && score != tt.wantScore {
				t.Errorf("score = %d, want %d", score, tt.wantScore)
			}
		})
	}
}

func TestSplitWords(t *testing.T) {
	got := SplitWords("Mary-Jane.O'Neil 42")
	want := []string{"mary", "jane", "o", "neil", "42"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SplitWords() = %v, want %v", got, want)
	}
}
