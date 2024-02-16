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

type ExpressionData struct {
	Exp string `json:"expression"`
}

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
	var data ExpressionData
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

func getExpressionHandler(w http.ResponseWriter, r *http.Request) {}

func getValueHandler(w http.ResponseWriter, r *http.Request) {}

func getOperationsHandler(w http.ResponseWriter, r *http.Request) {}

func getTaskHandler(w http.ResponseWriter, r *http.Request) {}
