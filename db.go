package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jinzhu/gorm"
	// dialect postgres
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var db *gorm.DB

// InitDB call this initially in main
func InitDB() {
	var err error
	var connStr = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("POSTGRES_DB"))
	db, err = gorm.Open("postgres", connStr)
	if err != nil {
		log.Panic(err)
	}
}

// SaveGpsToDb guarda un punto de gps en la base de datos
func SaveGpsToDb(gps GpsPing, recorrido_id int) {
	query := `
		INSERT INTO gps (timestamp, lat, lng, id_gps, speed, angle, linea_id, interno, reecorrido_id) VALUES (?)
	`
	db.Exec(query, []interface{}{
		gps.Timestamp,
		gps.Lat,
		gps.Lng,
		gps.IDGps,
		gps.Speed,
		gps.Angle,
		gps.LineaID,
		gps.Interno,
		recorrido_id,
	})
}
