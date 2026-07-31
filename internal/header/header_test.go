package header

import "testing"

func TestGetNextPageURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		linkHeader string
		want       string
	}{
		{
			name:       "next relation first",
			linkHeader: `<https://api.github.com/items?page=2>; rel="next", <https://api.github.com/items?page=5>; rel="last"`,
			want:       "https://api.github.com/items?page=2",
		},
		{
			name:       "next relation after previous",
			linkHeader: `<https://api.github.com/items?page=1>; rel="prev", <https://api.github.com/items?page=3>; rel="next"`,
			want:       "https://api.github.com/items?page=3",
		},
		{name: "missing next relation", linkHeader: `<https://api.github.com/items?page=1>; rel="prev"`},
		{name: "empty header"},
		{
			name:       "malformed entry",
			linkHeader: `not-a-link, <https://api.github.com/items?page=4>; rel="next"`,
			want:       "https://api.github.com/items?page=4",
		},
		{name: "relation is not exact", linkHeader: `<https://api.github.com/items?page=2>; rel="next last"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := GetNextPageURL(tt.linkHeader); got != tt.want {
				t.Fatalf("GetNextPageURL(%q) = %q, want %q", tt.linkHeader, got, tt.want)
			}
		})
	}
}
