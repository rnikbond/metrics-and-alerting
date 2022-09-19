package storage

import (
	"testing"
)

func BenchmarkCreateMetric(b *testing.B) {

	b.Run("Interface", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CreateMetric(GaugeType, "testGauge", 123.00101011)
		}
	})

	b.Run("FunctionOptions", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewMetric(GaugeType, "testGauge", WithValueFloat64(123.00101011))
		}
	})
}
