package kubeagent

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
)

func NewResponseWriter(w io.Writer) http.ResponseWriter {
	return &respWriter{header: http.Header{}, w: w}
}

type respWriter struct {
	header     http.Header
	w          io.Writer
	statusCode int
}

func (f *respWriter) StatusCode() int {
	return f.statusCode
}

func (f *respWriter) Header() http.Header {
	return f.header
}

func (f *respWriter) WriteHeader(statusCode int) {
	f.statusCode = statusCode

	text := http.StatusText(statusCode)
	if text == "" {
		text = "status code " + strconv.Itoa(statusCode)
	}

	_, _ = fmt.Fprintf(f.w, "HTTP/1.1 %03d %s\r\n", statusCode, text)
	_ = f.header.WriteSubset(f.w, map[string]bool{})
	_, _ = io.WriteString(f.w, "\r\n")
}

func (f *respWriter) Write(bytes []byte) (int, error) {
	return f.w.Write(bytes)
}
