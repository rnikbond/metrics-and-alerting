package storage

import (
	"errors"
	"fmt"
	"log"

	"metrics-and-alerting/pkg/config"
)

type InMemoryStorage struct {
	metrics  []Metric
	signKey  []byte
	isVerify bool
}

// VerifySign - Проверка подписи метрики
func (ims InMemoryStorage) VerifySign(metric Metric) error {
	if !ims.isVerify {
		return nil
	}

	if len(ims.signKey) < 1 {
		return nil
	}

	hash, err := Sign(metric, ims.signKey)
	if err != nil {
		return err
	}

	if hash != metric.Hash {
		return ErrSignFailed
	}
	return nil
}

// Find - Поиск метрики пл типу и идентификатору.
// Возвращается индекс метрики в слайсе и ошибка, если такой метрики не существует
func (ims *InMemoryStorage) Find(typeMetric, id string) (int, error) {

	for i, metric := range ims.metrics {
		if metric.MType == typeMetric && metric.ID == id {
			return i, nil
		}
	}

	return -1, ErrNotFound
}

// CreateIfNotExist - Ищет метрику по типу и идентификатору.
// Если метриик не существует, то создается новая с указанными типом и идентификатором.
// Возвращается индекс метрики в слайсе
func (ims *InMemoryStorage) CreateIfNotExist(typeMetric, id string) (int, error) {

	if len(id) < 1 {
		return 0, ErrInvalidID
	}

	if typeMetric != GaugeType && typeMetric != CounterType {
		return 0, ErrInvalidType
	}

	idx, err := ims.Find(typeMetric, id)
	// Такой метрики еще не существует - добавляем
	if errors.Is(err, ErrNotFound) {
		ims.metrics = append(ims.metrics, Metric{
			ID:    id,
			MType: typeMetric,
		})
		idx = len(ims.metrics) - 1
		err = nil
	}

	return idx, err
}

func (ims *InMemoryStorage) Init(cfg config.Config) error {
	ims.signKey = []byte(cfg.SecretKey)
	ims.isVerify = cfg.VerifyOnUpdate
	return nil
}

// Upset Обновление значения метрики
// Для типа "gauge" - значение обновляется на value
// Для типа "counter" -  старому значению добавляется новое значение value
func (ims *InMemoryStorage) Upset(metric Metric) error {

	if err := ims.VerifySign(metric); err != nil {
		return fmt.Errorf("can not update metric: %w", err)
	}

	switch metric.MType {
	case GaugeType:
		if metric.Value == nil {
			return fmt.Errorf("can not update metric: %w", ErrInvalidValue)
		}

		idx, err := ims.CreateIfNotExist(metric.MType, metric.ID)
		if err != nil {
			return fmt.Errorf("can not update metric: %w", err)
		}

		ims.metrics[idx].Value = metric.Value

	case CounterType:
		if metric.Delta == nil {
			return fmt.Errorf("can not update metric: %w", ErrInvalidValue)
		}

		idx, err := ims.CreateIfNotExist(metric.MType, metric.ID)
		if err != nil {
			return fmt.Errorf("can not update metric: %w", err)
		}

		if ims.metrics[idx].Delta == nil {
			ims.metrics[idx].Delta = metric.Delta
		} else {
			*ims.metrics[idx].Delta += *metric.Delta
		}

	default:
		return fmt.Errorf("can not update metric: %w", ErrUnknownType)
	}

	return nil
}

// UpsetData - Обновление всех метрик
func (ims *InMemoryStorage) UpsetData(metrics []Metric) error {
	for _, metric := range metrics {
		if err := ims.Upset(metric); err != nil {
			return fmt.Errorf("can not update metrics: %w", err)
		}
	}

	return nil
}

// Get - Получение полность заполненной метрики
func (ims InMemoryStorage) Get(metric Metric) (Metric, error) {

	idx, err := ims.Find(metric.MType, metric.ID)
	if err != nil {
		return Metric{}, err
	}

	if hash, err := Sign(ims.metrics[idx], ims.signKey); err == nil {
		ims.metrics[idx].Hash = hash
	}

	return ims.metrics[idx], nil
}

// GetData - Получение всех, полностью заполненных, метрик
func (ims InMemoryStorage) GetData() []Metric {

	if len(ims.signKey) > 0 {
		for idx := range ims.metrics {
			if hash, err := Sign(ims.metrics[idx], ims.signKey); err == nil {
				ims.metrics[idx].Hash = hash
			}
		}
	}

	return ims.metrics
}

// Delete - Удаление метрики
func (ims *InMemoryStorage) Delete(metric Metric) error {

	idx, err := ims.Find(metric.MType, metric.ID)
	if err != nil {
		return err
	}

	ims.metrics = append(ims.metrics[:idx], ims.metrics[idx+1:]...)
	return nil
}

func (ims *InMemoryStorage) Reset() error {
	ims.metrics = ims.metrics[:0]
	return nil
}

func (ims InMemoryStorage) CheckHealth() bool {
	return true
}

func (ims InMemoryStorage) Destroy() {
	log.Println("Destroy memory storage... Goodbye :)")
}
