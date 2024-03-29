package reporter

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"google.golang.org/grpc"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/logpack"
	"metrics-and-alerting/pkg/metric"
	"net/http"

	"github.com/go-resty/resty/v2"

	pb "metrics-and-alerting/proto"
)

const (
	ReportAsURL       = "URL"
	ReportAsJSON      = "JSON"
	ReportAsBatchJSON = "BatchJSON"
	ReportAsGRPC      = "GRPC"
)

type (
	OptionReporter func(*Reporter)

	Reporter struct {
		addr      string
		signKey   []byte
		storage   storage.Repository
		rpcClient pb.MetricsClient
		logger    *logpack.LogPack
		publicKey *rsa.PublicKey
	}
)

func NewReporter(addr string, storage storage.Repository, logger *logpack.LogPack, opts ...OptionReporter) *Reporter {

	r := &Reporter{
		addr:    addr,
		storage: storage,
		logger:  logger,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func WithSignKey(key []byte) OptionReporter {
	return func(reporter *Reporter) {
		reporter.signKey = key
	}
}

func WithKey(key []byte) OptionReporter {
	return func(reporter *Reporter) {

		if len(key) == 0 {
			return
		}

		block, _ := pem.Decode(key)
		if block == nil {
			//reporter.logger.Err.Printf("key %s has invalid format!\n", key)
			return
		}

		publicKey, errKey := x509.ParsePKIXPublicKey(block.Bytes)
		if errKey != nil {
			reporter.logger.Err.Printf("could not create rsa.PublicKey: %v\n", errKey)
			return
		}

		switch pub := publicKey.(type) {
		case *rsa.PublicKey:
			reporter.publicKey = pub
		default:
			reporter.logger.Err.Println("failed create rsa.PublicKey: key is not RSA!")
		}
	}
}

func WithRPC(conn *grpc.ClientConn) OptionReporter {
	return func(reporter *Reporter) {
		if conn != nil {
			reporter.rpcClient = pb.NewMetricsClient(conn)
		}
	}
}

func (r Reporter) Encrypt(data []byte) ([]byte, error) {
	if r.publicKey == nil {
		return data, nil
	}

	hashFunc := sha256.New()
	dataLen := len(data)
	step := r.publicKey.Size() - hashFunc.Size()*2 - 2
	var encryptedBytes []byte
	for start := 0; start < dataLen; start += step {
		finish := start + step
		if finish > dataLen {
			finish = dataLen
		}

		encryptedBlockBytes, err := rsa.EncryptOAEP(
			hashFunc,
			rand.Reader,
			r.publicKey,
			data[start:finish],
			nil)

		if err != nil {
			return nil, err
		}

		encryptedBytes = append(encryptedBytes, encryptedBlockBytes...)
	}

	return encryptedBytes, nil
}

func (r Reporter) Report(ctx context.Context, reportType string) error {

	switch reportType {
	case ReportAsURL:
		if err := r.reportURL(ctx); err != nil {
			return err
		}

	case ReportAsJSON:
		if err := r.reportJSON(ctx); err != nil {
			return err
		}

	case ReportAsBatchJSON:
		if err := r.reportBatchJSON(ctx); err != nil {
			return err
		}
	case ReportAsGRPC:
		if err := r.reportGRPC(ctx); err != nil {
			return err
		}

	default:
		return fmt.Errorf("could not report metrics: unknown report type")
	}

	return nil
}

// reportGRPC Отправка метрик GRPC шлюз
func (r Reporter) reportGRPC(ctx context.Context) error {

	metrics, errStorage := r.storage.GetBatch()
	if errStorage != nil {
		return errStorage
	}
	for _, m := range metrics {

		sign, errSign := m.Sign(r.signKey)
		if errSign != nil {
			return fmt.Errorf("could not report metrics: %v", errSign)
		}

		m.Hash = sign

		var errResp error

		switch m.MType {
		case metric.CounterType:
			_, errResp = r.rpcClient.UpsertCounter(ctx, &pb.UpsertCounterRequest{
				Id:    m.ID,
				Delta: *m.Delta,
				Hash:  m.Hash,
			})
		case metric.GaugeType:
			_, errResp = r.rpcClient.UpsertGauge(ctx, &pb.UpsertGaugeRequest{
				Id:    m.ID,
				Value: *m.Value,
				Hash:  m.Hash,
			})
		}

		if errResp != nil {
			return fmt.Errorf("failed upsert metric: %s", errResp)
		}
	}

	return nil
}

// reportURL Отправка метрик через URL отдельными запросами
func (r Reporter) reportURL(ctx context.Context) error {

	metrics, errStorage := r.storage.GetBatch()
	if errStorage != nil {
		return fmt.Errorf("could not report metrics: %v", errStorage)
	}

	client := resty.New()

	for _, m := range metrics {

		resp, err := client.R().
			SetHeader("Content-Type", "text/plain").
			SetPathParams(m.Map()).
			SetContext(ctx).
			Post(r.addr + "/update/" + "{type}/{name}/{value}")

		if err != nil {
			return fmt.Errorf("could not send metrics as URL: %w", err)
		}

		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("server return no success status on update metrics as URL: %d", resp.StatusCode())
		}
	}

	return nil
}

// reportJSON Отправка метрик в виде JSON отдельными запросами
func (r Reporter) reportJSON(ctx context.Context) error {

	metrics, errStorage := r.storage.GetBatch()
	if errStorage != nil {
		return fmt.Errorf("could not report metrics: %v", errStorage)
	}

	client := resty.New()

	for _, m := range metrics {

		sign, errSign := m.Sign(r.signKey)
		if errSign != nil {
			return fmt.Errorf("could not report metrics: %v", errSign)
		}

		m.Hash = sign

		data, err := json.Marshal(&m)
		if err != nil {
			return fmt.Errorf("error encode metric to JSON: %w", err)
		}

		data, err = r.Encrypt(data)
		if err != nil {
			return fmt.Errorf("error encrypt metric marshaled data: %w", err)
		}

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(data).
			SetContext(ctx).
			Post(r.addr + "/update")

		if err != nil {
			return fmt.Errorf("could not send metrics as JSON: %w", err)
		}

		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("server return no success status on update metrics as JSON: %d", resp.StatusCode())
		}
	}

	return nil
}

// reportBatchJSON Отправка метрик в виде JSON одним запросом
func (r Reporter) reportBatchJSON(ctx context.Context) error {

	metrics, errStorage := r.storage.GetBatch()
	if errStorage != nil {
		return fmt.Errorf("could not report metrics: %v", errStorage)
	}

	// TODO :: Разобраться, как изменять текущий слайс, а не записывать в новый
	metricsSigned := make([]metric.Metric, len(metrics))

	for i, m := range metrics {

		sign, errSign := m.Sign(r.signKey)
		if errSign != nil {
			return fmt.Errorf("could not report metrics: %v", errSign)
		}

		m.Hash = sign
		metricsSigned[i] = m

	}

	data, err := json.Marshal(&metricsSigned)
	if err != nil {
		return fmt.Errorf("error encode metrics to JSON: %w", err)
	}

	data, err = r.Encrypt(data)
	if err != nil {
		return fmt.Errorf("error encrypt metric marshaled data: %w", err)
	}

	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("X-Real-IP", "125.3.21.1").
		SetBody(data).
		SetContext(ctx).
		Post(r.addr + "/updates")

	if err != nil {
		return fmt.Errorf("could not send metrics as Batch-JSON: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server return no success status on update metrics as Batch-JSON: %d", resp.StatusCode())
	}

	return nil
}
