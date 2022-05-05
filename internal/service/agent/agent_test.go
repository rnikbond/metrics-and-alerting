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
			name: "TestAgentReport-GaugeType =>[OK]",
			agent: &Agent{
				ServerURL: server.URL,
				Storage:   &storage.MemoryStorage{},
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
			if err := tt.agent.reportURL(tt.args.ctx, client, tt.args.nameMetric, tt.args.valueMetric, tt.args.typeMetric); (err != nil) != tt.wantErr {
				t.Errorf("AgentMeticsData.report() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	server.Close()
}
