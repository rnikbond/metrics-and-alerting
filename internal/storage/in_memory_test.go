package storage

import (
	"strconv"
	"testing"
)

func BenchmarkInMemoryStorage_Find(b *testing.B) {

	memStore := InMemoryStorage{}

	for i := 0; i < 1000; i++ {

		var delta = int64(i)

		m := Metric{
			ID:    "testMetric_" + strconv.Itoa(i),
			MType: CounterType,
			Delta: &delta,
		}

		if err := memStore.Upsert(m); err != nil {
			b.Errorf("error upsert metric: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		var delta = int64(i)

		m := Metric{
			ID:    "testMetric_" + strconv.Itoa(i),
			MType: CounterType,
			Delta: &delta,
		}

		if err := memStore.Upsert(m); err != nil {
			b.Errorf("error upsert metric: %v", err)
		}
	}
}

func BenchmarkInMemoryStorage_Upsert(b *testing.B) {

	memStore := InMemoryStorage{}

	for i := 0; i < b.N; i++ {

		var delta = int64(i)

		m := Metric{
			ID:    "testMetric_" + strconv.Itoa(i),
			MType: CounterType,
			Delta: &delta,
		}

		if err := memStore.Upsert(m); err != nil {
			b.Errorf("error upsert metric: %v", err)
		}
	}
}
