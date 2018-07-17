package main

import (
	"github.com/paulmach/go.geo"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"fmt"
	"time"
	"sync"
	"net/http"
	"regexp"
	"io/ioutil"
	"encoding/json"
	"os"
	"log"
	// "strconv"
	// "strings"
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
var connStr = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("POSTGRES_DB"))
var db, err = gorm.Open("postgres", connStr)
// get ids from db and save into an array
func getRecorridoIDs() {
	type Mapping struct {
		providerLineaID int
		cualbondiLineaSlug string
		cualbondiRecorridoIDs []int
	}

	idMapping := []Mapping{
		{1, "509", []int{}},
		{4, "500", []int{}},
		{5,  "502", []int{}},
		{6,  "503", []int{}},
		{7,  "504", []int{}},
		{8,  "505", []int{}},
		{9,  "506", []int{}},
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
			re.id as id
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
	
	// no entra nunca a este while
	for rows.Next() {
		var (
			id int
        )
        if err := rows.Scan(&id); err != nil {
			panic(err)
        }
        fmt.Printf("got: %d\n", id)
	}
}

func main(){
	var wg sync.WaitGroup

	// can spawn any number of goroutines in parallel, main program will never end
	getRecorridoIDs()
	// go crawl()
	// go getHash()
	// sleep forever
	wg.Add(1)
	wg.Wait()
}

func getHash(){
	defer time.AfterFunc(time.Duration(5)*time.Second, getHash)

	response, err := http.Get("https://www.gpsbahia.com.ar")
	if err != nil {
		fmt.Print("ERROR1! ")
		fmt.Println(err)
		return
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
// get id from a gpsping, use previous gps value
func getRecorridoID(gps GpsPing) string {
	return "1"
}

func crawlOne(url string){
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
		fmt.Println("recorridoID", recorridoID)
	}
}

func crawl(){
	delay := time.Duration(5)
	defer time.AfterFunc(delay*time.Second, crawl)

	baseURL := "https://www.gpsbahia.com.ar/web/get_track_data"
	lineas := []int{1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 30, 31}

	if hash != "" {
		for _, lineaID := range lineas {
			url := fmt.Sprintf("%s/%d/%s", baseURL, lineaID, hash)
			go crawlOne(url)
		}
	}
}
