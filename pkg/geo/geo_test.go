package geo

import (
	"log"
	"testing"
)

func regionGeo() *Geo {
	re, _ := LoadRegion()

	g := NewGeo()

	addToGeo(g, re)

	return g
}

func addToGeo(g *Geo, region *Region) {
	var items []*GeoItem

	for _, n := range region.Districts {
		items = append(items, &GeoItem{
			Lng:    n.Center.Lng,
			Lat:    n.Center.Lat,
			Member: n.Name,
		})
		if len(n.Districts) > 0 {
			addToGeo(g, n)
		}
	}

	if err := g.Add(items...); err != nil {
		log.Fatalln(err)
	}
}

func TestNewGeo(t *testing.T) {
	g := regionGeo()

	pos, err := g.Pos("中华人民共和国", "澳门特别行政区")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(pos[0])
	t.Log(g.Hash("澳门特别行政区"))

	t.Log(g.Dist([2]string{"成都市", "南部县"}, "km"))

	radius, err := g.RadiusByMember("南部县", 60, "km")
	if err != nil {
		t.Fatal(err)
	}

	for _, n := range radius {
		t.Log(n.Member, n.Lat, n.Lng, n.Dist)
	}
}
