package serverMetrics

import (
	"net/http"
	"reflect"
	"testing"
)

func TestStartMetricsHttpServer(t *testing.T) {
	tests := []struct {
		name string
		want *http.Server
	}{
		{
			want: &http.Server{Addr: ":8080"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StartMetricsHttpServer(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StartMetricsHttpServer() = %v, want %v", got, tt.want)
			}
		})
	}
}
