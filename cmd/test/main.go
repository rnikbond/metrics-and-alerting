package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	handler "metrics-and-alerting/internal/server/handlers"
	"metrics-and-alerting/internal/storage"

	"github.com/go-resty/resty/v2"
)

func getJSON() {
	metric := storage.Metrics{
		ID:    "PollCount",
		MType: storage.CounterType,
	}

	data, err := json.Marshal(metric)
	if err != nil {
		log.Println(err.Error())
		return
	}

	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		Post("http://127.0.0.1:8080" + handler.PartURLValue)

	if err != nil {
		log.Println(err.Error())
		return
	}

	fmt.Printf("[get JSON] response status: %d\n", resp.StatusCode())

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		log.Println(" \nJSON: " + string(data) +
			".\nMetric: " + metric.MType + "/" + metric.ID +
			".\nFailed update metric: " + resp.Status() + ". " + string(respBody))
	} else {
		respBody := resp.Body()
		if err := json.Unmarshal(respBody, &metric); err != nil {
			fmt.Println("error unmarshal response: ", err.Error())
		} else {
			fmt.Println("get answer: ", metric.String())
		}
	}
}

func updateJSON() {

	var val int64
	val = 123

	metric := storage.Metrics{
		ID:    "PollCount",
		MType: storage.CounterType,
		Delta: &val,
	}

	data, err := json.Marshal(metric)
	if err != nil {
		log.Println(err.Error())
		return
	}

	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		Post("http://127.0.0.1:8080" + handler.PartURLUpdate)

	if err != nil {
		log.Println(err.Error())
		return
	}

	fmt.Printf("[updateJSON] response status: %d\n", resp.StatusCode())

	if resp.StatusCode() != http.StatusOK {
		respBody := resp.Body()
		log.Println(" \nJSON: " + string(data) +
			".\nMetric: " + metric.MType + "/" + metric.ID +
			".\nFailed update metric: " + resp.Status() + ". " + string(respBody))
	}
}

func main() {

	updateJSON()
	getJSON()
}
