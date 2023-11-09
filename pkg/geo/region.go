package geo

import (
	"encoding/json"
	"io"
	"os"
)

type Region struct {
	Name   string `json:"name"`
	Center struct {
		Lat float64 `json:"latitude"`
		Lng float64 `json:"longitude"`
	}
	Level     string    `json:"level"`
	Districts []*Region `json:"districts"`
}

func LoadRegion() (*Region, error) {
	f, err := os.Open("./region.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var r Region
	return &r, json.Unmarshal(b, &r)
}
