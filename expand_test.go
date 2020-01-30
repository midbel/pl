package pl

import (
	"strings"
	"testing"
)

func TestExpander(t *testing.T) {
	args := []string{"foo", "bar", "FOOBAR", "  foobar  ", "/tmp/foobar.lst"}
	data := []struct {
		Input []string
		Want  string
	}{
		{
			Input: []string{"echo", "foo bar"},
			Want:  "echo foo bar",
		},
		{
			Input: []string{"echo", "{1}", "{2:}"},
			Want:  "echo foo bar",
		},
		{
			Input: []string{"echo", "welcom {1:upper}! good luck {4:trim}!"},
			Want:  "echo welcom FOO! good luck foobar!",
		},
		{
			Input: []string{"echo", "{2:}", "{1}"},
			Want:  "echo bar foo",
		},
		{
			Input: []string{"echo", "{1}-{2:}", "{1}"},
			Want:  "echo foo-bar foo",
		},
		{
			Input: []string{"echo", "{5:base}", "{5:dir}", "{5:ext}"},
			Want:  "echo foobar.lst /tmp .lst",
		},
		{
			Input: []string{"echo", "{1:upper}", "{3:lower}", "{1:len}", "{4:trim}"},
			Want:  "echo FOO foobar 3 foobar",
		},
		{
			Input: []string{"echo", "{3#FOO}", "{3%BAR}"},
			Want:  "echo BAR FOO",
		},
	}
	for i, d := range data {
		ex, err := NewExpander(d.Input)
		if err != nil {
			t.Errorf("%d) fail to parse %s: %s", i+1, d.Input, err)
			continue
		}
		as, err := ex.Expand(args)
		if err != nil {
			t.Errorf("%d) fail to expand %s: %s", i+1, d.Input, err)
			continue
		}
		got := strings.Join(as, " ")
		if got != d.Want {
			t.Errorf("%d) want %q, got %q", i+1, d.Want, got)
		}
	}
}
