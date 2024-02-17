package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var storage *Storage
var daemonResponses map[string]time.Time
var daemons map[string]Daemon

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
	r.HandleFunc("/get-value", getValueHandler).Methods("GET")
	go HeartbeatMonitoring(time.Second * 20)
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal("failed to launch server")
		return
	}
}

type ExpressionDataJSON struct {
	Exp string `json:"expression"`
}

type IdReceiveJSON struct {
	Id string `json:"id"`
}

type Expression struct {
	Exp    string
	Id     string
	Status string
	Result int32
}

func stringToHash(str string) string {
	hasher := sha256.New()
	hasher.Write([]byte(str))
	hashedString := fmt.Sprintf("%x", hasher.Sum(nil))

	return hashedString
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
	id := stringToHash(data.Exp)
	// TODO: valid checking
	if isValid := true; isValid {
		_, ok := storage.GetById(id)
		if ok {
			_ = json.NewEncoder(w).Encode("expression already exists (" + id + ")")
			log.Println("expression already exists: ", id)
			return
		}
		id, err := storage.Add(id, data.Exp)
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
	// TODO: put expression in rabbitMQ queue
}

// Получение списка выражений со статусами
func getExpressionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		log.Println("ERROR: method not allowed")
		return
	}
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
func getValueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
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
	var data IdReceiveJSON
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "error parsing JSON", 500)
		log.Println("ERROR: ", err)
		return
	}
	exp, ok := storage.GetById(data.Id)
	if !ok {
		http.Error(w, "such expression doesnt exist", 400)
		log.Println("such expression doesnt exist: ", data.Id)
		return
	}
	if exp.Status == "done" {
		err = json.NewEncoder(w).Encode(exp.Result)
		log.Println("successfully returned result of: " + data.Id)
		return
	} else {
		http.Error(w, "the expression isn't calculated yet", 400)
		log.Println("the expression isn't calculated yet: ", data.Id)
		return
	}
}

func HeartbeatMonitoring(d time.Duration) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cur := time.Now()
			for daemonId, lastBeat := range daemonResponses {
				if cur.Sub(lastBeat) > d {
					log.Println("daemon dead:", daemonId)
					err := storage.UpdateDaemon(daemonId, "dead")
					if err != nil {
						log.Println("cant update daemon: ", daemonId)
					} else {
						log.Println("daemon updated successfully")
					}
				}
			}
		}
	}
}
