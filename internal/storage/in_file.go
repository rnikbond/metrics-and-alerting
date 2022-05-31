package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"metrics-and-alerting/pkg/config"
)

type FileStorage struct {
	fileName string
	interval time.Duration
	inMemory InMemoryStorage
}

func (fs *FileStorage) File(flag int) (*os.File, error) {
	if len(fs.fileName) < 1 {
		return nil, ErrInvalidFilePath
	}

	return os.OpenFile(fs.fileName, flag, 0777)
}

func (fs FileStorage) IsAsyncSave() bool {
	return fs.interval == 0
}

func (fs FileStorage) Save() error {
	file, err := fs.File(os.O_CREATE | os.O_WRONLY | os.O_TRUNC)
	if err != nil {
		err = fmt.Errorf("error open file fo rewrite: %w", err)
		log.Println(err)
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	metrics := fs.inMemory.GetData()
	for _, metric := range metrics {

		data, err := json.Marshal(&metric)
		if err != nil {
			log.Printf("error encode metric '%s'. %v\n", metric.ShotString(), err)
			continue
		}

		if _, err = writer.Write(data); err == nil {
			writer.WriteByte('\n')
		} else {
			log.Printf("error write JSON metric '%s' in file storage: %v\n", string(data), err)
		}
	}

	return writer.Flush()
}

func (fs *FileStorage) Restore() error {

	file, err := fs.File(os.O_RDONLY)
	if err != nil {
		err = fmt.Errorf("error open file fo read: %w", err)
		log.Println(err)
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		data := scanner.Bytes()

		metric := Metric{}
		if err := json.Unmarshal(data, &metric); err != nil {
			log.Printf("error decode metric. JSON: %s. %v", string(data), err)
			continue
		}

		if err := fs.inMemory.Update(metric); err != nil {
			log.Printf("error updating metric in memofy file storage: %s. %v", metric.ShotString(), err)
		}
	}

	return nil
}

func (fs *FileStorage) Init(cfg config.Config) error {
	fs.fileName = cfg.StoreFile
	fs.interval = cfg.StoreInterval

	fs.inMemory = InMemoryStorage{}
	if err := fs.inMemory.Init(cfg); err != nil {
		return fmt.Errorf("error init memory storage in file storage: %w", err)
	}

	if cfg.Restore {
		if err := fs.Restore(); err != nil {
			return fmt.Errorf("error restore file storage: %w", err)
		}
	}

	if fs.IsAsyncSave() {
		return nil
	}

	// Запуск горутинки интервального сохранения метрик
	go func() {
		ticker := time.NewTicker(fs.interval)

		for {
			<-ticker.C
			fmt.Println("store in file ...")
			if err := fs.Save(); err != nil {
				log.Printf("error regular save in file storage: %v\n", err)
			}
		}
	}()

	return nil
}

// Update Обновление значения метрики
func (fs *FileStorage) Update(metric Metric) error {

	if err := fs.inMemory.Update(metric); err != nil {
		return fmt.Errorf("error update metric in file storage: %w", err)
	}

	if fs.IsAsyncSave() {
		if err := fs.Save(); err != nil {
			return fmt.Errorf("error save metrics in file storage: %w", err)
		}
	}

	return nil
}

// UpdateData Всех метрик
func (fs *FileStorage) UpdateData(metrics []Metric) error {

	if err := fs.inMemory.UpdateData(metrics); err != nil {
		return fmt.Errorf("error update metric in file storage: %w", err)
	}

	if fs.IsAsyncSave() {
		if err := fs.Save(); err != nil {
			return fmt.Errorf("error save metrics in file storage: %w", err)
		}
	}

	return nil
}

// Get - Получение полность заполненной метрики
func (fs FileStorage) Get(metric Metric) (Metric, error) {
	return fs.inMemory.Get(metric)
}

// GetData - Получение всех, полностью заполненных, метрик
func (fs FileStorage) GetData() []Metric {
	return fs.inMemory.GetData()
}

// Delete - Удаление метрики
func (fs *FileStorage) Delete(metric Metric) error {

	if err := fs.inMemory.Delete(metric); err != nil {
		return fmt.Errorf("error delete metric in memory file storage: %w", err)
	}

	if fs.IsAsyncSave() {
		if err := fs.Save(); err != nil {
			return fmt.Errorf("error save metrics in file storage: %w", err)
		}
	}

	return nil
}

func (fs *FileStorage) Reset() error {
	if err := fs.inMemory.Reset(); err != nil {
		return fmt.Errorf("error reset memory storage in file storage: %w", err)
	}

	file, err := fs.File(os.O_TRUNC)
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("error reset file storage: %w", err)
	}

	return file.Close()
}

func (fs FileStorage) CheckHealth() bool {
	_, err := os.Stat(fs.fileName)
	return !errors.Is(err, os.ErrNotExist)
}

func (fs FileStorage) Destroy() {
	if err := fs.Save(); err != nil {
		log.Printf("error save data in file storage defore destroy: %v\n", err)
	}

	fs.inMemory.Destroy()
}
