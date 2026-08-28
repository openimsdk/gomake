package util

import (
	"io"

	"github.com/openimsdk/tools/utils/datautil"
)

func MultiWriter(writers ...io.Writer) io.Writer {
	return io.MultiWriter(datautil.Filter(writers, func(w io.Writer) (io.Writer, bool) { return w, w != nil })...)
}

type WriterFunc func(p []byte) (n int, err error)

func (f WriterFunc) Write(p []byte) (n int, err error) { return f(p) }
