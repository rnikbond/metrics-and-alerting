package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"metrics-and-alerting/internal/storage"
	"metrics-and-alerting/pkg/config"

	"github.com/go-resty/resty/v2"
)

func TestAgent_report(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {}))
	defer server.Close()

	cfg := config.Config{
		Addr:           server.URL,
		ReportInterval: 10 * time.Second,
		PollInterval:   2 * time.Second,
	}

	type args struct {
		ctx         context.Context
		nameMetric  string
		valueMetric string
		typeMetric  string
	}
	tests := []struct {
		name    string
		agent   *Agent
		args    args
		wantErr bool
	}{
		{
			name: "TestAgentReport-GaugeType =>[OK]",
			agent: &Agent{
				Config:  &cfg,
				Storage: &storage.MemoryStorage{},
			},
			args: args{
				ctx:         context.Background(),
				nameMetric:  "Alloc",
				valueMetric: "1.1",
				typeMetric:  storage.GaugeType,
			},
			wantErr: false,
		},
		{
			name: "TestAgentReport-EmptyMetric =>[Error]",
			agent: &Agent{
				Config:  &cfg,
				Storage: &storage.MemoryStorage{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "TestAgentReport-Without: Type and Value =>[Error]",
			agent: &Agent{
				Config:  &cfg,
				Storage: &storage.MemoryStorage{},
			},
			args: args{
				ctx:        context.Background(),
				nameMetric: "Alloc",
			},
			wantErr: true,
		},
		{
			name: "TestAgentReport-Without: Type and Name =>[Error]",
			agent: &Agent{
				Config:  &cfg,
				Storage: &storage.MemoryStorage{},
			},
			args: args{
				ctx:         context.Background(),
				valueMetric: "1.1",
			},
			wantErr: true,
		},
	}

	client := resty.New()

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			m := storage.Metrics{
				ID:    tt.args.nameMetric,
				MType: tt.args.typeMetric,
			}

			switch m.MType {
			case storage.GaugeType:
				val, _ := strconv.ParseFloat(tt.args.valueMetric, 64)
				m.Value = &val
			case storage.CounterType:
				val, _ := strconv.ParseInt(tt.args.valueMetric, 10, 64)
				m.Delta = &val
			}

			if err := tt.agent.reportURL(tt.args.ctx, client, &m); (err != nil) != tt.wantErr {
				t.Errorf("AgentMeticsData.report() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	server.Close()
}
