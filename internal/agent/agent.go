package agent

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"metrics-and-alerting/pkg/errs"
	"strings"
	"time"

	"metrics-and-alerting/internal/agent/services/reporter"
	"metrics-and-alerting/internal/agent/services/scanner"
	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/logpack"
	"metrics-and-alerting/pkg/metric"
)

type OptionsAgent func(*Agent)

type Agent struct {
	reportInterval time.Duration
	pollInterval   time.Duration
	addr           string
	reportType     string
	signKey        []byte
	publicKey      []byte
	storage        storage.Repository
	conn           *grpc.ClientConn
	logger         *logpack.LogPack
}

// NewAgent Создание экземпляра агента.
// Используется паттерн "Функциональные опции"
func NewAgent(storage storage.Repository, opts ...OptionsAgent) *Agent {
	a := &Agent{
		storage: storage,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

func WithReportInterval(interval time.Duration) OptionsAgent {
	return func(agent *Agent) {
		agent.reportInterval = interval
	}
}

func WithPollInterval(interval time.Duration) OptionsAgent {
	return func(agent *Agent) {
		agent.pollInterval = interval
	}
}

func WithAddr(addr string) OptionsAgent {
	return func(agent *Agent) {
		agent.addr = addr
	}
}

func WithLogger(logger *logpack.LogPack) OptionsAgent {
	return func(agent *Agent) {
		agent.logger = logger
	}
}

func WithReportURL(reportURL string) OptionsAgent {
	return func(agent *Agent) {
		agent.reportType = reportURL
	}
}

func WithSignKey(key []byte) OptionsAgent {
	return func(agent *Agent) {
		agent.signKey = key
	}
}

func WithKey(key []byte) OptionsAgent {
	return func(agent *Agent) {
		agent.publicKey = key
	}
}

// Start Запуск агента для сбора и отправки метрик
func (a Agent) Start(ctx context.Context) error {

	if a.storage == nil {
		return fmt.Errorf("could not start agent: not setted storage")
	}

	if len(a.addr) == 0 {
		return fmt.Errorf("could not start agent: not setted report address")
	}

	if len(a.reportType) == 0 {
		return fmt.Errorf("could not start agent: not setted report type")
	}

	if a.reportType == reporter.ReportAsGRPC {
		parts := strings.Split(a.addr, ":")
		if len(parts) == 0 {
			return fmt.Errorf("invalid address grpc gate")
		}

		var errConn error
		a.conn, errConn = grpc.Dial(":"+parts[len(parts)-1], grpc.WithTransportCredentials(insecure.NewCredentials()))
		if errConn != nil {
			return fmt.Errorf("failed create gRPC client connection: %w", errConn)
		}
	}

	go a.updateMetrics(ctx)
	go a.reportMetrics(ctx)

	return nil
}

func (a *Agent) updateMetrics(ctx context.Context) {

	scan := scanner.NewScanner(a.storage)
	ticker := time.NewTicker(a.pollInterval)

	for {
		select {

		case <-ticker.C:
			if err := scan.Scan(); err != nil {
				a.logger.Err.Printf("scan task failed with error: %v\n", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) reportMetrics(ctx context.Context) {

	report := reporter.NewReporter(
		a.addr,
		a.storage,
		a.logger,
		reporter.WithSignKey(a.signKey),
		reporter.WithKey(a.publicKey),
		reporter.WithRPC(a.conn))

	ticker := time.NewTicker(a.reportInterval)

	for {
		select {

		case <-ticker.C:
			if err := report.Report(ctx, a.reportType); err != nil {
				a.logger.Err.Printf("report failed with error: %v\n", err)
			}

			// Сброс значения метрики PollCount
			pollCount, _ := metric.CreateMetric(metric.CounterType, "PollCount", metric.WithValueInt(0))
			if err := a.storage.Delete(pollCount); err != nil && err != errs.ErrNotFound {
				a.logger.Err.Printf("error delete metric %s after report: %v\n", pollCount.ShotString(), err)
			}

		case <-ctx.Done():

			if a.conn != nil {
				if err := a.conn.Close(); err != nil {
					a.logger.Err.Printf("failed close gPRC connection: %v\n", err)
				}
			}

			return
		}
	}
}
