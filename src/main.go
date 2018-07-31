package main

import (
	// "github.com/davecgh/go-spew/spew"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sync"
	"time"
)

// GpsPing defines one gps data from one bus
type GpsPing struct {
	Timestamp string  `json:"dt_tracker"`
	Lat       float64 `json:",string"`
	Lng       float64 `json:",string"`
	Angle     float64 `json:",string"`
	Speed     float64 `json:",string"`
	IDGps     int     `json:"gps,string"`
	LineaID   int     `json:"linea_id,string"`
	Interno   string  `json:"interno"`
}

// Response from gpsbahia server
type Response struct {
	Status string    `json:"status"`
	Data   []GpsPing `json:"data"`
}

var hash = ""
var gpsBufferMapping = NewGpsBufferMapping()

func main() {
	InitDB()
	populateIDMapping()
	SearchTest()
	//go crawl()
	crawl()
	// run forever
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}

// scraps the hash needed for gps updates for provider website
func getHash() {
	for {
		response, err := http.Get("https://www.gpsbahia.com.ar")
		if err != nil {
			fmt.Print("ERROR1! ")
			fmt.Println(err)
			return
		}
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Print("ERROR2! ")
			fmt.Println(err)
			return
		}
		html := string(body)
		r := regexp.MustCompile(`(?m)hash2 = "(.*)"`)
		match := r.FindStringSubmatch(html)

		if match != nil {
			hash = match[1]
			fmt.Print("got new hash: ")
			fmt.Println(hash)
		}
		response.Body.Close()
		time.Sleep(60 * time.Second)
	}
}

func crawlOne(url string) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	var response Response
	body, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		fmt.Println(err2)
	}
	json.Unmarshal(body, &response)
	for _, gps := range response.Data {

		recorridoID, shouldSave := gpsBufferMapping.update(gps)

		if shouldSave {
			SaveGpsToDb(gps, recorridoID)
			SendToPub(gps, recorridoID)
		}
	}
}

func crawl() {
	go getHash()
	for {
		baseURL := "https://www.gpsbahia.com.ar/web/get_track_data"
		lineas := []int{1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 30, 31}
		// lineas := []int{7}
		
		if hash != "" {
			for _, lineaID := range lineas {
				url := fmt.Sprintf("%s/%d/%s", baseURL, lineaID, hash)
				// go crawlOne(url)
				crawlOne(url)
			}
		}
		time.Sleep(5 * time.Second)
	}
}
