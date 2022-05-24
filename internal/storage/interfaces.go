package storage

type ExternalStorage interface {
	Close() error
	CheckHealth() bool

	ReadAll() ([]Metrics, error)
	WriteAll(metrics []Metrics) error
}
