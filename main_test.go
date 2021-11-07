package main

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAirportCodes(t *testing.T) {
	assert := assert.New(t)

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

		assert.Equal(test.expected, output)
	}
}

type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func TestLoadAirports(t *testing.T) {
	assert := assert.New(t)

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

		if test.errExpected {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
	}
}

func TestFindAirport(t *testing.T) {
	assert := assert.New(t)

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

		if test.errExpected {
			assert.NotNil(err)
		} else {
			if assert.Nil(err) {
				assert.Equal(test.want, out)
			}
		}
	}
}

func TestGetNOAAinterval(t *testing.T) {
	assert := assert.New(t)
	env := &Env{}

	t.Run("default", func(t *testing.T) {
		// Test default
		err := env.GetNOAAinterval()
		if assert.Nil(err) {
			assert.Equal(12, env.NOAAinterval)
		}
	})

	t.Run("pass", func(t *testing.T) {
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
			err := os.Setenv("NOAA_INTERVAL", test.input)
			if err != nil {
				t.Fatalf("unable to set interval: %v", err)
			}

			err = env.GetNOAAinterval()
			if test.errExpected {
				assert.NotNil(err)
			} else {
				if assert.Nil(err) {
					assert.Equal(test.want, env.NOAAinterval)
				}
			}
		}
	})

}
