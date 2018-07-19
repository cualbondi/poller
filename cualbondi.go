package main

import (
	"fmt"
	"log"

	geo "github.com/paulmach/go.geo"
)

// Recorrido from cualbondi database
type Recorrido struct {
	ID        int
	Ruta      *geo.Path
	LineaSlug string
}

// GetRecorridos get ids from db and save into an array
func GetRecorridos(ciudadSlug string, lineaSlugs []string) []Recorrido {
	query := `
		SELECT
			re.id as rid,
			ST_AsBinary(re.ruta) as rruta,
			li.slug as lslug
		FROM core_recorrido re
			JOIN core_linea li on (re.linea_id = li.id)
			JOIN catastro_ciudad_lineas ccl on (ccl.linea_id = li.id)
			JOIN catastro_ciudad ci on (ccl.ciudad_id = ci.id)
		WHERE
			ci.slug = ?
			AND li.slug in (?)
	`

	rows, err := db.Raw(query, ciudadSlug, lineaSlugs).Rows()
	defer rows.Close()

	if err != nil {
		log.Println(query)
		log.Fatal(err)
	}

	var results []Recorrido
	for rows.Next() {
		var recorrido Recorrido
		if err := rows.Scan(&recorrido.ID, &recorrido.Ruta, &recorrido.LineaSlug); err != nil {
			panic(err)
		}
		results = append(results, recorrido)
	}
	return results
}

// Search returns las rutas en *rutas* que van desde *A* hacia *B*
// TODO: para esto deberia ser facil hacer unit test!
func Search(rutas []*geo.Path, A *geo.Point, B *geo.Point) []*geo.Path {
	return []*geo.Path{}
}

func testProject() {
	p1 := geo.NewPoint(0, 0)
	p2 := geo.NewPoint(1, 0)
	p3 := geo.NewPoint(0.5, 1)
	l := geo.NewLine(p1, p2)
	proj := l.Project(p3)
	fmt.Println(proj)
}
