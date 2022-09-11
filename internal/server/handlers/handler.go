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

		if !strings.Contains(r.Header.Get(ContentEncoding), GZip) {
			next.ServeHTTP(w, r)
			return
		}

		writer := gzip.NewWriter(w)
		defer func() {
			if err := writer.Close(); err != nil {
				log.Printf("error close GZIP writer: %v\n", err)
			}
		}()

		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: writer}, r)
	})
}

func (h Handler) CompressResponse(w http.ResponseWriter, r *http.Request, data string) {

	switch r.Header.Get(AcceptEncoding) {
	case GZip:
		w.Header().Set(ContentEncoding, GZip)
		if _, err := io.WriteString(w, data); err != nil {
			h.logger.Err.Printf("error compress data to GZIP: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	default:
		if _, err := w.Write([]byte(data)); err != nil {
			h.logger.Err.Printf("error write data in response body: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func BodyReader(r *http.Request) (io.ReadCloser, error) {

	switch r.Header.Get(ContentEncoding) {
	case GZip:
		return gzip.NewReader(r.Body)
	}

	return r.Body, nil
}
