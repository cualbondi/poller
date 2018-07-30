package main

import (
	"fmt"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/paulsmith/gogeos/geos"
)

// Recorrido from cualbondi database
type Recorrido struct {
	ID        int
	Ruta      *geos.Geometry
	LineaSlug string
}

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
		var ruta []byte
		if err := rows.Scan(&recorrido.ID, &ruta, &recorrido.LineaSlug); err != nil {
			panic(err)
		}
		recorrido.Ruta = geos.Must(geos.FromWKB(ruta))
		results = append(results, recorrido)
	}
	return results
}

func getGeomArr(g *geos.Geometry) []*geos.Geometry {
	var arr []*geos.Geometry
	_type, err := g.Type()
	if err != nil {
		log.Fatal(err)
	}
	switch _type {
	case geos.LINESTRING:
		arr = append(arr, g)
	case geos.MULTILINESTRING:
		n, err := g.NGeometry()
		if err != nil {
			log.Fatal(err)
		}
		for i := 0; i < n; i++ {
			arr = append(arr, geos.Must(g.Geometry(i)))
		}
	default:
		// log.Println("unknown geometry type %v\n%v", _type, g)
	}
	return arr
}

// SolutionInternal 2 segments and possible ways to go from A to B
type SolutionInternal struct {
	Aseg  *geos.Geometry
	Bseg  *geos.Geometry
	Aproj *geos.Geometry
	Bproj *geos.Geometry
	Apos  float64
	Bpos  float64
	dist  float64
	diff  float64
}

// Search returns las rutas en *rutas* que van desde *A* hacia *B*
// TODO: para esto deberia ser facil hacer unit test!
func Search(recorridos []Recorrido, A *geos.Geometry, B *geos.Geometry) []Recorrido {
	var ret = []Recorrido{}
	var buffsize = 0.002 // alrededor de 100mts
	var Abuff = geos.Must(A.Buffer(buffsize))
	var Bbuff = geos.Must(B.Buffer(buffsize))
	for _, recorrido := range recorridos {
		var in = false
		var minlength float64 = 100000
		var Aint = getGeomArr(geos.Must(Abuff.Intersection(recorrido.Ruta)))
		var Bint = getGeomArr(geos.Must(Bbuff.Intersection(recorrido.Ruta)))
		var solutions = []SolutionInternal{}
		for _, A := range Aint {
			for _, B := range Bint {
				sol := SolutionInternal{
					Aseg:  A,
					Bseg:  B,
					Aproj: geos.Must(A.Interpolate(0.5)),
					Bproj: geos.Must(B.Interpolate(0.5)),
				}
				sol.Apos = recorrido.Ruta.Project(sol.Aproj)
				sol.Bpos = recorrido.Ruta.Project(sol.Bproj)
				sol.diff = sol.Bpos - sol.Apos
				if sol.diff > 0 {
					solutions = append(solutions, sol)
					in = true
					if sol.diff < minlength {
						minlength = sol.diff
					}
				}
			}
		}
		if in {
			ret = append(ret, recorrido)
		}
	}
	return ret
}

// SearchDirection does a geo search for the 2 recorridoIDs of LineaId between A and B
func SearchDirection(lineaID int, A *geos.Geometry, B *geos.Geometry) []Recorrido {
	var recorridos []Recorrido
	for _, m := range idMapping {
		if lineaID == m.providerLineaID {
			recorridos = m.cualbondiRecorridos
			break
		}
	}
	return Search(recorridos, A, B)
}

// SearchTest tests the search function
func SearchTest() {
	r1 := geos.Must(geos.FromWKT("LINESTRING(2 0, -2 0)"))
	r2 := geos.Must(geos.FromWKT("LINESTRING(-2 2, 4 2, 4 1.5, -2 1.5)"))
	var rutas = []Recorrido{
		Recorrido{
			ID:   1,
			Ruta: r1,
		},
		Recorrido{
			ID:   2,
			Ruta: r2,
		},
	}
	A := geos.Must(geos.FromWKT("POINT(0 1)"))
	B := geos.Must(geos.FromWKT("POINT(1 1)"))
	ret := Search(rutas, A, B)
	spew.Dump(ret)
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
