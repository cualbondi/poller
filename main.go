package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sync"
	"time"
	// "strconv"
	// "strings"
)

// Mapping between gpsbahia id and cualbondi recorrido ids
type Mapping struct {
	providerLineaID       int
	cualbondiLineaSlug    string
	cualbondiRecorridoIDs []Recorrido
}

var idMapping = []Mapping{
	{1, "509", []Recorrido{}},
	{4, "500", []Recorrido{}},
	{5, "502", []Recorrido{}},
	{6, "503", []Recorrido{}},
	{7, "504", []Recorrido{}},
	{8, "505", []Recorrido{}},
	{9, "506", []Recorrido{}},
	{10, "507", []Recorrido{}},
	{11, "512", []Recorrido{}},
	{12, "513", []Recorrido{}},
	{14, "514", []Recorrido{}},
	{16, "517", []Recorrido{}},
	{17, "518", []Recorrido{}},
	{18, "519", []Recorrido{}},
	{19, "519a", []Recorrido{}},
	//13, 513 EX     no
	//15, 516        no
	//30: 520        no
	//31: 504 EX     no
}

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

var maxPingsToBuffer = 2

// GpsBuffer for last pings
type GpsBuffer struct {
	m     map[int][]GpsPing
	mutex sync.Mutex
}

func (buffer *GpsBuffer) push(gps GpsPing) {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	pings, ok := buffer.m[gps.IDGps]
	if !ok {
		buffer.m[gps.IDGps] = []GpsPing{gps}
		return
	}
	if len(pings) < maxPingsToBuffer {
		buffer.m[gps.IDGps] = append(pings, gps)
		return
	}
	for i := 1; i < maxPingsToBuffer; i++ {
		pings[i-1] = pings[i]
	}
	pings[maxPingsToBuffer-1] = gps
}

var hash = ""
var recorridoIDs []int
var gpsBuffer = GpsBuffer{make(map[int][]GpsPing), sync.Mutex{}}

func main() {
	InitDB()
	var wg sync.WaitGroup

	var lineaSlugs = []string{}
	for _, item := range idMapping {
		lineaSlugs = append(lineaSlugs, item.cualbondiLineaSlug)
	}
	res := GetRecorridos("bahia-blanca", lineaSlugs)
	for _, r := range res {
		for i, m := range idMapping {
			if m.cualbondiLineaSlug == r.LineaSlug {
				idMapping[i].cualbondiRecorridoIDs = append(m.cualbondiRecorridoIDs, Recorrido{ID: r.ID, Ruta: r.Ruta, LineaSlug: r.LineaSlug})
			}
		}
	}

	//fmt.Println(idMapping)

	go crawl()
	go getHash()

	wg.Add(1)
	wg.Wait()
}

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

// rid, gpsant
// get id from a gpsping, use previous gps value
func getRecorridoID(gps GpsPing) string {
	// for _, m := range response.Data {
	// 	recorridoID := getRecorridoID(gps)
	// 	fmt.Println("recorridoID", recorridoID)
	// }
	return "1"
}

func saveGpsToMap(gps GpsPing) {
	gpsBuffer.push(gps)
}

func crawlOne(url string) {
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
		SaveGpsToDb(gps)
		saveGpsToMap(gps)
	}
}

func crawl() {
	for {
		baseURL := "https://www.gpsbahia.com.ar/web/get_track_data"
		lineas := []int{1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 30, 31}

		if hash != "" {
			for _, lineaID := range lineas {
				url := fmt.Sprintf("%s/%d/%s", baseURL, lineaID, hash)
				go crawlOne(url)
			}
		}
		time.Sleep(5 * time.Second)
	}
}
