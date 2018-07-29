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
	"github.com/paulsmith/gogeos/geos"
)

// Mapping between gpsbahia id and cualbondi recorrido ids
type Mapping struct {
	providerLineaID     int
	cualbondiLineaSlug  string
	cualbondiRecorridos []Recorrido
}
var idMapping = []Mapping{
	{1, "509", []Recorrido{}},
	{3, "319", []Recorrido{}},
	{4, "500", []Recorrido{}},
	{5, "502", []Recorrido{}},
	{6, "503", []Recorrido{}},
	{7, "504", []Recorrido{}},
	{8, "505", []Recorrido{}},
	{9, "506", []Recorrido{}},
	{10, "507", []Recorrido{}},
	{11, "512", []Recorrido{}},
	{12, "513", []Recorrido{}},
	{13, "513ex", []Recorrido{}},
	{14, "514", []Recorrido{}},
	{15, "516", []Recorrido{}},
	{16, "517", []Recorrido{}},
	{17, "518", []Recorrido{}},
	{18, "519", []Recorrido{}},
	{19, "519a", []Recorrido{}},
	{30, "520", []Recorrido{}},
	{31, "504ex", []Recorrido{}},
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

var hash = ""
var recorridoIDs []int
var gpsBuffer = GpsBuffer{make(map[int][]GpsPing), sync.Mutex{}}

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

// populate a map that liks provider ids with cb data
func populateIDMapping() {
	var lineaSlugs = []string{}
	for _, item := range idMapping {
		lineaSlugs = append(lineaSlugs, item.cualbondiLineaSlug)
	}
	res := GetRecorridos("bahia-blanca", lineaSlugs)
	for _, r := range res {
		for i, m := range idMapping {
			if m.cualbondiLineaSlug == r.LineaSlug {
				idMapping[i].cualbondiRecorridos = append(m.cualbondiRecorridos, Recorrido{ID: r.ID, Ruta: r.Ruta, LineaSlug: r.LineaSlug})
			}
		}
	}

	for _, item := range idMapping {
		if len(item.cualbondiRecorridos) != 2 {
			fmt.Printf("Warning: slug %v does not have 2 recorridos \n", item.cualbondiLineaSlug)
		}
	}
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
		// TODO: handle error
	}
	defer resp.Body.Close()

	var response Response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// TODO: handle error
	}
	json.Unmarshal(body, &response)
	for _, gps := range response.Data {
		var recorridos []Recorrido
		for _, m := range idMapping {
			if gps.LineaID == m.providerLineaID {
				recorridos = m.cualbondiRecorridos
				break
			}
		}

		var gpsPrev GpsPing
		var recorridoID int
		if len(gpsBuffer.m[gps.IDGps]) > 0 {
			gpsPrev = gpsBuffer.m[gps.IDGps][len(gpsBuffer.m[gps.IDGps])-1]
			var A = geos.Must(geos.NewPoint(geos.NewCoord(gpsPrev.Lng, gpsPrev.Lat)))
			var B = geos.Must(geos.NewPoint(geos.NewCoord(gps.Lng, gps.Lat)))
			recorridos = Search(recorridos, A, B)
			if len(recorridos) > 0 {
				recorridoID = recorridos[0].ID
			}
			//logResult(gps, recorridoID, A, B, recorridos)
		}
		pushed := gpsBuffer.push(gps)
		// if recorridoID == 0 {
		// 	continue
		// }
		//fmt.Println(pushed, gps.Timestamp)
		if pushed {
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
