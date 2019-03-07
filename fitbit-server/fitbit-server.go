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
	"time"
)

var params authParams
type authParams struct {
	client_id     string
	client_secret string
}

var accessTokenInfo ResponseInfo
type ResponseInfo struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	UserId       string `json:"user_id"`
}

type Heart struct {
	ActivitiesHeartIntraday ActivityHeartIntraday	`json:"activities-heart-intraday"`
}

type ActivityHeartIntraday struct {
	Dataset  []HeartIntradayDatapoint `json:"dataset"`
	Interval int                      `json:"datasetInterval"`
	Type     string                   `json:"datasetType"`
}

type HeartIntradayDatapoint struct {
	Time      string    `json:"time"`
	HeartRate int `json:"value"`
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

	fmt.Println("authOnSuccess() says: " + resp.Status)

	err = json.NewDecoder(resp.Body).Decode(&accessTokenInfo)
	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Fprintf(w, "successfully registered!")
	}
}

func getHeartRateData() Heart {
    startTime, endTime := getTime()
	reqURL := "https://api.fitbit.com/1/user/" + accessTokenInfo.UserId + "/activities/heart/date/today/1d/1sec/time/" + startTime + "/" + endTime + ".json"

    getReq, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		fmt.Println("some get reqq err")
	}

    getReq.Header.Set("Authorization", "Bearer "+accessTokenInfo.AccessToken)
	fmt.Println(getReq)
	client := &http.Client{}
	getResp, err := client.Do(getReq)
	if err != nil {
		fmt.Println("some client do get req error")
	}

    fmt.Println("hetHeartRateData() says: " + getResp.Status)
	var heartRateData Heart
	err = json.NewDecoder(getResp.Body).Decode(&heartRateData)
	return heartRateData
}

func getTime() (string, string) {
	loc, _ := time.LoadLocation("Europe/Stockholm")
	startTime := time.Now().Add(time.Duration(-18) * time.Minute).In(loc).Format("15:04")
	endTime := time.Now().Add(time.Duration(-17) * time.Minute).In(loc).Format("15:04")
	return startTime, endTime
}

func isSleeping(w http.ResponseWriter, r *http.Request) {
	refreshToken()

	heartRate := getHeartRateData()
    lowerBpm, upperBpm := analyzeHeartData(heartRate)

	if lowerBpm == 999 {
		fmt.Fprintf(w, "CRUCIAL: WE HAVE NO DATA!")
        w.WriteHeader(204)
	} else if  upperBpm - lowerBpm <= 20 && upperBpm <= 70 { // Test case (original value might be 10 & 50)
		w.WriteHeader(418) // Person is sleeping
	} else {
		w.WriteHeader(200) // Person is awake
	}
}

func analyzeHeartData(heartRateData Heart) (int, int) {
	lower := 999
	upper := 0
	for _, dataPoint := range heartRateData.ActivitiesHeartIntraday.Dataset {
		if lower > dataPoint.HeartRate {
			lower = dataPoint.HeartRate
		}
		if upper < dataPoint.HeartRate {
			upper = dataPoint.HeartRate
		}
	}
	return lower, upper
}

func refreshToken() {
	client := &http.Client{}
	v := url.Values{}
	v.Add("grant_type", "refresh_token")
	v.Add("refresh_token", accessTokenInfo.RefreshToken)

    req, err := http.NewRequest("POST", "https://api.fitbit.com/oauth2/token", strings.NewReader(v.Encode()))
	if err != nil {
		fmt.Println("Unable to create request.")
	}

	req.Header.Set("Authorization", "Basic "+concAuth(params.client_id, params.client_secret))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		fmt.Println("some client Do err")
	}

	fmt.Println("refreshToken() says: " + resp.Status)
	err = json.NewDecoder(resp.Body).Decode(&accessTokenInfo)
	if err != nil {
		fmt.Println("error:", err)
	}
}

func main() {
	params = authParams{os.Args[1], os.Args[2]}

	http.HandleFunc("/", welcomeMessage)
	http.HandleFunc("/register/", register)
	http.HandleFunc("/success/", authOnSuccess)
	http.HandleFunc("/issleeping/", isSleeping)

	err := http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/phewstoc.sladic.se/fullchain.pem", "/etc/letsencrypt/live/phewstoc.sladic.se/privkey.pem", nil)
	if err != nil {
		log.Fatal("listenAndServe: ", err)
	}
}
