package main

import (
	"strings"
	"testing"
)

func TestCombinationNext(t *testing.T) {
	data := []struct {
		Input string
		Want  [][]string
	}{
		{
			Input: "A B C ::: D E F",
			Want: [][]string{
				{"A", "D"},
				{"A", "E"},
				{"A", "F"},
				{"B", "D"},
				{"B", "E"},
				{"B", "F"},
				{"C", "D"},
				{"C", "E"},
				{"C", "F"},
			},
		},
		{
			Input: "A B C :::+ D E F",
			Want: [][]string{
				{"A", "D"},
				{"B", "E"},
				{"C", "F"},
			},
		},
		{
			Input: "a b c :::+ 1 2 3 ::: X Y :::+ 11 22",
			Want: [][]string{
				{"a", "1", "X", "11"},
				{"a", "1", "Y", "22"},
				{"b", "2", "X", "11"},
				{"b", "2", "Y", "22"},
				{"c", "3", "X", "11"},
				{"c", "3", "Y", "22"},
			},
		},
		{
			Input: "A B C :::+ G H I ::: D E F",
			Want: [][]string{
				{"A", "G", "D"},
				{"A", "G", "E"},
				{"A", "G", "F"},
				{"B", "H", "D"},
				{"B", "H", "E"},
				{"B", "H", "F"},
				{"C", "I", "D"},
				{"C", "I", "E"},
				{"C", "I", "F"},
			},
		},
		{
			Input: "A B C ::: G H I :::+ D E F",
			Want: [][]string{
				{"A", "G", "D"},
				{"A", "H", "E"},
				{"A", "I", "F"},
				{"B", "G", "D"},
				{"B", "H", "E"},
				{"B", "I", "F"},
				{"C", "G", "D"},
				{"C", "H", "E"},
				{"C", "I", "F"},
			},
		},
	}
	for i, d := range data {
		args := strings.Split(d.Input, " ")
		c := Combine(args)
		var j int
		for vs := c.Next(); vs != nil; vs = c.Next() {
			if j >= len(d.Want) {
				t.Errorf("combination exeeded")
				break
			}
			want := strings.Join(d.Want[j], "+")
			got := strings.Join(vs, "+")
			if got != want {
				t.Errorf("%d) combination %d failed: want %s, got %s", i+1, j+1, want, got)
			}
			j++
		}
	}
}
