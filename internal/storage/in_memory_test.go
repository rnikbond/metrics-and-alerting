package storage

import (
	"strconv"
	"testing"

	"metrics-and-alerting/internal/storage/memstore"
	"metrics-and-alerting/pkg/metric"
)

func BenchmarkInMemoryStorage_Upsert(b *testing.B) {

	memStore := memstore.Storage{}

	for i := 0; i < b.N; i++ {

		var delta = int64(i)

		m := metric.Metric{
			ID:    "testMetric_" + strconv.Itoa(i),
			MType: metric.CounterType,
			Delta: &delta,
		}

		if err := memStore.Upsert(m); err != nil {
			b.Errorf("error upsert metric: %v", err)
		}
	}
}
