package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

type jsonReq struct {
	url     string
	method  string
	jsonstr []byte
}

func main() {
	request := []jsonReq{
		{"http://localhost:8080/transactionse", "GET", nil},
		{"http://localhost:8080/transactionservice/transaction/10", "PUT", []byte(`{ "amount": 5000, "type":"cars" }`)},
		{"http://localhost:8080/transactionservice/transaction/11", "PUT", []byte(`{ "amount": 10000, "type": "shopping", "parent_id": 10}`)},
		{"http://localhost:8080/transactionservice/transaction/10", "GET", nil},
		{"http://localhost:8080/transactionservice/transaction/11", "GET", nil},
		{"http://localhost:8080/transactionservice/types/cars", "GET", nil},
		{"http://localhost:8080/transactionservice/sum/10", "GET", nil},
		{"http://localhost:8080/transactionservice/sum/11", "GET", nil},
		{"http://localhost:8080/transactionservice/sum/12", "GET", nil},
	}

	for _, r := range request {
		fmt.Println("URL:>", r.url)
		req, err := http.NewRequest(r.method, r.url, bytes.NewBuffer(r.jsonstr))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("response Body:", string(body))
		resp.Body.Close()
	}

}
