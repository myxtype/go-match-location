package region

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

func LoadRegion(file string) (*Region, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var r Region
	return &r, json.Unmarshal(data, &r)
}
