package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"metrics-and-alerting/internal/storage"

	"github.com/go-resty/resty/v2"
)

func TestAgent_report(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {}))
	defer server.Close()

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
			name: "test agent metrics #1",
			agent: &Agent{
				ServerURL: server.URL,
				Metrics:   &storage.MetricsData{},
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
			name: "test agent metrics #2",
			agent: &Agent{
				Metrics: &storage.MetricsData{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "test agent metrics #3",
			agent: &Agent{
				Metrics: &storage.MetricsData{},
			},
			args: args{
				ctx:        context.Background(),
				nameMetric: "Alloc",
			},
			wantErr: true,
		},
		{
			name: "test agent metrics #4",
			agent: &Agent{
				Metrics: &storage.MetricsData{},
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
			if err := tt.agent.report(tt.args.ctx, client, tt.args.nameMetric, tt.args.valueMetric, tt.args.typeMetric); (err != nil) != tt.wantErr {
				t.Errorf("AgentMeticsData.report() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	server.Close()
}
