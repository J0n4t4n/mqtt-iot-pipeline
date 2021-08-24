package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	co := mqtt.NewClientOptions()
	co.AddBroker("tcp://stable201.container.demo.local:10106")
	co.SetUsername("ezmeral")
	co.SetPassword("NMH4JieRkWb!LH79KmZW6sNwWJ!E9X")

	client := mqtt.NewClient(co)
	<-client.Connect().Done()

	storage := map[string]bool{}
	mapping := map[string]string{
		"1": "weight-sensor-7",
		"2": "weight-sensor-2",
		"3": "weight-sensor-4",
		"4": "weight-sensor-6",
		"5": "weight-sensor-9",
		"6": "weight-sensor-8",
		"7": "weight-sensor-3",
		"8": "weight-sensor-5",
		"9": "weight-sensor-1",
	}

	reverse_mapping := make(map[string]string)

	for k, v := range mapping {
		reverse_mapping[v] = k
	}

	<-client.Subscribe("homie/+/integer/integer", 0, func(cl mqtt.Client, me mqtt.Message) {
		messageHandler(cl, me, storage, reverse_mapping)
	}).Done()

	http.HandleFunc("/backend/availability", func(w http.ResponseWriter, r *http.Request) {
		availabilityHandler(w, r, storage, mapping)
	})

	http.HandleFunc("/backend/recommendation", func(w http.ResponseWriter, r *http.Request) {
		recommendationHandler(w, r, storage, mapping)
	})

	http.HandleFunc("/health", healthHandler)

	http.ListenAndServe(":8080", nil)

	log.Println(storage)
}

func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func messageHandler(cl mqtt.Client, me mqtt.Message, storage map[string]bool, reverse_mapping map[string]string) {
	aims_path := "aims-portal"
	//aims_path := "localhost"

	s := strings.Split(me.Topic(), "/")
	s = s[1:]

	log.Println(s[0])
	log.Println(string(me.Payload()))

	productAvailable, _ := strconv.ParseFloat(string(me.Payload()), 64)

	storage[s[0]] = productAvailable > 300

	log.Println(storage)

	res, err := http.Get(
		fmt.Sprintf("http://%s:8000/portal/articles/%s?stationCode=HPE1", aims_path, reverse_mapping[s[0]]),
	)
	if err != nil {
		log.Fatal("Crap: ", err)
	}

	body, _ := ioutil.ReadAll(res.Body)

	ourArticle := article{}
	//log.Println("This is the body: ", string(body))
	json.Unmarshal(body, &ourArticle)
	log.Println(ourArticle)

	available := "0"
	if storage[s[0]] {
		available = "1"
	}

	article :=
		dataList{
			DataList: []article{
				{
					Id:          reverse_mapping[s[0]],
					Name:        ourArticle.Name,
					Nfc:         ourArticle.Nfc,
					StationCode: "HPE1",
					Data: articleData{
						Etc_1: available,
					},
				},
			},
		}

		//curl -X POST "http://localhost:8000/portal/articles" -H "accept: application/json" -H "Content-Type: application/json" -d "{ \"dataList\": [ { \"data\": { \"ETC_1\": \"1\" }, \"id\": \"1\", \"name\": \"Hammer\", \"stationCode\": \"HPE1\" } ]}" -vvvvv
		//curl "http://localhost:8000/portal/articles/1?stationCode=HPE1" -vvvvv

	jsonArticle, _ := json.Marshal(article)

	log.Println(string(jsonArticle))

	resp, _ := http.Post(
		fmt.Sprintf("http://%s:8000/portal/articles", aims_path),
		"application/json",
		bytes.NewBuffer(jsonArticle),
	)
	log.Println(resp)
	// Push to Aims
}

type articleData struct {
	Etc_1 string `json:"ETC_1"`
}

type article struct {
	Id          string      `json:"id"`
	Name        string      `json:"name"`
	Nfc         string      `json:"nfc"`
	StationCode string      `json:"stationCode"`
	Data        articleData `json:"data"`
}

type dataList struct {
	DataList []article `json:"dataList"`
}

func checkAvailability(productId string) bool {
	/*
		res, err := http.Get(
			fmt.Sprintf("http://localhost:8000/portal/articles/%d?stationCode=HPE1", productIdInt),
		)
		if err != nil {
			log.Fatal("Crap: ", err)
		}

		body, _ := ioutil.ReadAll(res.Body)

		ourArticle := article{}
		log.Println("This is the body: ", string(body))
		json.NewDecoder(res.Body).Decode(&ourArticle)
		log.Println(ourArticle)
	*/
	return true
}

type availabilityResponse struct {
	Available bool `json:"available"`
}

func availabilityHandler(w http.ResponseWriter, r *http.Request, storage map[string]bool, mapping map[string]string) {
	setupResponse(&w, r)
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

	log.Println("This product was requested: ", productId[0])

	avaRes := availabilityResponse{
		Available: storage[mapping[productId[0]]],
	}

	available, _ := json.Marshal(avaRes)
	w.Write(available)
}

type recommendationResponse struct {
	Recommendations []string `json:"recommendations"`
}

func recommendationHandler(w http.ResponseWriter, r *http.Request, storage map[string]bool, mapping map[string]string) {
	recommender_path := "http://recommender-deployment-svc:1234"
	//recommender_path := "http://localhost:1234"

	setupResponse(&w, r)
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

	log.Println("This product was requested: ", productId[0])

	res, err := http.Get(
		fmt.Sprintf("%s/recommendation?productId=%s", recommender_path, productId[0]),
	)
	if err != nil {
		log.Fatal("Crap: ", err)
	}

	body, _ := ioutil.ReadAll(res.Body)

	var ids []string
	json.Unmarshal([]byte(body), &ids)

	availableIds := []string{}

	log.Println("Recommended: ", ids)

	for i := range ids {
		if storage[mapping[ids[i]]] {
			availableIds = append(availableIds, ids[i])
		}
	}

	recRes := recommendationResponse{
		Recommendations: availableIds,
	}

	log.Println("Available: ", recRes)
	jsonIds, _ := json.Marshal(recRes)
	w.Write(jsonIds)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w, r)
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Write([]byte("OK"))
}
