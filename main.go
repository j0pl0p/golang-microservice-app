package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/j0pl0p/final-task-GO-YL/messages"
	_ "github.com/mattn/go-sqlite3"
	amqp "github.com/rabbitmq/amqp091-go"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var storage *Storage
var daemonResponses map[string]time.Time
var conn *amqp.Connection
var ch *amqp.Channel
var signCalcDurations map[string]time.Duration = map[string]time.Duration{
	"plus":  20 * time.Millisecond,
	"minus": 20 * time.Millisecond,
	"mul":   20 * time.Millisecond,
	"div":   20 * time.Millisecond,
}

func main() {
	var err error
	storage, err = NewStorage("data/db.db")
	defer storage.db.Close()
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Println("Connected to the database")

	conn, err = amqp.Dial("amqp://defaultuser:defaultpass@localhost:5672/")
	if err != nil {
		log.Fatal("unable to start RMQ server: " + err.Error())
		return
	}
	defer conn.Close()
	ch, err = conn.Channel()
	if err != nil {
		log.Fatal("unable to open channel: " + err.Error())
		return
	}
	defer ch.Close()
	log.Println("RMQ started, channel opened")

	qRes, err := ch.QueueDeclare(
		"resQueue",
		false,
		false,
		false,
		false,
		nil,
	)
	resultsConsumed, err := ch.Consume(
		qRes.Name, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		log.Fatalf("failed to register a consumer. Error: %s", err)
	}
	go func() {
		for res := range resultsConsumed {
			log.Printf("received a resultMessage: %s, saving to storage...", res.Body)
			msg, err := messages.FromBytes[messages.Result](res.Body)
			if err != nil {
				log.Println("cant convert bytes to message")
				continue
			}
			err = storage.SaveResult(msg.Id, msg.Res)
			if err != nil {
				log.Println("cant save result:", err.Error())
				continue
			}
		}
	}()
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")

	r := mux.NewRouter()
	r.HandleFunc("/add-expression", addExpressionHandler).Methods("POST")
	r.HandleFunc("/get-expressions", getExpressionHandler).Methods("GET")
	r.HandleFunc("/get-value", getValueHandler).Methods("GET")
	r.HandleFunc("/set-calc-durations", setCalcDurationsHandler).Methods("POST")
	r.HandleFunc("/add-new-daemon", makeNewDaemonHandler).Methods("GET")
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

type CalcDurationsJSON struct {
	Plus  int `json:"plus"`
	Minus int `json:"minus"`
	Mul   int `json:"mul"`
	Div   int `json:"div"`
}

type Expression struct {
	Exp    string
	Id     string
	Status string
	Result float32
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
	} else {
		http.Error(w, "expression invalid", 400)
		log.Println("ERROR: invalid expression")
		return
	}
	qTask, err := ch.QueueDeclare(
		"tasksQueue",
		false,
		false,
		false,
		false,
		nil,
	)
	tm := messages.Task{
		Id:         id,
		Expression: data.Exp,
		Durations:  signCalcDurations,
	}
	bytes, err := messages.ToBytes[messages.Task](tm)
	if err != nil {
		http.Error(w, "ERROR: "+err.Error(), 500)
		log.Println("cant turn message into bytes")
		return
	}
	err = ch.Publish(
		"",
		qTask.Name,
		false,
		false,
		amqp.Publishing{ContentType: "application/json", Body: bytes},
	)
	if err != nil {
		http.Error(w, "ERROR: "+err.Error(), 500)
		log.Println("cant send the message")
		return
	}
	log.Println("successfully sent message")
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

// Установка новых длительностей вычисления для каждого оператора (+, -, *, /)
func setCalcDurationsHandler(w http.ResponseWriter, r *http.Request) {
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
	var data CalcDurationsJSON
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "error parsing JSON", 500)
		log.Println("ERROR: ", err)
		return
	}
	SetNewCalcDurations(
		time.Duration(data.Plus)*time.Millisecond,
		time.Duration(data.Minus)*time.Millisecond,
		time.Duration(data.Mul)*time.Millisecond,
		time.Duration(data.Div)*time.Millisecond,
	)
	log.Println("successfully set new calc durations")
}

// HeartbeatMonitoring Мониторинг активности демонов
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

// SetNewCalcDurations Установка новых настроек длительности расчета каждой операции (+, - *, /)
func SetNewCalcDurations(plus, minus, mul, div time.Duration) {
	signCalcDurations = map[string]time.Duration{
		"plus":  plus,
		"minus": minus,
		"mul":   mul,
		"div":   div,
	}
}

// Хендлер для получения новых ID для демонов
func makeNewDaemonHandler(w http.ResponseWriter, r *http.Request) {
	id := uuid.NewString()
	err := storage.AddNewDaemon(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Println("ERROR: ", err)
		return
	}
	err = json.NewEncoder(w).Encode(id)
	return
}
