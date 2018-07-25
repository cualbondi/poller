package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/paulsmith/gogeos/geos"
	// "strconv"
	// "strings"
)

// Mapping between gpsbahia id and cualbondi recorrido ids
type Mapping struct {
	providerLineaID     int
	cualbondiLineaSlug  string
	cualbondiRecorridos []Recorrido
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
	// TODO: tal vez aca agregar otra condicion para que solamente appendee el nuevo punto si esta a mas de x cantidad de metros
	// o agrandar el maxPingsToBuffer a no se, 500, y modificar la funcion getLatest para que reciba un 2 argumentos mas _distancia_ y _point_, que devuelva el latest que esta a mas que _distancia_ de _point_
	if len(pings) < maxPingsToBuffer {
		buffer.m[gps.IDGps] = append(pings, gps)
		return
	}
	for i := 1; i < maxPingsToBuffer; i++ {
		pings[i-1] = pings[i]
	}
	pings[maxPingsToBuffer-1] = gps
}

func (buffer *GpsBuffer) getLatest(id int) GpsPing {
	value := buffer.m[id]
	return value[len(value)-1]
}

var hash = ""
var recorridoIDs []int
var gpsBuffer = GpsBuffer{make(map[int][]GpsPing), sync.Mutex{}}

func main() {
	InitDB()

	populateIdMapping()

	SearchTest()

	//go crawl()
	crawl()

	// run forever
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}

// populate a map that liks provider ids with cb data
func populateIdMapping(){
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
			var A = geos.Must(geos.NewPoint(geos.NewCoord(gpsPrev.Lat, gpsPrev.Lng)))
			var B = geos.Must(geos.NewPoint(geos.NewCoord(gps.Lat, gps.Lng)))
			recorridos = Search(recorridos, A, B)
			if len(recorridos) > 0 {
				recorridoID = recorridos[0].ID
			}
		}
		gpsBuffer.push(gps)
		if recorridoID == 0 {
			// fmt.Println("no match found")
			continue
		}
		fmt.Println(recorridoID)
		SaveGpsToDb(gps, recorridoID)
		SendToPub(gps, recorridoID)
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
