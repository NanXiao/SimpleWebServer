package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"sync"
)

const (
	statusOK                       = "ok"
	statusNoSpecifiedID            = "Can't find the specified transaction ID"
	statusTransactionEncodingError = "Transaction encoidng is error"
	statusNoSpecifiedType          = "Can't find the specified type"
)

type transactionID uint64

type Transaction struct {
	Amount    float64       `json:"amount"`
	TransType string        `json:"type"`
	ParentID  transactionID `json:"parent_id,omitempty"`
}

type TransactionSum struct {
	Sum float64 `json:"sum"`
}

type TransactionStatus struct {
	Status string `json:"status"`
}

var transactionTable map[transactionID]*Transaction
var typeTable map[string][]transactionID
var sumTable map[transactionID]*TransactionSum

var transactionTableLock sync.RWMutex
var typeTableLock sync.RWMutex
var sumTableLock sync.RWMutex

type routeMethod struct {
	method      string
	pattern     string
	handlerFunc http.HandlerFunc
}

var routeMethods = []routeMethod{
	{method: "PUT", pattern: "/transaction/{transaction_id}", handlerFunc: putTransactionHandler},
	{method: "GET", pattern: "/transaction/{transaction_id}", handlerFunc: getTransactionHandler},
	{method: "GET", pattern: "/types/{type}", handlerFunc: getTypeHandler},
	{method: "GET", pattern: "/sum/{transaction_id}", handlerFunc: getSumHandler},
}

func convertTransactionID(s string) (transactionID, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	return transactionID(v), err
}

func sendResponse(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if reflect.TypeOf(v).String() == "main.TransactionStatus" {
		if v.(TransactionStatus).Status != statusOK {
			w.WriteHeader(http.StatusBadRequest)
		}
	}

	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func putTransactionHandler(w http.ResponseWriter, r *http.Request) {
	var trans Transaction
	var duplicated bool

	id, err := convertTransactionID(mux.Vars(r)["transaction_id"])
	if err != nil {
		sendResponse(w, TransactionStatus{statusNoSpecifiedID})
		return
	}

	if err = json.NewDecoder(r.Body).Decode(&trans); err != nil {
		sendResponse(w, TransactionStatus{statusTransactionEncodingError})
		return
	}

	transactionTableLock.Lock()
	if _, ok := transactionTable[id]; ok {
		duplicated = true
	}
	transactionTable[id] = &trans
	transactionTableLock.Unlock()

	typeTableLock.Lock()
	if !duplicated {
		typeTable[trans.TransType] = append(typeTable[trans.TransType], id)
	}
	typeTableLock.Unlock()

	sumTableLock.Lock()
	if trans.ParentID != 0 {
		if _, ok := sumTable[trans.ParentID]; ok {
			sumTable[trans.ParentID].Sum += trans.Amount
		} else {
			sumTable[trans.ParentID] = &TransactionSum{Sum: trans.Amount}
		}
	} else {
		sumTable[id] = &TransactionSum{Sum: trans.Amount}
	}
	sumTableLock.Unlock()

	sendResponse(w, TransactionStatus{statusOK})
}

func getTransactionHandler(w http.ResponseWriter, r *http.Request) {
	id, err := convertTransactionID(mux.Vars(r)["transaction_id"])
	if err != nil {
		sendResponse(w, TransactionStatus{statusNoSpecifiedID})
		return
	}

	transactionTableLock.RLock()
	defer transactionTableLock.RUnlock()

	if trans, ok := transactionTable[id]; !ok {
		sendResponse(w, TransactionStatus{statusNoSpecifiedID})
	} else {
		sendResponse(w, trans)
	}
}

func getTypeHandler(w http.ResponseWriter, r *http.Request) {
	typeTableLock.RLock()
	defer typeTableLock.RUnlock()

	if transIDs, ok := typeTable[mux.Vars(r)["type"]]; !ok {
		sendResponse(w, TransactionStatus{statusNoSpecifiedType})
	} else {
		sendResponse(w, transIDs)
	}
}

func getSumHandler(w http.ResponseWriter, r *http.Request) {
	id, err := convertTransactionID(mux.Vars(r)["transaction_id"])
	if err != nil {
		sendResponse(w, TransactionStatus{statusNoSpecifiedID})
		return
	}

	sumTableLock.RLock()
	if tranSum, ok := sumTable[id]; ok {
		sendResponse(w, tranSum)
		sumTableLock.RUnlock()
		return
	}
	sumTableLock.RUnlock()

	transactionTableLock.RLock()
	if trans, ok := transactionTable[id]; ok {
		sendResponse(w, TransactionSum{trans.Amount})
		transactionTableLock.RUnlock()
		return
	}
	transactionTableLock.RUnlock()

	sendResponse(w, TransactionStatus{statusNoSpecifiedID})
}

func main() {
	transactionTable = make(map[transactionID]*Transaction)
	typeTable = make(map[string][]transactionID)
	sumTable = make(map[transactionID]*TransactionSum)

	router := mux.NewRouter().PathPrefix("/transactionservice").Subrouter().StrictSlash(true)
	for _, v := range routeMethods {
		router.Methods(v.method).Path(v.pattern).HandlerFunc(v.handlerFunc)
	}

	log.Fatal(http.ListenAndServe("localhost:8080", router))
}
