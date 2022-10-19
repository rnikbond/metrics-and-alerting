package handler

import (
	"compress/gzip"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
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
	XRealIP         = "X-Real-IP"
	ContentType     = "Content-Type"
	ContentEncoding = "Content-Encoding"
	AcceptEncoding  = "Accept-Encoding"

	TextPlain       = "text/plain"
	TextHTML        = "text/html"
	ApplicationJSON = "application/json"
	GZip            = "gzip"
)

type (
	OptionsHandler func(*Handler)

	Handler struct {
		store         storage.Repository
		logger        *logpack.LogPack
		privateKey    *rsa.PrivateKey
		trustedSubnet []string
	}

	gzipWriter struct {
		http.ResponseWriter
		Writer io.Writer
	}
)

func New(store storage.Repository, logger *logpack.LogPack, opts ...OptionsHandler) *Handler {
	h := &Handler{
		store:  store,
		logger: logger,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

func WithKey(key string) OptionsHandler {
	return func(h *Handler) {

		if len(key) == 0 {
			return
		}

		block, _ := pem.Decode([]byte(key))
		if block == nil {
			h.logger.Err.Println("failed decode private key!")
			return
		}

		privateKey, errParse := x509.ParsePKCS1PrivateKey(block.Bytes)
		if errParse != nil {
			h.logger.Err.Printf("failed parse private key: %v\n", privateKey)
			return
		}

		h.privateKey = privateKey
	}
}

func WithTrustedSubnet(subnet string) OptionsHandler {
	return func(h *Handler) {

		if len(subnet) == 0 {
			return
		}

		subnet = strings.TrimSpace(subnet)
		h.trustedSubnet = strings.Split(subnet, ",")
	}
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Trust Middleware Проверяет, находится ли IP адрес клиента в списке IP адресов, от которых принимаются запросы.
// Если такого скиска нет, то запросы обрабатываются от любого IP адреса.
func (h Handler) Trust(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(h.trustedSubnet) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := r.Header.Get(XRealIP)

		for _, ip := range h.trustedSubnet {
			if ip == clientIP {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.WriteHeader(http.StatusForbidden)
	})
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

func (h Handler) Decrypt(r io.ReadCloser) ([]byte, error) {

	data, errRead := io.ReadAll(r)
	defer func() {
		if err := r.Close(); err != nil {
			h.logger.Err.Printf("could not close body http.Request: %v\n", err)
		}
	}()

	if h.privateKey == nil {
		return data, errRead
	}

	dataLen := len(data)
	step := h.privateKey.PublicKey.Size()
	var decryptedBytes []byte

	for start := 0; start < dataLen; start += step {
		finish := start + step
		if finish > dataLen {
			finish = dataLen
		}

		decryptedBlockBytes, err := h.privateKey.Decrypt(nil, data[start:finish], &rsa.OAEPOptions{Hash: crypto.SHA256})
		if err != nil {
			return nil, err
		}

		decryptedBytes = append(decryptedBytes, decryptedBlockBytes...)
	}

	return decryptedBytes, nil
}

func BodyReader(r *http.Request) (io.ReadCloser, error) {

	switch r.Header.Get(ContentEncoding) {
	case GZip:
		return gzip.NewReader(r.Body)
	}

	return r.Body, nil
}
