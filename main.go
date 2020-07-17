package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"flag"
)

const loginUri = "index.html"
const statsUri = "statsifcwanber.html"

var host string
var username string
var password string

func init() {
	flag.StringVar(&host, "host", "", "DSL modem host")
	flag.StringVar(&username, "host", "", "DSL modem username")
	flag.StringVar(&password, "host", "", "DSL modem password")
}

func main() {
	cj, _ := cookiejar.New(nil)

	client := &http.Client{
		Jar: cj,
	}

	loginUrl := fmt.Sprintf("http://%s/%s", host, loginUri)
	//statsUrl := fmt.Sprintf("http://%s/%s", host, statsUri)

	resp, err := client.PostForm(loginUrl, url.Values{
		"username":     {username},
		"password":     {password},
		"validateCode": {""},
		"loginfo":      {"on"},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Println(resp.Status)
}
