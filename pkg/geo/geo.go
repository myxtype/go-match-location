package geo

import (
	"errors"
	"fmt"
	"github.com/myxtype/go-match-location/pkg/geohash"
	"github.com/myxtype/go-match-location/pkg/sortedset"
)

var (
	ErrNull = errors.New("ErrNull")
)

type Geo struct {
	sortedSet *sortedset.SortedSet
}

type GeoItem struct {
	Lng    float64
	Lat    float64
	Member string
}

type GeoPoint struct {
	Lng    float64
	Lat    float64
	Member string
	Dist   float64
	Score  float64
}

func NewGeo() *Geo {
	return &Geo{sortedSet: sortedset.Make()}
}

// Add a location into SortedSet
func (g *Geo) Add(items ...*GeoItem) error {
	if len(items) == 0 {
		return errors.New("ERR wrong number of arguments for 'geoadd' command")
	}

	elements := make([]*sortedset.Element, len(items))
	for i := 0; i < len(items); i++ {
		item := items[i]

		if item.Lat < -90 || item.Lat > 90 || item.Lng < -180 || item.Lng > 180 {
			return errors.New(fmt.Sprintf("ERR invalid longitude,latitude pair %v,%v", item.Lat, item.Lng))
		}

		code, err := geohash.EncodeWGS84(item.Lng, item.Lat)
		if err != nil {
			return err
		}
		elements[i] = &sortedset.Element{
			Member: item.Member,
			Score:  float64(code),
		}
	}

	i := 0
	for _, e := range elements {
		if g.sortedSet.Add(e.Member, e.Score) {
			i++
		}
	}
	return nil
}

func (g *Geo) Remove(members ...string) error {
	if len(members) == 0 {
		return errors.New("ERR wrong number of arguments for 'geoadd' command")
	}

	for i := 0; i < len(members); i++ {
		g.sortedSet.Remove(members[i])
	}

	return nil
}

// Pos returns location of a member
func (g *Geo) Pos(members ...string) ([]*GeoItem, error) {
	if len(members) == 0 {
		return nil, errors.New("ERR wrong number of arguments for 'geopos' command")
	}

	positions := make([]*GeoItem, len(members))
	for i := 0; i < len(members); i++ {
		member := members[i]
		elem, exists := g.sortedSet.Get(member)
		if !exists {
			continue
		}

		lng, lat := geohash.DecodeToLongLatWGS84(uint64(elem.Score))
		positions[i] = &GeoItem{
			Lng:    lng,
			Lat:    lat,
			Member: member,
		}
	}

	return positions, nil
}

// Dist returns the distance between two locations
func (g *Geo) Dist(members [2]string, unit string) (float64, error) {
	positions := make([][]float64, 2)

	for i := 0; i < 2; i++ {
		member := members[i]
		elem, exists := g.sortedSet.Get(member)
		if !exists {
			return 0, ErrNull
		}

		lng, lat := geohash.DecodeToLongLatWGS84(uint64(elem.Score))
		positions[i] = []float64{lng, lat}
	}

	dist := geohash.GetDistance(positions[0][0], positions[0][1], positions[1][0], positions[1][1])

	mul, err := extractUnit(unit)
	if err != nil {
		return 0, err
	}

	return dist / mul, nil
}

// Hash return geo-hash-code of given position
func (g *Geo) Hash(members ...string) ([]string, error) {
	if len(members) == 0 {
		return nil, errors.New("ERR wrong number of arguments for 'geohash' command")
	}

	hashs := make([]string, len(members))
	for i := 0; i < len(members); i++ {
		member := members[i]
		elem, exists := g.sortedSet.Get(member)
		if !exists {
			hashs[i] = ""
			continue
		}

		str := geohash.ToString(uint64(elem.Score))
		hashs[i] = str
	}
	return hashs, nil
}

// Radius returns members within max distance of given point
func (g *Geo) Radius(lng, lat, radius float64, unit string) ([]*GeoPoint, error) {
	mul, err := extractUnit(unit)
	if err != nil {
		return nil, err
	}
	return g.geoRadius0(lat, lng, radius*mul, unit)
}

// RadiusByMember returns members within max distance of given member's location
func (g *Geo) RadiusByMember(member string, radius float64, unit string) ([]*GeoPoint, error) {
	elem, ok := g.sortedSet.Get(member)
	if !ok {
		return nil, ErrNull
	}
	lng, lat := geohash.DecodeToLongLatWGS84(uint64(elem.Score))

	mul, err := extractUnit(unit)
	if err != nil {
		return nil, err
	}

	return g.geoRadius0(lat, lng, radius*mul, unit)
}

func (g *Geo) membersOfGeoHashBox(longitude, latitude, radius float64, hash *geohash.HashBits) ([]*GeoPoint, error) {
	points := make([]*GeoPoint, 0, 32)
	boxMin, boxMax := geohash.ScoresOfGeoHashBox(hash)

	lower := &sortedset.ScoreBorder{Value: float64(boxMin)}
	upper := &sortedset.ScoreBorder{Value: float64(boxMax)}

	elements := g.sortedSet.Range(lower, upper, 0, -1, true)

	for _, v := range elements {
		x, y := geohash.DecodeToLongLatWGS84(uint64(v.Score))

		dist := geohash.GetDistance(x, y, longitude, latitude)

		if radius >= dist {
			p := &GeoPoint{
				Lng:    x,
				Lat:    y,
				Dist:   dist,
				Score:  v.Score,
				Member: v.Member,
			}
			points = append(points, p)
		}
	}

	return points, nil
}

func (g *Geo) geoMembersOfAllNeighbors(geoRadius *geohash.Radius, lon, lat, radius float64) ([]*GeoPoint, error) {
	neighbors := [9]*geohash.HashBits{
		&geoRadius.Hash,
		&geoRadius.North,
		&geoRadius.South,
		&geoRadius.East,
		&geoRadius.West,
		&geoRadius.NorthEast,
		&geoRadius.NorthWest,
		&geoRadius.SouthEast,
		&geoRadius.SouthWest,
	}

	var lastProcessed int = 0
	plist := make([]*GeoPoint, 0, 64)

	for i, area := range neighbors {
		if area.IsZero() {
			continue
		}
		if lastProcessed != 0 &&
			area.Bits == neighbors[lastProcessed].Bits &&
			area.Step == neighbors[lastProcessed].Step {
			continue
		}
		ps, err := g.membersOfGeoHashBox(lon, lat, radius, area)
		if err != nil {
			return nil, err
		} else {
			plist = append(plist, ps...)
		}
		lastProcessed = i
	}
	return plist, nil
}

func (g *Geo) geoRadius0(lat0 float64, lng0 float64, radius float64, unit string) ([]*GeoPoint, error) {
	radiusArea, err := geohash.GetAreasByRadiusWGS84(lng0, lat0, radius)
	if err != nil {
		return nil, err
	}

	plist, err := g.geoMembersOfAllNeighbors(radiusArea, lng0, lat0, radius)
	if err != nil {
		return nil, err
	}

	mul, err := extractUnit(unit)
	if err != nil {
		return nil, err
	}

	if mul != 1 {
		for _, n := range plist {
			n.Dist = n.Dist / mul
		}
	}

	return plist, nil
}

func extractUnit(unit string) (float64, error) {
	switch unit {
	case "m":
		return 1, nil
	case "km":
		return 1000, nil
	case "ft":
		return 0.3048, nil
	case "mi":
		return 1609.34, nil
	default:
		return -1, errors.New("Unsupported unit provided. please use m, km, ft, mi")
	}
}
