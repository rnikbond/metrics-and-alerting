package storage

type ExternalStorage interface {
	ReadAll() ([]Metrics, error)
	WriteAll(metrics []Metrics) error
}
