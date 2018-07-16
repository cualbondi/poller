package main

import "github.com/paulmach/go.geo"
import (
	"fmt"
	"time"
	"sync"
	// "strconv"
	"net/http"
	"regexp"
	"io/ioutil"
	"encoding/json"
)

func testProject(){
	p1 := geo.NewPoint(0, 0)
	p2 := geo.NewPoint(1, 0)
	p3 := geo.NewPoint(0.5, 1)
	l := geo.NewLine(p1, p2)
	proj := l.Project(p3)
	fmt.Println(proj)
}

var hash = ""
var recorridoIDs []int

func getRecorridoIDs() {
	
}

func main(){
	var wg sync.WaitGroup

	// can spawn any number of goroutines in parallel, main program will never end
	getRecorridoIDs()
	go crawl()
	go getHash()
	// sleep forever
	wg.Add(1)
	wg.Wait()
}

func getHash(){
	defer time.AfterFunc(time.Duration(5)*time.Second, getHash)

	response, err := http.Get("https://www.gpsbahia.com.ar")
	if err != nil {
		// handle error
		fmt.Print("ERROR1! ")
		fmt.Println(err)
		return // will retry after 5 seconds because of defer
	}
	defer response.Body.Close()
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

}

// GpsPing defines one gps data from one bus
type GpsPing struct {
	Timestamp string `json:"dt_tracker"`
	Lat float64 `json:",string"`
	Lng float64 `json:",string"`
	Angle float64 `json:",string"`
	Speed float64 `json:",string"`
	IDGps int `json:"gps,string"`
	LineaID int `json:"linea_id,string"`
	Interno string `json:"interno"`
}

// Response from gpsbahia server
type Response struct {
	Status string `json:"status"`
	Data []GpsPing `json:"data"`
}

// rid, gpsant
// rid, gpsant

func getRecorridoID(gps GpsPing) string {
	return "1"
}

func crawlOne(url string){
	// fmt.Println(url)

	resp, err := http.Get(url)
	if err != nil {
		// handle error
	}
	defer resp.Body.Close()

	var response Response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
	}
	json.Unmarshal(body, &response)
	for _, gps := range response.Data {
		recorridoID := getRecorridoID(gps)
		fmt.Println(recorridoID)
	}
}

func crawl(){
	delay := time.Duration(5)
	defer time.AfterFunc(delay*time.Second, crawl)

	baseURL := "https://www.gpsbahia.com.ar/web/get_track_data"
	lineas := []int{1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 30, 31}

	// 1  509
	// 2
	// 3  319
	// 4  500
	// 5  502
	// 6  503
	// 7  504
	// 8  505
	// 9  506
	// 10 507
	// 11 512
	// 12 513
	// 13 513 EX     no
	// 14 514
	// 15 516        no
	// 16 517
	// 17 518
	// 18 519
	// 19 519 A
	// 30 520        no
	// 31 504 EX     no

	if hash != "" {
		for _, lineaID := range lineas {
			url := fmt.Sprintf("%s/%d/%s", baseURL, lineaID, hash)
			go crawlOne(url)
		}
	}
}
