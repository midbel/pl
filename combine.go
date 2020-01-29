package pl

import (
	"errors"
	"fmt"
)

var ErrDone = errors.New("done")

type Source interface {
	Next() ([]string, error)
	Done() bool
	Reset()
}

type single struct {
	values []string
	index  int
}

func Empty() Source {
	return &single{}
}

func Single(vs []string) Source {
	xs := make([]string, len(vs))
	copy(xs, vs)
	return &single{
		values: xs,
		index:  -1,
	}
}

func (s *single) Next() ([]string, error) {
	if s.Done() {
		return nil, ErrDone
	}
	s.index++
	return []string{s.values[s.index]}, nil
}

func (s *single) Done() bool {
	return len(s.values) == 0 || s.index+1 >= len(s.values)
}

func (s *single) Reset() {
	s.index = -1
}

type link struct {
	left  Source
	right Source
	wrap  bool
}

func LinkStrings(ls, rs []string) Source {
	var (
		left  = Single(ls)
		right = Single(rs)
	)
	return LinkSources(left, right)
}

func LinkSources(ls, rs Source) Source {
	return &link{
		left:  ls,
		right: rs,
	}
}

func (l *link) Next() ([]string, error) {
	if l.Done() {
		return nil, ErrDone
	}
	left, err := l.left.Next()
	if err != nil {
		return nil, err
	}
	right, err := l.right.Next()
	if err != nil {
		return nil, err
	}
	return append(left, right...), nil
}

func (l *link) Done() bool {
	return l.left.Done() || l.right.Done()
}

func (l *link) Reset() {
	l.left.Reset()
	l.right.Reset()
}

type combination struct {
	left  Source
	right Source
	next  []string
}

func CombineStrings(ls, rs []string) Source {
	var (
		left  = Single(ls)
		right = Single(rs)
	)
	return CombineSources(left, right)
}

func CombineSources(left, right Source) Source {
	return &combination{
		left:  left,
		right: right,
	}
}

func (c *combination) Next() ([]string, error) {
	if c.Done() {
		return nil, ErrDone
	}
	if c.right.Done() {
		c.right.Reset()
		c.next = c.next[:0]
	}
	if len(c.next) == 0 {
		ls, _ := c.left.Next()
		c.next = append(c.next, ls...)
	}
	left := make([]string, len(c.next))
	copy(left, c.next)

	right, _ := c.right.Next()
	return append(left, right...), nil
}

func (c *combination) Done() bool {
	return c.left.Done() && c.right.Done()
}

func (c *combination) Reset() {
	c.left.Reset()
	c.right.Reset()
}

const (
	combit = ":::"
	linkit = ":::+"
)

const (
	bindLowest int = iota
	bindCombit
	bindLinkit
)

var bindings = map[string]int{
	linkit: bindLinkit,
	combit: bindCombit,
}

type parser struct {
	values []string
	pos    int
	next   int
}

func Parse(args []string) (Source, error) {
	p := parser{values: args}
	return p.Parse()
}

func (p *parser) Parse() (Source, error) {
	return p.parse(bindLowest)
}

func (p *parser) parse(bp int) (Source, error) {
	left := p.parseValues()
	for !p.isDone() && bp < p.peekPower() {
		right, err := p.parseBinding(left)
		if err != nil {
			return nil, err
		}
		left = right
	}
	return left, nil
}

func (p *parser) parseValues() Source {
	var ds []string
	for !p.isDone() {
		if str := p.peek(); isLink(str) || isCombination(str) {
			break
		}
		ds = append(ds, p.nextArg())
	}
	return Single(ds)
}

func (p *parser) parseBinding(left Source) (Source, error) {
	var (
		marker = p.nextArg()
		bp     = p.currPower()
	)
	right, err := p.parse(bp)
	if err != nil {
		return nil, err
	}

	if isLink(marker) {
		left = LinkSources(left, right)
	} else if isCombination(marker) {
		left = CombineSources(left, right)
	} else {
		err = fmt.Errorf("parse error: unexpected marker %s", marker)
	}

	return left, err
}

func (p *parser) peekPower() int {
	return strPower(p.peek())
}

func (p *parser) currPower() int {
	return strPower(p.current())
}

func (p *parser) isDone() bool {
	return p.next >= len(p.values)
}

func (p *parser) nextArg() string {
	if p.next >= len(p.values) {
		return ""
	}
	p.pos, p.next = p.next, p.next+1
	str := p.values[p.pos]
	return str
}

func (p *parser) current() string {
	return p.at(p.pos)
}

func (p *parser) peek() string {
	return p.at(p.next)
}

func (p *parser) at(x int) string {
	if x >= len(p.values) {
		return ""
	}
	return p.values[x]
}

func strPower(str string) int {
	bp, ok := bindings[str]
	if !ok {
		bp = bindLowest
	}
	return bp
}

func IsDelimiter(str string) bool {
	return isLink(str) || isCombination(str)
}

func isLink(str string) bool {
	return str == linkit
}

func isCombination(str string) bool {
	return str == combit
}
