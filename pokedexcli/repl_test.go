package main

import "testing"

func TestUnit_cleanInput(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{{
		input:    " ",
		expected: []string{},
	}, {
		input:    "hello",
		expected: []string{"hello"},
	}, {
		input:    "  hello  world  ",
		expected: []string{"hello", "world"},
	}}

	for _, c := range cases {
		actual := cleanInput(c.input)

		if len(actual) != len(c.expected) {
			t.Fatalf("expected length is %d but got %d, output: %q", len(c.expected), len(actual), actual)
		}

		for i := range actual {
			word := actual[i]
			expectedWord := c.expected[i]

			if word != expectedWord {
				t.Errorf("expected word is %q but got %q", word, expectedWord)
			}
		}
	}
}
