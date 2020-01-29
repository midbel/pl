package pl

import (
	"fmt"
	"testing"
)

func TestWords(t *testing.T) {
	data := []struct {
		Input string
		Words []string
	}{
		{
			Input: `one`,
			Words: []string{"one"},
		},
		{
			Input: `one two three`,
			Words: []string{"one", "two", "three"}},
		{
			Input: `one "two two" three`,
			Words: []string{"one", "two two", "three"}},
		{
			Input: `"one"`,
			Words: []string{"one"},
		},
		{
			Input: `one" string with "space`,
			Words: []string{"one string with space"},
		},
		{
			Input: `one" string with space" "another string"`,
			Words: []string{"one string with space", "another string"},
		},
		{
			Input: `one "\"two\"" three`,
			Words: []string{"one", "\\\"two\\\"", "three"},
		},
	}
	for i, d := range data {
		ws, err := Words(d.Input)
		if err != nil {
			t.Errorf("%d) fail %s: %s", i+1, d.Input, err)
			continue
		}
		if err := cmpWords(d.Words, ws); err != nil {
			t.Errorf("%d) fail %s: %s", i+1, d.Input, err)
		}
	}
}

func cmpWords(want, got []string) error {
	if len(got) != len(want) {
		return fmt.Errorf("number of words mismatched! want %d, got %d (%q)", len(want), len(got), got)
	}
	for i := 0; i < len(want); i++ {
		if want[i] != got[i] {
			return fmt.Errorf("word mismatched at %d! want %s, got %s", i, want[i], got[i])
		}
	}
	return nil
}
