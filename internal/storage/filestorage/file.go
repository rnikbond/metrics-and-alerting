package filestorage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"metrics-and-alerting/internal/storage/memstore"
	"metrics-and-alerting/pkg/errs"
	"metrics-and-alerting/pkg/logpack"
	metricPkg "metrics-and-alerting/pkg/metric"
)

type Storage struct {
	fileName string
	logger   *logpack.LogPack
	memory   *memstore.Storage
}

func New(fileName string, logger *logpack.LogPack) *Storage {

	store := &Storage{
		fileName: fileName,
		logger:   logger,
		memory:   memstore.New(),
	}

	return store
}

func (store Storage) open(flag int) (*os.File, error) {
	if len(store.fileName) < 1 {
		return nil, errs.ErrInvalidFilePath
	}

	return os.OpenFile(store.fileName, flag, 0777)
}

func (store Storage) Flush() error {

	file, errFile := store.open(os.O_CREATE | os.O_WRONLY | os.O_TRUNC)
	if errFile != nil {
		return fmt.Errorf("error open fileStorage fo rewrite: %w", errFile)
	}

	defer func() {
		if err := file.Close(); err != nil {
			store.logger.Err.Printf("Could not close file after flush: %v\n", err)
		}
	}()

	writer := bufio.NewWriter(file)
	metrics, errMemory := store.memory.GetBatch()
	if errMemory != nil {
		return fmt.Errorf("could not save metrics. Memory storage returned error: %w", errMemory)
	}

	data, errEncode := json.Marshal(&metrics)
	if errEncode != nil {
		return fmt.Errorf("could not save metrics. Marshal slice metrics retured error: %w", errEncode)
	}

	if _, errWrite := writer.Write(data); errWrite != nil {
		return fmt.Errorf("could not save metrics. Can not write in file: %w", errWrite)
	}

	return writer.Flush()
}

func (store *Storage) Restore() error {

	file, err := store.open(os.O_RDONLY)
	if err != nil {
		return fmt.Errorf("could not restore metrics. Can not open file for read: %w", err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			store.logger.Err.Printf("Could not close file after restore: %v\n", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		data := scanner.Bytes()

		var metrics []metricPkg.Metric

		if err := json.Unmarshal(data, &metrics); err != nil {
			return fmt.Errorf("could not restore metrics. Can not Unmarshal from file: %w", err)
		}

		if err := store.memory.UpsertBatch(metrics); err != nil {
			return fmt.Errorf("could not restore metrics. Can not write in memory storage: %w", err)
		}
	}

	return nil
}

func (store *Storage) Upsert(metric metricPkg.Metric) error {

	if err := store.memory.Upsert(metric); err != nil {
		return fmt.Errorf("could not upsert metric: %w", err)
	}

	return nil
}

func (store *Storage) UpsertBatch(metrics []metricPkg.Metric) error {

	if err := store.memory.UpsertBatch(metrics); err != nil {
		return fmt.Errorf("error update batch metrics in file storage: %w", err)
	}

	return nil
}

func (store Storage) Get(metric metricPkg.Metric) (metricPkg.Metric, error) {
	return store.memory.Get(metric)
}

func (store Storage) GetBatch() ([]metricPkg.Metric, error) {
	return store.memory.GetBatch()
}

// Delete - Удаление метрики
func (store *Storage) Delete(metric metricPkg.Metric) error {

	if err := store.memory.Delete(metric); err != nil {
		return fmt.Errorf("could not delete metric: %w", err)
	}

	return nil
}

func (store *Storage) Health() bool {
	_, err := os.Stat(store.fileName)
	return !errors.Is(err, os.ErrNotExist)
}

func (store *Storage) Close() error {
	return nil
}
