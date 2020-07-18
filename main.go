package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/PennState/subexp"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const loginUri = "index.html"
const statsUri = "statsifcwanber.html"

func main() {
	host := flag.String("host", "", "DSL modem host")
	username := flag.String("username", "", "DSL modem username")
	password := flag.String("password", "", "DSL modem password")
	flag.Parse()

	numstats := []string{"Syncs", "Uptime", "SNRDS", "SNRUS", "AttDS", "AttUS", "PwrDS", "PwrUS", "AttainableDS", "AttainableUS", "RateDS", "RateUS"}
	statsRe := regexp.MustCompile(`(?s).*?>Synchronized Time:</td><td colspan="2">(?P<Uptime>.+?)\&nbsp;</td>`+
		`.*?Synchronizations:</td><td colspan="2">(?P<Syncs>\d+)` +
		`.*?>SNR Margin \(0\.1 dB\):</td><td>(?P<SNRDS>.+?)</td><td>(?P<SNRUS>.+?)</td>` +
		`.*?>Attenuation \(0\.1 dB\):</td><td>(?P<AttDS>.+?)</td><td>(?P<AttUS>.+?)</td>` +
		`.*?>Output Power \(0\.1 dBm\):</td><td>(?P<PwrDS>.+?)</td><td>(?P<PwrUS>.+?)</td>` +
		`.*?><nobreak>Attainable Rate \(Kbps\):</nobreak></td><td>(?P<AttainableDS>.+?)</td><td>(?P<AttainableUS>.+?)</td>`+
		`.*?>Rate \(Kbps\):</td><td>(?P<RateDS>.+?)</td><td>(?P<RateUS>.+?)</td>`)
	uptimeRe := regexp.MustCompile(`(?P<days>\d+) (?P<hours>\d+):(?P<minutes>\d+):(?P<seconds>\d+)`)

	// log in
	cj, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cj,
	}

	loginUrl := fmt.Sprintf("http://%s/%s", *host, loginUri)
	statsUrl := fmt.Sprintf("http://%s/%s", *host, statsUri)

	// order matters!
	data := fmt.Sprintf("username=%s&password=%s&validateCode=", *username, *password)

	req, err := http.NewRequest("POST", loginUrl, bytes.NewBufferString(data))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error when logging in: %s", err)
	}
	defer resp.Body.Close()

	// fetch stats page
	resp, err = client.Get(statsUrl)
	if err != nil {
		log.Fatalf("Error when getting stats: %s", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error when getting stats (%s): %s", resp.Status, body)
	}

	// find matches
	m := subexp.Capture(statsRe, string(body))
	if m == nil {
		log.Fatal("Could not find stats in page")
	}

	// create fieldset
	var fieldset []string
	for _, s := range numstats {
		v, err := m.FirstByName(s)
		if err != nil {
			continue
		}
		if _, err := strconv.Atoi(v); err != nil {
			continue
		}
		fieldset = append(fieldset, fmt.Sprintf("%s=%s", s, v))
	}

	// convert uptime to a time.Duration
	var uptime time.Duration
	uptimeStr, _ := m.FirstByName("Uptime")
	t := subexp.Capture(uptimeRe, uptimeStr)
	if t != nil {
		days, _ := t.FirstByName("days")
		hours, _ := t.FirstByName("hours")
		minutes, _ := t.FirstByName("minutes")
		seconds, _ := t.FirstByName("seconds")
		uptime = time.Duration(parseInt64(days)*int64(time.Hour)*24 +
			parseInt64(hours)*int64(time.Hour) +
			parseInt64(minutes)*int64(time.Minute) +
			parseInt64(seconds)*int64(time.Second))
	}

	fieldset = append(fieldset, fmt.Sprintf("uptime=%.0f", uptime.Seconds()))

	// print influxdb line protocl
	fmt.Printf("dsl,host=%s %s\n", *host, strings.Join(fieldset, ","))
}

// parsInt64 converts a string to an int64, returning 0 if empty or in the case of error
func parseInt64(value string) int64 {
	if len(value) == 0 {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return int64(parsed)
}
