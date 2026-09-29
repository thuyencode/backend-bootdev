package filter

import "testing"

func TestUnit_Censor(t *testing.T) {
	cases := []struct{ input, expected string }{
		{
			input:    "This is a kerfuffle opinion I need to share with the world",
			expected: "This is a **** opinion I need to share with the world",
		},
		{
			input:    "I hear Mastodon is better than Chirpy. sharbert I need to migrate",
			expected: "I hear Mastodon is better than Chirpy. **** I need to migrate",
		},
		{
			input:    "I really need a kerfuffle to go to bed sooner, Fornax !",
			expected: "I really need a **** to go to bed sooner, **** !",
		},
	}

	for _, c := range cases {
		actual := Censor(c.input)

		if actual != c.expected {
			t.Errorf("want %q, have %q", c.expected, actual)
		}
	}
}
