package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var params authParams

type authParams struct {
	client_id     string
	client_secret string
	response_type string
	scope         string
	redirect_uri  string
}

func welcomeMessage(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)
	fmt.Fprintf(w, "Welcome to the Phewstoc fitbit companion app!")
}

func register(w http.ResponseWriter, r *http.Request) {
	var registerUrl = "https://www.fitbit.com/oauth2/authorize?response_type=code&client_id=" + params.client_id + "&scope=heartrate"
	http.Redirect(w, r, registerUrl, http.StatusSeeOther)
}

func concAuth(clientId string, clientSecret string) string {
	idAndSecret := clientId + ":" + clientSecret
	return b64.StdEncoding.EncodeToString([]byte(idAndSecret))
}

type HeartRate int64
type Calorie float64

type Heart struct {
	ActivitiesHeartIntraday ActivityHeartIntraday `json:"activities-heart-intraday"`
}

type ActivityHeartIntraday struct {
	Dataset  []HeartIntradayDatapoint `json:"dataset"`
	Interval int                      `json:"datasetInterval"`
	Type     string                   `json:"datasetType"`
}

type HeartIntradayDatapoint struct {
	Time      string    `json:"time"`
	HeartRate HeartRate `json:"value"`
}


func authOnSuccess(w http.ResponseWriter, r *http.Request) {
	authURL := "https://api.fitbit.com/oauth2/token"

	keys, ok := r.URL.Query()["code"]
	if !ok || len(keys[0]) < 1 {
		log.Println("Url param 'code' is missing")
	}

	client := &http.Client{}

	v := url.Values{}
    v.Add("client_id", params.client_id)
	v.Add("grant_type", "authorization_code")
	v.Add("code", keys[0])

    req, err := http.NewRequest("POST", authURL, strings.NewReader(v.Encode()))
	if err != nil {
		fmt.Println("authentication error while obtaining acesstoken")
	}

	req.Header.Set("Authorization", "Basic "+concAuth(params.client_id, params.client_secret))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		fmt.Println("some client Do err")
	}

	fmt.Println(resp.Status)
	type ResponseInfo struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		UserId       string `json:"user_id"`
	}
	var accessTokenInfo ResponseInfo
	err = json.NewDecoder(resp.Body).Decode(&accessTokenInfo)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(accessTokenInfo)

    reqURL := "https://api.fitbit.com/1/user/" + accessTokenInfo.UserId + "/activities/heart/date/today/1d/1sec/time/05:00/05:01.json"
	getReq, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		fmt.Println("some get reqq err")
	}
	getReq.Header.Set("Authorization", "Bearer "+accessTokenInfo.AccessToken)
	fmt.Println(getReq)
	getResp, err := client.Do(getReq)
	if err != nil {
		fmt.Println("some client do get req error")
	}
	fmt.Println(getResp.Status)

	var heartRate Heart
	//	body, err := ioutil.ReadAll(getResp.Body)
	//	fmt.Println(string(body))
	err = json.NewDecoder(getResp.Body).Decode(&heartRate)
	fmt.Fprint(w, heartRate.ActivitiesHeartIntraday)
	// fmt.Println(sleepLog.Sleep[0].Data)
	// fmt.Fprint(w, getResp)
	// fmt.Println(resp)
}

func fitbitData(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	params = authParams{os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5]}

	http.HandleFunc("/", welcomeMessage)
	http.HandleFunc("/register/", register)
	http.HandleFunc("/success/", authOnSuccess)
	http.HandleFunc("/subscribe/sleep/", fitbitData)

	err := http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/phewstoc.sladic.se/fullchain.pem", "/etc/letsencrypt/live/phewstoc.sladic.se/privkey.pem", nil)
	if err != nil {
		log.Fatal("listenAndServe: ", err)
	}
}
