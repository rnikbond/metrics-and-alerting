package filestorage

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"metrics-and-alerting/internal/storage/memorystorage"
	"metrics-and-alerting/pkg/errs"
	"metrics-and-alerting/pkg/logpack"
	metricPkg "metrics-and-alerting/pkg/metric"
)

type Storage struct {
	fileName      string
	intervalFlush time.Duration
	logger        *logpack.LogPack
	memory        *memorystorage.MemoryStorage
	ctx           context.Context
	cancel        context.CancelFunc
}

func New(fileName string, intervalFlush time.Duration, logger *logpack.LogPack) *Storage {

	store := &Storage{
		fileName:      fileName,
		intervalFlush: intervalFlush,
		logger:        logger,
		memory:        memorystorage.NewStorage(),
	}

	store.ctx, store.cancel = context.WithCancel(context.Background())

	if store.asyncSave() {
		return nil
	}

	// Запуск задачи для сброса матрик в файл
	// TODO :: Вынести в сервис Flusher
	go func() {
		ticker := time.NewTicker(store.intervalFlush)

		for {
			select {
			case <-ticker.C:
				if err := store.save(); err != nil {
					store.logger.Err.Printf("Cold not flush metrics in file: %v\n", err)
				}

			case <-store.ctx.Done():
				return
			}

		}
	}()

	return store
}

func (store Storage) openFile(flag int) (*os.File, error) {
	if len(store.fileName) < 1 {
		return nil, errs.ErrInvalidFilePath
	}

	return os.OpenFile(store.fileName, flag, 0777)
}

func (store Storage) asyncSave() bool {
	return store.intervalFlush == 0
}

func (store Storage) save() error {
	file, err := store.openFile(os.O_CREATE | os.O_WRONLY | os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("error open fileStorage fo rewrite: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			store.logger.Err.Printf("could not close file after save: %v\n", err)
		}
	}()

	writer := bufio.NewWriter(file)
	metrics, errMemory := store.memory.GetSlice()
	if errMemory != nil {
		return fmt.Errorf("could not save metrics. Memory storage returned error: %w", err)
	}

	data, err := json.Marshal(&metrics)
	if err != nil {
		return fmt.Errorf("could not save metrics. Marshal slice metrics retured error: %w", err)
	}

	if _, err = writer.Write(data); err != nil {
		return fmt.Errorf("could not save metrics. Can not write in file: %w", err)
	}

	return writer.Flush()
}

func (store *Storage) Restore() error {

	file, err := store.openFile(os.O_RDONLY)
	if err != nil {
		err = fmt.Errorf("could not restore metrics. Can not open file for read: %w", err)
		store.logger.Err.Println(err)
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			store.logger.Err.Printf("could not close file after restore: %v\n", err)
		}
	}()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		data := scanner.Bytes()

		var metrics []metricPkg.Metric

		if err := json.Unmarshal(data, &metrics); err != nil {
			store.logger.Err.Printf("could not restore metrics. Can not Unmarshal from file: %v\n", err)
			continue
		}

		if err := store.memory.UpsertSlice(metrics); err != nil {
			store.logger.Err.Printf("could not restore metrics. Can not write in memory storage: %v\n", err)
		}
	}

	return nil
}

func (store *Storage) Set(metric metricPkg.Metric) error {
	if err := store.memory.Set(metric); err != nil {
		err = fmt.Errorf("could not set metric: %w", err)
		store.logger.Err.Println(err)
		return err
	}

	if store.asyncSave() {
		if err := store.save(); err != nil {
			store.logger.Err.Printf("could not flush metrics: %v\n", err)
		}
	}

	return nil
}

func (store *Storage) Upsert(metric metricPkg.Metric) error {

	if err := store.memory.Upsert(metric); err != nil {
		err = fmt.Errorf("could not upsert metric: %w", err)
		store.logger.Err.Println(err)
		return err
	}

	if store.asyncSave() {
		if err := store.save(); err != nil {
			store.logger.Err.Printf("could not flush metrics: %v\n", err)
		}
	}

	return nil
}

func (store *Storage) UpsertSlice(metrics []metricPkg.Metric) error {

	if err := store.memory.UpsertSlice(metrics); err != nil {
		err = fmt.Errorf("error update batch metrics in file storage: %w", err)
		store.logger.Err.Println(err)
		return err
	}

	if store.asyncSave() {
		if err := store.save(); err != nil {
			store.logger.Err.Printf("could not flush metrics: %v\n", err)
		}
	}

	return nil
}

func (store *Storage) Get(metric metricPkg.Metric) (metricPkg.Metric, error) {
	return store.memory.Get(metric)
}

func (store *Storage) GetSlice() ([]metricPkg.Metric, error) {
	return store.memory.GetSlice()
}

// Delete - Удаление метрики
func (store *Storage) Delete(metric metricPkg.Metric) error {

	if err := store.memory.Delete(metric); err != nil {
		err = fmt.Errorf("could not delete metric: %w", err)
		store.logger.Err.Println(err)
		return err
	}

	if store.asyncSave() {
		if err := store.save(); err != nil {
			store.logger.Err.Printf("could not flush metrics: %v\n", err)
		}
	}

	return nil
}

func (store *Storage) String() string {
	return store.memory.String()
}

func (store *Storage) CheckHealth() bool {
	_, err := os.Stat(store.fileName)
	return !errors.Is(err, os.ErrNotExist)
}

func (store *Storage) Close() error {

	store.cancel()

	if store.asyncSave() {
		return nil
	}

	if err := store.save(); err != nil {
		store.logger.Err.Printf("could not flush metrics: %v\n", err)
	}

	return nil
}
