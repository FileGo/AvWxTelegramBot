package main

import (
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestGetAirportCodes(t *testing.T) {
	var tests = []struct {
		input    string
		expected []string
	}{
		{"LJU STN GLA", []string{"LJU", "STN", "GLA"}},
		{" LJU STN    GLA    ", []string{"LJU", "STN", "GLA"}},
		{"EGSS,STN,EIDW", []string{"EGSS", "STN", "EIDW"}},
		{"  EGSS, STN,   EIDW   ", []string{"EGSS", "STN", "EIDW"}},
		{"STN", []string{"STN"}},
		{"lJu sTN GLa", []string{"LJU", "STN", "GLA"}},
		{"", []string{}},
		{"             ", []string{}},
		{",     ,    ", []string{}},
	}

	for _, test := range tests {
		output := GetAirportCodes(test.input)

		if reflect.DeepEqual(output, test.expected) != true {
			t.Errorf("Test failed: \"%s\" input, \"%s\" expected, \"%s\" output",
				test.input, test.expected, output)
		}
	}
}

type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func TestLoadAirports(t *testing.T) {
	tests := []struct {
		r           io.Reader
		errExpected bool
	}{
		{strings.NewReader(""), true}, // empty string
		{strings.NewReader(`[{"ICAO":"OM11","IATA":"","Name":"Abu Dhabi Northeast Airport"}]`), false},
		{&strings.Reader{}, true}, // nil reader
		{errReader(0), true},      // nil reader
	}

	for _, test := range tests {
		env := Env{}
		err := env.LoadAirports(test.r)
		if test.errExpected && err == nil {
			t.Errorf("input: %v, error expected, but not returned", test.r)
		}

		if !test.errExpected && err != nil {
			t.Errorf("input: %v, unexpected error: %v", test.r, err)
		}
	}
}

func TestFindAirport(t *testing.T) {
	tests := []struct {
		arpts       []Airport
		code        string
		want        Airport
		errExpected bool
	}{
		{
			[]Airport{{"EGLL", "LHR", "London Heathrow"}}, "EGLL", Airport{"EGLL", "LHR", "London Heathrow"}, false,
		},
		{
			[]Airport{{"EGLL", "LHR", "London Heathrow"}}, "LHR", Airport{"EGLL", "LHR", "London Heathrow"}, false,
		},
		{
			[]Airport{}, "toolongcode", Airport{}, true, // code too long
		},
		{
			[]Airport{}, "22", Airport{}, true, // code too short
		},
		{
			[]Airport{{"EGLL", "LHR", "London Heathrow"}}, "JFK", Airport{}, true, // inexistent airport
		},
	}

	for _, test := range tests {
		env := Env{}
		env.Airports = test.arpts

		out, err := env.FindAirport(test.code)
		if test.errExpected && err == nil {
			t.Errorf("arpts: %v, code: %v, error expected, but not returned", test.arpts, test.code)
		}

		if !test.errExpected && err != nil {
			t.Errorf("arpts: %v, code: %v, unexpected error: %v", test.arpts, test.code, err)
		}

		if out != test.want {
			t.Errorf("arpts: %v, code: %v, got: %v, want: %v", test.arpts, test.code, out, test.want)
		}

	}
}

func TestGetNOAAinterval(t *testing.T) {
	// Test default
	out, err := GetNOAAinterval()
	if err != nil {
		t.Errorf("default interval, error received: %v", err)
	}

	if out != 12 {
		t.Errorf("default interval not 12, got: %v", out)
	}

	// Set a positive number
	tests := []struct {
		input       string
		want        int
		errExpected bool
	}{
		{"15", 15, false},
		{"-1", 0, true},
		{"aaaaaaaaaaaaa", 0, true},
	}

	for _, test := range tests {
		err = os.Setenv("NOAA_INTERVAL", test.input)
		if err != nil {
			t.Errorf("unable to set interval: %v", err)
		}

		out, err = GetNOAAinterval()

		if test.errExpected && err == nil {
			t.Errorf("input: %v, got: %v, error expected but not returned", test.input, out)
		}

		if !test.errExpected && err != nil {
			t.Errorf("input: %v, unexpected error: %v", test.input, err)
		}

		if out != test.want {
			t.Errorf("input: %v, got: %v, wanted: %v", test.input, out, test.want)
		}
	}

}
