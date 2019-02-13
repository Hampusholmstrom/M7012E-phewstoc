package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

var params authParams

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
	var registerUrl = "https://www.fitbit.com/oauth2/authorize?response_type=code&client_id=" + params.client_id +
		"&redirect_uri=phewstoc.sladic.se/success/&scope=sleep"
	http.Redirect(w, r, registerUrl, http.StatusSeeOther)
}

func success(w http.ResponseWriter, r *http.Request)  {
	fmt.Print(r)
	fmt.Fprintf(w, "Received! :)")
}

func fitbitData(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	params = authParams{os.Args[1], os.Args[2], os.Args[3], os.Args[4]}

	http.HandleFunc("/", answer)
	http.HandleFunc("/register/", register)
	http.HandleFunc("/register/success", success)
	http.HandleFunc("/subscribe/sleep/", fitbitData)

	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("listenAndServe: ", err)
	}
}
