package pl

import (
	"io"
	"sync"
)

const (
	prefixOut = '<'
	prefixErr = '>'
)

type writer struct {
	tag    string
	prefix byte

	mu    sync.Mutex
	inner io.Writer
}

func stdout(w io.Writer) io.Writer {
	return &writer{
		prefix: prefixOut,
		inner:  w,
	}
}

func stderr(w io.Writer) io.Writer {
	return &writer{
		prefix: prefixErr,
		inner:  w,
	}
}

func (w *writer) Write(xs []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.tag != "" {
		_, err := io.WriteString(w.inner, w.tag)
		if err != nil {
			return 0, err
		}
	}

	bs := make([]byte, 0, len(xs)+1)
	bs = append(bs, w.prefix)
	bs = append(bs, xs...)

	_, err := w.inner.Write(bs)
	return len(xs), err
}
