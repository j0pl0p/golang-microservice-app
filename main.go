package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"net/http"
)

var storage *Storage

func main() {
	var err error
	storage, err = NewStorage("data/db.db")
	defer storage.db.Close()
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Println("Connected to the database")
	r := mux.NewRouter()
	r.HandleFunc("/add-expression", addExpressionHandler).Methods("POST")
	r.HandleFunc("/get-expressions", getExpressionHandler).Methods("GET")
	r.HandleFunc("/get-value", getValueHandler).Methods("POST")
	r.HandleFunc("/get-operations", getOperationsHandler).Methods("GET")
	r.HandleFunc("/get-task", getTaskHandler).Methods("GET")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		return
	}
}

type ExpressionDataJSON struct {
	Exp string `json:"expression"`
}

type Expression struct {
	Exp    string
	Id     string
	Status string
}

// Добавление вычисления арифметического выражения
func addExpressionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		log.Println("ERROR: method not allowed")
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cant read body", 400)
		log.Println("ERROR: ", err)
		return
	}
	var data ExpressionDataJSON
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "error parsing JSON", 500)
		log.Println("ERROR: ", err)
		return
	}
	id := data.Exp // TODO: make a normal ids for expressions, not the expression themselves
	// TODO: valid checking
	if isValid := true; isValid {
		// TODO: id already exists checking
		id, err := storage.Add(data.Exp, id)
		if err != nil {
			http.Error(w, "something went wrong while adding the expression", 500)
			log.Println("ERROR: something went wrong while adding the expression")
			return
		}
		log.Println("expression added: ", id)
		_, _ = fmt.Fprint(w, "DONE: ", id)
		return
	} else {
		http.Error(w, "expression invalid", 400)
		log.Println("ERROR: invalid expression")
		return
	}
}

// Получение списка выражений со статусами
func getExpressionHandler(w http.ResponseWriter, r *http.Request) {
	data, err := storage.GetAll()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	err = json.NewEncoder(w).Encode(&data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Println("ERROR: ", err.Error())
		return
	}
	log.Println("successfully returned all expressions and statuses")
	return
}

// Получение значения выражения по его идентификатору
func getValueHandler(w http.ResponseWriter, r *http.Request) {}

// Получение списка доступных операций со временем их выполения
func getOperationsHandler(w http.ResponseWriter, r *http.Request) {}

// Получение задачи для выполения
func getTaskHandler(w http.ResponseWriter, r *http.Request) {}
