package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	// "io/ioutil"
	"os"
	"strings"
	// "github.com/satori/go.uuid"
)

var params authParams

type authParams struct {
	client_id     string
	client_secret string
	response_type string
	scope         string
	redirect_uri  string
}

func answer(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)
	fmt.Fprintf(w, "AWAKE")
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
	ActivitiesHeart         []ActivityHeart         `json:"activities-heart"`
	ActivitiesHeartIntraday []ActivityHeartIntraday `json:"activities-heart-intraday"`
}

type ActivityHeart struct {
	DateTime string             `json:"dateTime"`
	Value    ActivityHeartValue `json:"value"`
}

type ActivityHeartValue struct {
	CustomHeartRateZones []CustomHeartRateZone `json:"customHeartRateZones"`
	HeartRateZones       []HeartRateZone       `json:"heartRateZones"`
	RestingHeartRate     HeartRate             `json:"restingHeartRate"`
}

type CustomHeartRateZone struct {
	// TODO
}

type HeartRateZone struct {
	CaloriesOut Calorie   `json:"caloriesOut"`
	Max         HeartRate `json:"max"`
	Min         HeartRate `json:"min"`
	Minutes     HeartRate `json:"minutes"`
	Name        string    `json:"name"`
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

type Sleep struct {
	Sleep []SleepDatapoint `json:"sleep"`
}

type SleepDatapoint struct {
	Data                string      `json:"dateOfSleep"`
	Duration            int         `json:"duration"`
	Efficiency          int         `json:"efficiency"`
	IsMainSleep         bool        `json:"isMainSleep"`
	Levels              SleepLevels `json:"levels"`
	LogId               int         `json:"logId"`
	MinutesAfterWakeup  int         `json:"minutesAfterWakeup"`
	MinutesAsleep       int         `json:"minutesAsleep"`
	MinutesAwake        int         `json:"minutesAwake"`
	MinutesToFallAsleep int         `json:"minutesToFallAsleep"`
	StartTime           string      `json:"startTime"`
	TimeInBed           int         `json:"timeInBed"`
	Type                string      `json:"type"`
}

type SleepLevels struct {
	Summary   SleepLevelsSummary     `json:"summary"`
	Data      []SleepLevelsDatapoint `json:"data"`
	ShortData []SleepLevelsDatapoint `json:"shortData"`
}

type SleepLevelsSummary struct {
	Deep  SleepLevel `json:"deep"`
	Light SleepLevel `json:"light"`
	Rem   SleepLevel `json:"rem"`
	Wake  SleepLevel `json:"wake"`
}
type SleepLevel struct {
	Count           int `json:"count"`
	Minutes         int `json:"minutes"`
	AvgMinutes30Day int `json:"thirtyDayAvgMinutes"`
}

type SleepLevelsDatapoint struct {
	Datetime string `json:"datetime"`
	Level    string `json:"level"`
	Duration int    `json:"seconds"`
}

func success(w http.ResponseWriter, r *http.Request) {
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

	reqURL := "https://api.fitbit.com/1.2/user/" + accessTokenInfo.UserId + "/sleep/date/2019-02-22.json"
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

	var sleepLog Sleep
	//	body, err := ioutil.ReadAll(getResp.Body)
	//	fmt.Println(string(body))
	err = json.NewDecoder(getResp.Body).Decode(&sleepLog)
	fmt.Fprint(w, sleepLog)
	// fmt.Println(sleepLog.Sleep[0].Data)
	// fmt.Fprint(w, getResp)
	// fmt.Println(resp)
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
