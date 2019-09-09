package pl

import (
	"testing"
)

func TestBuilderDump(t *testing.T) {
	data := []struct {
		Cmd  string
		Want string
		Args []string
	}{
		{
			Cmd:  "cmd -k {1} {2}",
			Want: "cmd -k foo bar",
			Args: []string{"foo", "bar"},
		},
		{
			Cmd:  "cmd -k {-1} {-2}",
			Want: "cmd -k bar foo",
			Args: []string{"foo", "bar"},
		},
		{
			Cmd:  "cmd -k {-1:title} {-2:upper}",
			Want: "cmd -k Bar FOO",
			Args: []string{"foo", "bar"},
		},
		{
			Cmd:  "cmd -k {2} {1}",
			Want: "cmd -k bar foo",
			Args: []string{"foo", "bar"},
		},
		{
			Cmd:  "cmd -k {1:upper} {2:title}",
			Want: "cmd -k FOO Bar",
			Args: []string{"foo", "bar"},
		},
	}
	for i, d := range data {
		b, err := Build(splitCommand(d.Cmd))
		if err != nil {
			t.Errorf("%d) builder failed: %s", i+1, err)
		}
		str, err := b.Dump(d.Args)
		if err != nil {
			t.Errorf("%d) unexpected error (%s): %s", i+1, d.Want, err)
		}
		if str != d.Want {
			t.Errorf("%d) want: %s, got: %s", i+1, d.Want, str)
		}
	}
}

func splitCommand(str string) []string {
	const (
		single byte = '\''
		double      = '"'
		space       = ' '
		equal       = '='
	)

	skipN := func(b byte) int {
		var i int
		if b == space || b == single || b == double {
			i++
		}
		return i
	}

	var (
		ps  []string
		j   int
		sep byte = space
	)
	for i := 0; i < len(str); i++ {
		if str[i] == sep || str[i] == equal {
			if i > j {
				j += skipN(str[j])
				ps, j = append(ps, str[j:i]), i+1
				if sep == single || sep == double {
					sep = space
				}
			}
			continue
		}
		if sep == space && (str[i] == single || str[i] == double) {
			sep, j = str[i], i+1
		}
	}
	if str := str[j:]; len(str) > 0 {
		i := skipN(str[0])
		ps = append(ps, str[i:])
	}
	return ps
}
