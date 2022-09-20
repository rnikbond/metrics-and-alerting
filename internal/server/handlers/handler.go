package handler

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"

	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/logpack"
)

const (
	idxType  = 0
	idxName  = 1
	idxValue = 2

	partsGetURL    = 2
	partsUpdateURL = 3
)

const (
	ContentType     = "Content-Type"
	ContentEncoding = "Content-Encoding"
	AcceptEncoding  = "Accept-Encoding"

	TextPlain       = "text/plain"
	TextHTML        = "text/html"
	ApplicationJSON = "application/json"
	GZip            = "gzip"
)

type (
	Handler struct {
		store  storage.Repository
		logger *logpack.LogPack
	}

	gzipWriter struct {
		http.ResponseWriter
		Writer io.Writer
	}
)

func New(store storage.Repository, logger *logpack.LogPack) *Handler {
	return &Handler{
		store:  store,
		logger: logger,
	}
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (h Handler) DecompressRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !strings.Contains(r.Header.Get(AcceptEncoding), GZip) {
			next.ServeHTTP(w, r)
			return
		}

		writer := gzip.NewWriter(w)
		defer func() {
			if err := writer.Close(); err != nil {
				log.Printf("error close gzip writer: %v\n", err)
			}
		}()

		w.Header().Set(ContentEncoding, GZip)
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: writer}, r)
	})
}

func BodyReader(r *http.Request) (io.ReadCloser, error) {

	switch r.Header.Get(ContentEncoding) {
	case GZip:
		return gzip.NewReader(r.Body)
	}

	return r.Body, nil
}
