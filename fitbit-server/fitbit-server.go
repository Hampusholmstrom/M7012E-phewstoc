package main

import (
	"fmt"
	"net/http"
	"log"
)

type authParams struct {
	client_id string
	response_type string
	scope string
	redirect_uri string
}

func answer(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)
	fmt.Fprintf(w, "AWAKE")
}

func register(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, registerUrl, http.StatusSeeOther)
}

func success(w http.ResponseWriter, r *http.Request)  {

}

func fitbitData(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	http.HandleFunc("/", answer)
	http.HandleFunc("/register/", register)
	http.HandleFunc("/register/success", success)
	http.HandleFunc("/subscribe/sleep/", fitbitData)
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("listenAndServe: ", err)
	}
}
