package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/paulmach/go.geo"
	// "strconv"
	// "strings"
)

func testProject() {
	p1 := geo.NewPoint(0, 0)
	p2 := geo.NewPoint(1, 0)
	p3 := geo.NewPoint(0.5, 1)
	l := geo.NewLine(p1, p2)
	proj := l.Project(p3)
	fmt.Println(proj)
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
var connStr = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("POSTGRES_DB"))
var db, err = gorm.Open("postgres", connStr)
var mutex = &sync.Mutex{}

var pingsTable = make(map[int][]GpsPing)
// get ids from db and save into an array
func getRecorridoIDs() {
	type Mapping struct {
		providerLineaID       int
		cualbondiLineaSlug    string
		cualbondiRecorridoIDs []int
	}

	idMapping := []Mapping{
		{1, "509", []int{}},
		{4, "500", []int{}},
		{5, "502", []int{}},
		{6, "503", []int{}},
		{7, "504", []int{}},
		{8, "505", []int{}},
		{9, "506", []int{}},
		{10, "507", []int{}},
		{11, "512", []int{}},
		{12, "513", []int{}},
		{14, "514", []int{}},
		{16, "517", []int{}},
		{17, "518", []int{}},
		{18, "519", []int{}},
		{19, "519A", []int{}},
		//13, 513 EX     no
		//15, 516        no
		//30: 520        no
		//31: 504 EX     no
	}
	var slugs = []string{}
	for _, item := range idMapping {
		slugs = append(slugs, item.cualbondiLineaSlug)
	}
	// stmt, err := db.Prepare("SELECT set_config('log_statement', 'all', true);")
	// rows, err := stmt.Query()
	// defer rows.Close()
	query := `
		SELECT
			re.id as id,
			li.slug as slug
		FROM core_recorrido re
			JOIN core_linea li on (re.linea_id = li.id)
			JOIN catastro_ciudad_lineas ccl on (ccl.linea_id = li.id)
			JOIN catastro_ciudad ci on (ccl.ciudad_id = ci.id)
		WHERE
			ci.slug = ?
			AND li.slug in (?)
	`

	rows, err := db.Raw(query, "bahia-blanca", slugs).Rows()
	defer rows.Close()

	if err != nil {
		log.Println(query)
		log.Fatal(err)
	}

	for rows.Next() {
		var (
			id   int
			slug string
		)
		if err := rows.Scan(&id, &slug); err != nil {
			panic(err)
		}
		fmt.Printf("got: %d - %s\n", id, slug)
	}
}

func main() {
	var wg sync.WaitGroup

	// can spawn any number of goroutines in parallel, main program will never end

	// getRecorridoIDs()
	
	go crawl()
	go getHash()

	// sleep forever
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
		time.Sleep(5*time.Second)
	}
}

// rid, gpsant
// get id from a gpsping, use previous gps value
func getRecorridoID(gps GpsPing) string {
	return "1"
}

func saveGpsToDb(gps GpsPing){
	return
}

func saveGpsToMap(gps GpsPing){
	mutex.Lock()
	pingsTable[gps.IDGps] = append(pingsTable[gps.IDGps], gps)
	mutex.Unlock()
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
		go saveGpsToDb(gps)
		go saveGpsToMap(gps)
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
		time.Sleep(5*time.Second)
	}
}
