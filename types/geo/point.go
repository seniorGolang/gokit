package geo

import (
	"errors"
	"math"

	"github.com/globalsign/mgo/bson"
)

const (
	pointType = "Point"
)

var (
	ErrGeoType   = errors.New("wrong geometry type")
	ErrGeoCoords = errors.New("coordinates has wrong format for point type")
)

type Point struct {
	// @gkg example=43.116418 desc="latitude"
	Lat float64 `json:"lat" fake:"latitude"`
	// @gkg example=131.882475 desc="longitude"
	Lon float64 `json:"lon" fake:"longitude"`
}

func (point *Point) GetReduced(precision float64) *Point {
	return &Point{
		Lat: math.Round(point.Lat*precision) / precision,
		Lon: math.Round(point.Lon*precision) / precision,
	}
}

func (point *Point) GetBSON() (interface{}, error) {

	value := struct {
		Type        string    `bson:"type"`
		Coordinates []float64 `bson:"coordinates"`
	}{
		Type:        pointType,
		Coordinates: []float64{0, 0},
	}

	if point != nil {
		value.Coordinates = []float64{point.Lon, point.Lat}
	}

	return value, nil
}

func (point *Point) SetBSON(raw bson.Raw) (err error) {

	decodedType := new(struct {
		Type string `bson:"type"`
	})

	if err = raw.Unmarshal(decodedType); err != nil {
		return
	}

	if decodedType.Type != pointType {
		return ErrGeoType
	}

	decodedGeo := new(struct {
		Coordinates []float64 `bson:"coordinates"`
	})

	if err = raw.Unmarshal(decodedGeo); err != nil {
		return
	}

	if len(decodedGeo.Coordinates) != 2 {
		return ErrGeoCoords
	}

	point.Lon = decodedGeo.Coordinates[0]
	point.Lat = decodedGeo.Coordinates[1]
	return
}
