package storage

type ExternalStorage interface {
	Ping() bool

	ReadAll() ([]Metrics, error)
	WriteAll(metrics []Metrics) error
}
