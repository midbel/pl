package main

import (
	"bufio"
	"math/rand"
	"os"
)

const (
	combArg = ":::"
	linkArg = ":::+"
)

type Source interface {
	Next() []string
}

type stdin struct {
	scan  *bufio.Scanner
	empty bool
}

func Stdin(empty bool) Source {
	s := bufio.NewScanner(os.Stdin)
	return &stdin{scan: s, empty: empty}
}

func (s *stdin) Next() []string {
	if err := s.scan.Err(); err != nil || !s.scan.Scan() {
		return nil
	}
	var vs []string

	str := s.scan.Text()
	if !s.empty && len(str) == 0 {
		return s.Next()
	}
	return append(vs, str)
}

type Combination struct {
	data  [][]string
	combi []int
	size  int
}

func Combine(as []string) Source {
	return combineAndShuffle(as, false)
}

func Shuffle(as []string) Source {
	return combineAndShuffle(as, true)
}

func (c *Combination) Next() []string {
	if c.isDone() {
		return nil
	}
	c.next(c.size - 1)
	vs := make([]string, c.size)

	for i := 0; i < c.size; i++ {
		vs[i] = c.data[i][c.combi[i]]
	}
	return vs
}

func (c *Combination) next(i int) {
	if i < 0 {
		return
	}

	var reset bool
	if c.combi[i] == len(c.data[i])-1 {
		c.combi[i] = 0
	}
	if j := i - 1; (j >= 0 && !reset && c.combi[j] == 0) || c.combi[i] == 0 {
		reset = !reset
	}
	step := 1

	c.combi[i]++
	if j := i - 1; j >= 0 && isLink(c.data[i]) {
		if z := len(c.data[j]); len(c.data[i]) > z && c.combi[i] > z-1 {
			c.combi[i], step, reset = len(c.data[i])-1, 0, true
		} else {
			c.combi[j] = c.combi[i]
			step++
		}
	}
	if reset {
		c.next(i - step)
	}
}

func (c *Combination) isDone() bool {
	for i := c.size - 1; i >= 0; i-- {
		var ix, lim int
		if j := i - 1; j >= 0 && isLink(c.data[i]) {
			ix, lim = c.combi[i], len(c.data[i])
			if z := len(c.data[j]); z < lim {
				lim = z
			}
			i--
		} else {
			ix, lim = c.combi[i], len(c.data[i])
		}
		if ix < lim-1 {
			return false
		}
	}
	return true
}

func (c *Combination) Reset() {
	if len(c.combi) == 0 {
		c.size = len(c.data)
		c.combi = make([]int, c.size)
	}
	for i := 0; i < c.size; i++ {
		c.combi[i] = 0
	}
}

func combineAndShuffle(as []string, shuffle bool) *Combination {
	args := joinArgs(as)
	if shuffle {
		for i := range args {
			typ, xs := args[i][0], args[i][1:]
			rand.Shuffle(len(xs), func(i, j int) {
				xs[i], xs[j] = xs[j], xs[i]
			})
			args[i] = append([]string{typ}, xs...)
		}
	}
	c := Combination{data: args}
	c.Reset()
	return &c
}

func joinArgs(args []string) [][]string {
	if len(args) == 0 {
		return nil
	}
	if !(args[0] == combArg || args[0] == linkArg) {
		args = append([]string{combArg}, args...)
	}
	var (
		as [][]string
		j  int
	)
	for i := 1; i < len(args); i++ {
		if args[i] == combArg || args[i] == linkArg {
			as = append(as, args[j:i])
			j = i
		}
	}
	return append(as, args[j:])
}

func isCombination(data []string) bool {
	return data[0] == combArg
}

func isLink(data []string) bool {
	return data[0] == linkArg
}
