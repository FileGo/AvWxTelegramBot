package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

type outputData struct {
	data []byte
	err  error
}

type NOAAResponseData []struct {
	RawOb  string `json:"rawOb"`
	RawTaf string `json:"rawTaf"`
}

func (env *Env) getData(ICAO string, data chan outputData, wg *sync.WaitGroup) {
	var out outputData
	defer wg.Done()

	url := fmt.Sprintf("%s/api/data/metar?ids=%s&format=json&taf=true&hours=1.51%d", urlPrefix, ICAO, env.NOAAinterval)

	response, err := env.httpClient.Get(url)
	if err != nil {
		out.err = err
		data <- out
		close(data)
		return
	}

	buf, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println(string(buf))

		out.err = err
		data <- out
		close(data)
		return
	}

	out.data = buf
	data <- out
	close(data)
}

// ParseMetarNOAA retrieves raw text of latest METAR from NOAA
func ParseNOAAData(input []byte) (string, string, error) {
	var nresp NOAAResponseData
	err := json.Unmarshal(input, &nresp)
	if err != nil && err != io.EOF {
		return "", "", fmt.Errorf("unable to parse json: %v", err)
	}

	// Check if any METARs are available
	if len(nresp) < 1 {
		return "", "", errors.New("no METAR found")
	}

	return nresp[0].RawOb, nresp[0].RawTaf, nil
}
