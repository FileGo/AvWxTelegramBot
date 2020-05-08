package main

import (
	"reflect"
	"testing"
)

func TestGetICAOs(t *testing.T) {
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
		output := GetICAOs(test.input)

		if reflect.DeepEqual(output, test.expected) != true {
			t.Errorf("Test failed: \"%s\" input, \"%s\" expected, \"%s\" output",
				test.input, test.expected, output)
		}
	}
}
