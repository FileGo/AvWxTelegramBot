package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sync"
)

const (
	dataSourceMetar = "metars"
	dataSourceTaf   = "tafs"
)

type outputData struct {
	data []byte
	err  error
}

// NOAAResponseMetar provides struct for XML unmarshalling of NOAA data
type NOAAResponseMetar struct {
	XMLName                   xml.Name `xml:"response"`
	Text                      string   `xml:",chardata"`
	Xsd                       string   `xml:"xsd,attr"`
	Xsi                       string   `xml:"xsi,attr"`
	Version                   string   `xml:"version,attr"`
	NoNamespaceSchemaLocation string   `xml:"noNamespaceSchemaLocation,attr"`
	RequestIndex              string   `xml:"request_index"`
	DataSource                struct {
		Text string `xml:",chardata"`
		Name string `xml:"name,attr"`
	} `xml:"data_source"`
	Request struct {
		Text string `xml:",chardata"`
		Type string `xml:"type,attr"`
	} `xml:"request"`
	Errors      string `xml:"errors"`
	Warnings    string `xml:"warnings"`
	TimeTakenMs string `xml:"time_taken_ms"`
	Data        struct {
		Text       string `xml:",chardata"`
		NumResults string `xml:"num_results,attr"`
		METAR      []struct {
			Text                string `xml:",chardata"`
			RawText             string `xml:"raw_text"`
			StationID           string `xml:"station_id"`
			ObservationTime     string `xml:"observation_time"`
			Latitude            string `xml:"latitude"`
			Longitude           string `xml:"longitude"`
			TempC               string `xml:"temp_c"`
			DewpointC           string `xml:"dewpoint_c"`
			WindDirDegrees      string `xml:"wind_dir_degrees"`
			WindSpeedKt         string `xml:"wind_speed_kt"`
			VisibilityStatuteMi string `xml:"visibility_statute_mi"`
			AltimInHg           string `xml:"altim_in_hg"`
			QualityControlFlags struct {
				Text     string `xml:",chardata"`
				NoSignal string `xml:"no_signal"`
			} `xml:"quality_control_flags"`
			SkyCondition struct {
				Text     string `xml:",chardata"`
				SkyCover string `xml:"sky_cover,attr"`
			} `xml:"sky_condition"`
			FlightCategory string `xml:"flight_category"`
			MetarType      string `xml:"metar_type"`
			ElevationM     string `xml:"elevation_m"`
		} `xml:"METAR"`
	} `xml:"data"`
}

// NOAAResponseTaf provides struct for XML unmarshalling of NOAA data
type NOAAResponseTaf struct {
	XMLName                   xml.Name `xml:"response"`
	Text                      string   `xml:",chardata"`
	Xsd                       string   `xml:"xsd,attr"`
	Xsi                       string   `xml:"xsi,attr"`
	Version                   string   `xml:"version,attr"`
	NoNamespaceSchemaLocation string   `xml:"noNamespaceSchemaLocation,attr"`
	RequestIndex              string   `xml:"request_index"`
	DataSource                struct {
		Text string `xml:",chardata"`
		Name string `xml:"name,attr"`
	} `xml:"data_source"`
	Request struct {
		Text string `xml:",chardata"`
		Type string `xml:"type,attr"`
	} `xml:"request"`
	Errors      string `xml:"errors"`
	Warnings    string `xml:"warnings"`
	TimeTakenMs string `xml:"time_taken_ms"`
	Data        struct {
		Text       string `xml:",chardata"`
		NumResults string `xml:"num_results,attr"`
		TAF        []struct {
			Text          string `xml:",chardata"`
			RawText       string `xml:"raw_text"`
			StationID     string `xml:"station_id"`
			IssueTime     string `xml:"issue_time"`
			BulletinTime  string `xml:"bulletin_time"`
			ValidTimeFrom string `xml:"valid_time_from"`
			ValidTimeTo   string `xml:"valid_time_to"`
			Latitude      string `xml:"latitude"`
			Longitude     string `xml:"longitude"`
			ElevationM    string `xml:"elevation_m"`
			Forecast      struct {
				Text                string `xml:",chardata"`
				FcstTimeFrom        string `xml:"fcst_time_from"`
				FcstTimeTo          string `xml:"fcst_time_to"`
				WindDirDegrees      string `xml:"wind_dir_degrees"`
				WindSpeedKt         string `xml:"wind_speed_kt"`
				VisibilityStatuteMi string `xml:"visibility_statute_mi"`
				WxString            string `xml:"wx_string"`
				SkyCondition        struct {
					Text           string `xml:",chardata"`
					SkyCover       string `xml:"sky_cover,attr"`
					CloudBaseFtAgl string `xml:"cloud_base_ft_agl,attr"`
				} `xml:"sky_condition"`
			} `xml:"forecast"`
		} `xml:"TAF"`
	} `xml:"data"`
}

func (env *Env) getData(dataSource string, ICAO string, data chan outputData, wg *sync.WaitGroup) {
	var out outputData
	defer wg.Done()

	url := fmt.Sprintf("%s?dataSource=%s&requestType=retrieve&format=xml&stationString=%s&hoursBeforeNow=%d", urlPrefix, dataSource, ICAO, env.NOAAinterval)

	response, err := env.httpClient.Get(url)
	if err != nil {
		out.err = err
		data <- out
		close(data)
		return
	}

	buf, err := io.ReadAll(response.Body)
	if err != nil {
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
func ParseMetarNOAA(input []byte) (string, error) {
	var nresp NOAAResponseMetar
	err := xml.Unmarshal([]byte(input), &nresp)
	if err != nil && err != io.EOF {
		return "", err
	}

	// Check if any METARs are available
	if len(nresp.Data.METAR) <= 0 {
		return "", errors.New("no METAR found")
	}

	// Return the newest one
	return nresp.Data.METAR[0].RawText, nil
}

// ParseTafNOAA retrieves raw text of latest METAR from NOAA
func ParseTafNOAA(input []byte) (string, error) {
	var nresp NOAAResponseTaf
	err := xml.Unmarshal([]byte(input), &nresp)
	if err != nil && err != io.EOF {
		return "", err
	}

	// Check if any METARs are available
	if len(nresp.Data.TAF) <= 0 {
		return "", errors.New("no TAF found")
	}

	// Return the newest one
	return nresp.Data.TAF[0].RawText, nil
}
