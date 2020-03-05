package pl

import (
	"fmt"
	"os"
	"testing"
)

func TestSplit(t *testing.T) {
	os.Setenv("TEST", "SPLIT")
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
		{
			Input: `""    ''`,
			Words: []string{"", ""},
		},
		{
			Input: `$FOO ${FOO}`,
			Words: []string{"", ""},
		},
		{
			Input: `$TEST "${TEST}"`,
			Words: []string{"SPLIT", "SPLIT"},
		},
	}
	for i, d := range data {
		ws, err := Split(d.Input)
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
		return fmt.Errorf("length mismatched! want %d, got %d (%q)", len(want), len(got), got)
	}
	for i := 0; i < len(want); i++ {
		if want[i] != got[i] {
			return fmt.Errorf("words mismatched at %d! want %s, got %s", i, want[i], got[i])
		}
	}
	return nil
}
