package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
    b64 "encoding/base64"
    "net/url"
    "strings"

    // "github.com/satori/go.uuid"
)

var params authParams

type authParams struct {
	client_id string
    client_secret string
	response_type string
	scope string
	redirect_uri string
}

func answer(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)
	fmt.Fprintf(w, "AWAKE")
}

func register(w http.ResponseWriter, r *http.Request) {
	var registerUrl = "https://www.fitbit.com/oauth2/authorize?response_type=code&client_id=" + params.client_id + "&scope=sleep"
	http.Redirect(w, r, registerUrl, http.StatusSeeOther)
}

func concAuth(clientId string, clientSecret string) string {
    idAndSecret := clientId + ":" + clientSecret
    return b64.StdEncoding.EncodeToString([]byte(idAndSecret))
}

func success(w http.ResponseWriter, r *http.Request)  {
	fmt.Print(r)
	fmt.Fprintf(w, "Received! :)")

    keys, ok := r.URL.Query()["code"]
    if !ok || len(keys[0]) < 1 {
        log.Println("Url param 'code' is missing")
    }
    fmt.Print(keys)


    authURL := "https://api.fitbit.com/oauth2/token"
    v := url.Values{}
    client := &http.Client{}
    v.Add("client_id", params.client_id)
    v.Add("grant_type", "authorization_code")
    v.Add("code", keys[0])
    req, err := http.NewRequest("POST", authURL, strings.NewReader(v.Encode()))
    if err != nil {
        fmt.Println("some Post Err")
    }
    req.Header.Set("Authorization", "Basic " + concAuth(params.client_id, params.client_secret))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    resp, err := client.Do(r)
    if err != nil {
        fmt.Println("some client Do err")
    }
    fmt.Println(resp.Status)
    fmt.Println(resp.Body)
}

func fitbitData(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	params = authParams{os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5]}

	http.HandleFunc("/", answer)
	http.HandleFunc("/register/", register)
	http.HandleFunc("/success/", success)
	http.HandleFunc("/subscribe/sleep/", fitbitData)

	err := http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/phewstoc.sladic.se/fullchain.pem", "/etc/letsencrypt/live/phewstoc.sladic.se/privkey.pem", nil)
	if err != nil {
		log.Fatal("listenAndServe: ", err)
	}
}
