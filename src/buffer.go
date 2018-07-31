package main

import (
	"sync"
	// "time"
	"github.com/paulsmith/gogeos/geos"
)

var maxPingsToBuffer = 5

// GpsBuffer for last pings
type GpsBuffer struct {
	values      []GpsPing
	results     []int
	lastUpdated string
	recorridoID int
	confidence  float64
}

func (buffer *GpsBuffer) push(gps GpsPing) {
	if !buffer.shouldPush(gps) {
		return
	}

	pings := buffer.values
	if len(pings) < maxPingsToBuffer {
		buffer.values = append(pings, gps)
	} else {
		// shift all values to the left
		buffer.values = append(pings[1:], gps)
	}
	buffer.update()
}

func (buffer *GpsBuffer) pushResult(id int) {
	if len(buffer.results) < maxPingsToBuffer {
		buffer.results = append(buffer.results, id)
	} else {
		// shift all values to the left
		buffer.results = append(buffer.results[1:], id)
	}
}

func (buffer *GpsBuffer) shouldPush(gps GpsPing) bool {
	threshold := 0.001
	gpsPrev := buffer.getLatest()
	A := geos.Must(geos.NewPoint(geos.NewCoord(gpsPrev.Lng, gpsPrev.Lat)))
	B := geos.Must(geos.NewPoint(geos.NewCoord(gps.Lng, gps.Lat)))

	distance, _ := A.Distance(B)

	return distance >= threshold

}

func (buffer *GpsBuffer) update() {
	last := buffer.values[len(buffer.values)-1]
	prev := buffer.values[len(buffer.values)-2]
	A := geos.Must(geos.NewPoint(geos.NewCoord(prev.Lng, prev.Lat)))
	B := geos.Must(geos.NewPoint(geos.NewCoord(last.Lng, last.Lat)))
	// search directions
	searchResult := SearchDirection(prev.LineaID, A, B)

	// update confidence
	resultID := 0
	if len(searchResult) == 1 {
		resultID = searchResult[0].ID
	}
	buffer.pushResult(resultID)
	buffer.updateRecorrido()
}

func (buffer *GpsBuffer) updateRecorrido() {
	m := make(map[int]int)
	for _, result := range buffer.results {
		_, ok := m[result]
		if !ok {
			m[result] = 1
		} else {
			m[result]++
		}
	}

	idmax := 0
	max := 0
	for idx, result := range m {
		if result >= max {
			idmax = idx
			max = result
		}
	}

	buffer.confidence = float64(max) / float64(len(buffer.results))
	buffer.recorridoID = idmax
}

func (buffer *GpsBuffer) getLatest() GpsPing {
	return buffer.values[len(buffer.values)-1]
}

func NewGpsBuffer(gps GpsPing) GpsBuffer {
	return GpsBuffer{[]GpsPing{gps}, []int{0}, gps.Timestamp, 0, 0}
}

// GpsBufferMapping is a map from ids to buffers
type GpsBufferMapping struct {
	m     map[int]GpsBuffer
	mutex sync.Mutex
}

func (mapping *GpsBufferMapping) update(gps GpsPing) (int, bool) {
	mapping.mutex.Lock()
	defer mapping.mutex.Unlock()

	buffer, ok := mapping.m[gps.IDGps]
	if !ok {
		// initialize buffer
		mapping.m[gps.IDGps] = NewGpsBuffer(gps)
		return 0, false
	}

	if buffer.lastUpdated == gps.Timestamp {
		return 0, false
	}

	buffer.lastUpdated = gps.Timestamp
	buffer.push(gps)

	if len(buffer.values) < 2 {
		return buffer.recorridoID, false
	}
	return buffer.recorridoID, true
}

// NewGpsBufferMapping is the GpsBufferMapping constructor
func NewGpsBufferMapping() GpsBufferMapping {
	return GpsBufferMapping{make(map[int]GpsBuffer), sync.Mutex{}}
}
