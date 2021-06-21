package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	co := mqtt.NewClientOptions()
	co.AddBroker("tcp://ezmeral52gateway1.demo.local:10003")

	client := mqtt.NewClient(co)
	<-client.Connect().Done()

	storage := map[string]bool{}

	<-client.Subscribe("sensors/#", 0, func(cl mqtt.Client, me mqtt.Message) {
		messageHandler(cl, me, storage)
	}).Done()

	http.HandleFunc("/availability", func(w http.ResponseWriter, r *http.Request) {
		availabilityHandler(w, r, storage)
	})

	http.HandleFunc("/health", healthHandler)

	http.ListenAndServe(":8080", nil)

	log.Println(storage)
}

func messageHandler(cl mqtt.Client, me mqtt.Message, storage map[string]bool) {
	s := strings.Split(me.Topic(), "/")
	s = s[1:]

	log.Println(s[0])
	log.Println(string(me.Payload()))

	productAvailable, _ := strconv.Atoi(string(me.Payload()))

	storage[s[0]] = productAvailable > 5000
}

func availabilityHandler(w http.ResponseWriter, r *http.Request, storage map[string]bool) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	query := r.URL.Query()
	productId, present := query["productId"]
	if !present || len(productId) == 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	w.Write([]byte(strconv.FormatBool(storage[productId[0]])))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Write([]byte("OK"))
}
