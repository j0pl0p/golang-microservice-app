package main

import (
	"fmt"
	"github.com/Knetic/govaluate"
	"github.com/j0pl0p/final-task-GO-YL/messages"
	amqp "github.com/rabbitmq/amqp091-go"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

// Daemon Структура демона
type Daemon struct {
	Id     string
	Status string
	Ch     *amqp.Channel
}

// NewDaemon Создание нового демона
func NewDaemon() *Daemon {
	resp, err := http.Get("http://localhost:8080/add-new-daemon")
	if err != nil {
		log.Println("cant make a get req", err.Error())
		return nil
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("cant read response body")
		return nil
	}
	id := string(responseBody)
	id = strings.Replace(id, `"`, ``, -1)
	id = strings.Replace(id, "\n", "", -1)
	log.Println(id)
	if err != nil {
		return nil
	}

	conn, err := amqp.Dial("amqp://defaultuser:defaultpass@localhost:5672/")
	ch, err := conn.Channel()

	return &Daemon{
		Id:     id,
		Status: "active",
		Ch:     ch,
	}
}

// UpdateStatus Обновление статуса демона
func (d *Daemon) UpdateStatus(newStatus string) {
	d.Status = newStatus
}

func main() {
	daemon := NewDaemon()
	tickingDuration := time.Second * 19
	qBeat, err := daemon.Ch.QueueDeclare(
		"beatQueue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to open a queue. Error: %s", err)
	}
	bytes, err := messages.ToBytes[messages.Beat](messages.Beat{Id: daemon.Id})
	if err != nil {
		log.Println("cant turn beat into bytes")
		return
	}

	ticker := time.NewTicker(tickingDuration)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				err = daemon.Ch.Publish(
					"",
					qBeat.Name,
					false,
					false,
					amqp.Publishing{ContentType: "application/json", Body: bytes},
				)
				if err != nil {
					log.Println("cant send the beat")
					return
				}
				log.Println("successfully sent beat")
			}
		}
	}()

	q, err := daemon.Ch.QueueDeclare(
		"tasksQueue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Println("cant declare the queue", err.Error())
		return
	}
	messagesConsumed, err := daemon.Ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)

	if err != nil {
		log.Fatalf("failed to register a consumer. Error: %s", err)
	}
	var forever chan struct{}

	go func() {
		for message := range messagesConsumed {
			log.Printf("received a message: %s", message.Body)
			msg, err := messages.FromBytes[messages.Task](message.Body)
			if err != nil {
				log.Println("cant convert bytes to message")
				continue
			}
			v, err := govaluate.NewEvaluableExpression(msg.Expression)
			if err != nil {
				log.Println("cant make new evaluable expression", msg.Expression)
				continue
			}
			res, err := v.Evaluate(nil)
			if err != nil {
				log.Println("cant evaluate the expression")
				continue
			}
			totalSleep := time.Duration(strings.Count(msg.Expression, "+"))*msg.Durations["plus"] +
				time.Duration(strings.Count(msg.Expression, "-"))*msg.Durations["minus"] +
				time.Duration(strings.Count(msg.Expression, "*"))*msg.Durations["mul"] +
				time.Duration(strings.Count(msg.Expression, "/"))*msg.Durations["div"]
			time.Sleep(totalSleep)
			resultFloat32, ok := res.(float64)
			if !ok {
				fmt.Println("Error: result is not a float64")
				continue
			}
			resultFloat32Converted := float32(resultFloat32)

			resultMessage := messages.Result{
				Id:  msg.Id,
				Res: resultFloat32Converted,
			}
			// TODO: send the result
			qRes, err := daemon.Ch.QueueDeclare(
				"resQueue",
				false,
				false,
				false,
				false,
				nil,
			)
			bytes, err := messages.ToBytes[messages.Result](resultMessage)
			if err != nil {
				log.Println("cant turn res into bytes")
				return
			}
			err = daemon.Ch.Publish(
				"",
				qRes.Name,
				false,
				false,
				amqp.Publishing{ContentType: "application/json", Body: bytes},
			)
			if err != nil {
				log.Println("cant send the res")
				return
			}
			log.Println("successfully sent res")
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
