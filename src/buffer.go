package main 

import (
	"sync"
)

var maxPingsToBuffer = 2

// GpsBuffer for last pings
type GpsBuffer struct {
	m     map[int][]GpsPing
	mutex sync.Mutex
}

func (buffer *GpsBuffer) push(gps GpsPing) bool {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	pings, ok := buffer.m[gps.IDGps]
	if !ok {
		buffer.m[gps.IDGps] = []GpsPing{gps}
		return true
	}
	// TODO: tal vez aca agregar otra condicion para que solamente appendee el nuevo punto si esta a mas de x cantidad de metros
	// o agrandar el maxPingsToBuffer a no se, 500, y modificar la funcion getLatest para que reciba un 2 argumentos mas _distancia_ y _point_, que devuelva el latest que esta a mas que _distancia_ de _point_

	// ignorar el punto de gps si ya existe uno previo con el mismo timestamp
	last := pings[len(pings)-1]
	if last.Timestamp == gps.Timestamp {
		return false
	}


	if len(pings) < maxPingsToBuffer {
		buffer.m[gps.IDGps] = append(pings, gps)
		return true
	}

	// shift all values to the left
	for i := 1; i < maxPingsToBuffer; i++ {
		pings[i-1] = pings[i]
	}
	pings[maxPingsToBuffer-1] = gps
	return true
}

func (buffer *GpsBuffer) getLatest(id int) GpsPing {
	value := buffer.m[id]
	return value[len(value)-1]
}
