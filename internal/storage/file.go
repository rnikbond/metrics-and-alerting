package storage

import (
	"bufio"
	"errors"
	"io/fs"
	"log"
	"os"
)

type FileStorage struct {
	FileName string
}

func (fileStore *FileStorage) File(flag int) (*os.File, error) {
	if len(fileStore.FileName) < 1 {
		return nil, ErrorInvalidFilePath
	}

	return os.OpenFile(fileStore.FileName, flag, 0777)
}

func (fileStore FileStorage) ReadAll() ([]Metrics, error) {

	file, err := fileStore.File(os.O_RDONLY)
	if err != nil {
		log.Println("error open file fo read: ", err)
		return []Metrics{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var metrics []Metrics

	for scanner.Scan() {
		data := scanner.Bytes()
		metric, errDecode := FromJSON(data)
		if errDecode != nil {
			log.Printf("error decode metric '%s'. %s\n", string(data), errDecode)
			continue
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

func (fileStore FileStorage) WriteAll(metrics []Metrics) error {
	file, err := fileStore.File(os.O_CREATE | os.O_WRONLY | os.O_TRUNC)
	if err != nil {
		log.Println("error open file fo write: ", err)
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, metric := range metrics {

		data, err := metric.ToJSON()
		if err != nil {
			log.Printf("error encode metric '%s'. %s\n", metric.ShotString(), err)
			continue
		}

		if _, err = writer.Write(data); err == nil {
			writer.WriteByte('\n')
		} else {
			log.Printf("error write metric in JSON view '%s'. %s\n", string(data), err)
		}
	}

	return writer.Flush()
}

func (fileStore FileStorage) Ping() bool {
	_, err := os.Stat(fileStore.FileName)
	return !errors.Is(err, fs.ErrNotExist)
}
