package metricHandler

import (
	"net/http"
	"reflect"
	"testing"

	storage "github.com/rnikbond/metrics-and-alerting/internal/storage"
)

func TestUpdateMetricGauge(t *testing.T) {
	type args struct {
		metrics storage.Metrics
	}
	tests := []struct {
		name string
		args args
		want http.HandlerFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UpdateMetricGauge(tt.args.metrics); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateMetricGauge() = %v, want %v", got, tt.want)
			}
		})
	}
}
