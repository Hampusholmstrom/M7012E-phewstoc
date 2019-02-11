package main

import (
	"fmt"
	"net/http"
	"log"
)

func answer(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)
	fmt.Fprintf(w, "AWAKE")
}

func fitbitData(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)		
}

func main() {
	http.HandleFunc("/", answer)
	http.HandleFunc("/subscribe/sleep/", fitbitData)
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("listenAndServe: ", err)
	}
}
