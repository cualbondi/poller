package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/paulsmith/gogeos/geos"

	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"

	// dialect postgres
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var db *gorm.DB
var redisClient *redis.Client

// InitDB call this initially in main
func InitDB() {
	var err error
	var connStr = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("POSTGRES_DB"))
	db, err = gorm.Open("postgres", connStr)
	if err != nil {
		log.Panic(err)
	}
	
	db.Exec(`
		CREATE TABLE IF NOT EXISTS gps (
			id bigserial not null CONSTRAINT pk PRIMARY KEY,
			timestamp timestamp,
			latlng geometry(Point, 0),
			id_gps bigint,
			speed float,
			angle float,
			recorrido_id int,
			meta text
		)
	`)

	redisClient = redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

// SaveGpsToDb guarda un punto de gps en la base de datos
func SaveGpsToDb(gps GpsPing, recorridoID int) {
	query := `
		INSERT INTO gps (timestamp, latlng, id_gps, speed, angle, recorrido_id, meta) VALUES (?)
	`
	var point, err = geos.Must(geos.NewPoint(geos.NewCoord(gps.Lat, gps.Lng))).Hex()
	if err != nil {
		log.Println("error decoding")
	}
	var meta = fmt.Sprintf("%d, %s", gps.LineaID, gps.Interno)
	var data = []interface{}{
		gps.Timestamp,
		string(point),
		gps.IDGps,
		gps.Speed,
		gps.Angle,
		recorridoID,
		meta,
	}
	db.Exec(query, data)
}

type pubmessage struct {
	RecorridoID int
	Timestamp   string
	Point		string
	Angle       float64
	Speed       float64
	IDGps       int
}

// SendToPub sends a message to the redis channel
func SendToPub(gps GpsPing, recorridoID int) {
	// send data to redis Pub/Sub
	
	var point, err = geos.Must(geos.NewPoint(geos.NewCoord(gps.Lat, gps.Lng))).ToWKT()
	if err != nil {
		log.Println("error decoding")
	}
	data, err := json.Marshal(pubmessage{
		RecorridoID: recorridoID,
		Timestamp:   gps.Timestamp,
		Point:		 point,
		Angle:       gps.Angle,
		Speed:       gps.Speed,
		IDGps:       gps.IDGps,
	})
	if err != nil {
		log.Println("json marshal", err)
	}
	_, err = redisClient.Publish("gps-<id_recorrido>", data).Result()
	if err != nil {
		log.Println("redis", err)
	}
}
