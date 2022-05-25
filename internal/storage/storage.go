package storage

import (
	"log"
	"sync"
	"time"

	"metrics-and-alerting/pkg/config"
)

type MemoryStorage struct {
	mu         sync.Mutex
	metrics    []Metrics
	cfg        config.Config
	extStorage ExternalStorage
}

// SetConfig - Инициализация конфигурации
func (st *MemoryStorage) SetConfig(cfg config.Config) {
	st.cfg = cfg
}

// SetExternalStorage - Инициализация внешнего хранилища
func (st *MemoryStorage) SetExternalStorage(extStorage ExternalStorage) {
	st.extStorage = extStorage

	if st.isSyncStore() {
		return
	}

	go func() {

		ticker := time.NewTicker(st.cfg.StoreInterval)
		for {
			<-ticker.C
			if st.extStorage == nil {
				continue
			}

			if err := st.extStorage.WriteAll(st.Data()); err != nil {
				log.Printf("error save in external storage: %v", err)
			}
		}
	}()
}

// ExternalStorage - Получение внешнего хранилища
func (st *MemoryStorage) ExternalStorage() ExternalStorage {
	return st.extStorage
}

// Find - Поиск метрики пл типу и идентификатору.
// Возвращается индекс метрики в слайсе и ошибка, если такой метрики не существует
func (st *MemoryStorage) Find(typeMetric, id string) (int, error) {

	for i, metric := range st.metrics {
		if metric.MType == typeMetric && metric.ID == id {
			return i, nil
		}
	}

	return -1, ErrorNotFound
}

// CreateIfNotExist - Ищет метрику по типу и идентификатору.
// Если метриик не существует, то создается новая с указанными типом и идентификатором.
// Возвращается индекс метрики в слайсе
func (st *MemoryStorage) CreateIfNotExist(typeMetric, id string) (int, error) {

	if len(id) < 1 {
		return -1, ErrorInvalidID
	}

	if typeMetric != GaugeType && typeMetric != CounterType {
		return -1, ErrorInvalidType
	}

	index, errFind := st.Find(typeMetric, id)

	if errFind != nil {
		// Такой метрики еще не существует - добавляем
		if errFind == ErrorNotFound {
			st.metrics = append(st.metrics, NewMetric(typeMetric, id))
			index = len(st.metrics) - 1
		} else {
			return -1, errFind
		}
	}

	return index, nil
}

// Update Обновление значения метрики.
// Для типа "gauge" - значение обновляется на value.
// Для типа "counter" -  старому значению добавляется новое значение value.
func (st *MemoryStorage) Update(metric *Metrics) error {

	// Проверка подписи метрики
	if len(st.cfg.SecretKey) > 0 {

		sign, err := Sign(metric, []byte(st.cfg.SecretKey))
		if err != nil {
			log.Printf("error get sign metric: %s\n", err.Error())
		}

		if sign != metric.Hash {
			return ErrorInvalidSignature
		}
	}

	switch metric.MType {
	case GaugeType:
		return st.Set(metric)
	case CounterType:
		return st.Add(metric)
	default:
		return ErrorUnknownType
	}
}

// Get Получение метрики
func (st *MemoryStorage) Get(typeMetric, id string) (Metrics, error) {

	index, err := st.Find(typeMetric, id)
	if err != nil {
		return Metrics{}, err
	}

	if len(st.cfg.SecretKey) > 0 {
		hash, err := Sign(&st.metrics[index], []byte(st.cfg.SecretKey))
		if err == nil {
			st.metrics[index].Hash = hash
		} else {
			log.Printf("error get sign metric '%s'. %s\n", st.metrics[index].ShotString(), err)
		}
	}

	return st.metrics[index], nil
}

// Set Изменение значения метрики
func (st *MemoryStorage) Set(metric *Metrics) error {

	index, err := st.CreateIfNotExist(metric.MType, metric.ID)
	if err != nil {
		return err
	}

	switch metric.MType {
	case GaugeType:
		if metric.Value == nil {
			return ErrorInvalidValue
		}

		st.metrics[index].Value = metric.Value

	case CounterType:
		if metric.Delta == nil {
			return ErrorInvalidValue
		}

		st.metrics[index].Delta = metric.Delta
	}

	if st.isSyncStore() {
		if err := st.Save(); err != nil {
			log.Printf("error sync save metrics in external storage: %s\n", err.Error())
		}
	}

	return nil
}

// Add Изменение значения метрики.
// Для типа "gauge" - value должно преобразовываться в float64.
// Для типа "counter" - value должно преобразовываться в int64.
func (st *MemoryStorage) Add(metric *Metrics) error {

	index, err := st.CreateIfNotExist(metric.MType, metric.ID)
	if err != nil {
		return err
	}

	switch metric.MType {
	case GaugeType:
		if metric.Value == nil {
			return ErrorInvalidValue
		}

		if st.metrics[index].Value == nil {
			st.metrics[index].Value = metric.Value
		} else {
			*st.metrics[index].Value += *metric.Value
		}

	case CounterType:
		if metric.Delta == nil {
			return ErrorInvalidValue
		}

		if st.metrics[index].Delta == nil {
			st.metrics[index].Delta = metric.Delta
		} else {
			*st.metrics[index].Delta += *metric.Delta
		}
	}

	if st.isSyncStore() {
		if err := st.Save(); err != nil {
			log.Printf("error sync save metrics in external storage: %s\n", err.Error())
		}
	}

	return nil
}

// Data - Получение всех метрик
func (st *MemoryStorage) Data() []Metrics {

	if len(st.cfg.SecretKey) > 0 {
		for index := range st.metrics {
			hash, err := Sign(&st.metrics[index], []byte(st.cfg.SecretKey))
			if err == nil {
				st.metrics[index].Hash = hash
			} else {
				log.Printf("error get sign metric '%s'. %s\n", st.metrics[index].ShotString(), err)
			}
		}
	}

	return st.metrics
}

// ResetMetrics - Сброс метрик
func (st *MemoryStorage) ResetMetrics() {
	st.metrics = nil
}

// Save - Сохранение метрик во внешнем хранилище
func (st *MemoryStorage) Save() error {

	if st.extStorage == nil {
		return ErrorExternalStorage
	}

	if len(st.cfg.SecretKey) > 0 {
		for index := range st.metrics {
			hash, err := Sign(&st.metrics[index], []byte(st.cfg.SecretKey))
			if err == nil {
				st.metrics[index].Hash = hash
			} else {
				log.Printf("error get sign metric '%s'. %s\n", st.metrics[index].ShotString(), err)
			}
		}
	}

	if err := st.extStorage.WriteAll(st.metrics); err != nil {
		return err
	}

	return nil
}

// Restore - Восстановление метрик из внешнего хранилища
func (st *MemoryStorage) Restore() error {

	st.ResetMetrics()

	if st.extStorage == nil {
		return ErrorExternalStorage
	}

	metrics, err := st.extStorage.ReadAll()
	if err != nil {
		return err
	}

	st.metrics = metrics

	return nil
}

func (st *MemoryStorage) isSyncStore() bool {

	if st.extStorage == nil {
		return false
	}

	return st.cfg.StoreInterval == 0
}
