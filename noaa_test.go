package main

import (
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testingHTTPClient(handler http.Handler, options ...bool) (*http.Client, func()) {
	s := httptest.NewTLSServer(handler)
	fail := false
	if len(options) > 0 {
		fail = options[0]
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				if fail {
					return &net.TCPConn{}, nil
				}
				return net.Dial(network, s.Listener.Addr().String())
			},
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return client, s.Close
}

func TestGetData(t *testing.T) {
	assert := assert.New(t)

	t.Run("pass", func(t *testing.T) {
		h := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			f, err := os.Open("./testdata/kjfk_metar.xml")
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			io.Copy(rw, f)
		})
		httpClient, teardown := testingHTTPClient(h)
		defer teardown()

		env := &Env{httpClient: httpClient}
		var wg sync.WaitGroup
		data := make(chan outputData, 1)

		wg.Add(1)
		go env.getData(dataSourceMetar, "KJFK", data, &wg)
		wg.Wait()

		out := <-data
		if assert.Nil(out.err) {
			assert.Contains(string(out.data), "KJFK")
		}
	})

	t.Run("http_fail", func(t *testing.T) {
		httpClient, teardown := testingHTTPClient(nil, true)
		defer teardown()

		env := &Env{httpClient: httpClient}
		var wg sync.WaitGroup
		data := make(chan outputData, 1)

		wg.Add(1)
		go env.getData(dataSourceMetar, "KJFK", data, &wg)
		wg.Wait()

		out := <-data
		if assert.NotNil(out.err) {
			assert.Contains(out.err.Error(), "http")
		}
	})
}

func TestGetMetarNOAA(t *testing.T) {
	assert := assert.New(t)

	t.Run("pass", func(t *testing.T) {
		buf, err := ioutil.ReadFile("./testdata/kjfk_metar.xml")
		if err != nil {
			t.Fatal(err)
		}

		out, err := ParseMetarNOAA(buf)
		if assert.Nil(err) {
			assert.Equal("KJFK", out[:4])
		}
	})

	t.Run("no_metar", func(t *testing.T) {
		buf, err := ioutil.ReadFile("./testdata/no_metar.xml")
		if err != nil {
			t.Fatal(err)
		}

		_, err = ParseMetarNOAA(buf)
		if assert.NotNil(err) {
			assert.Contains(err.Error(), "no METAR")
		}
	})

	t.Run("xml_fail", func(t *testing.T) {
		buf, err := ioutil.ReadFile("./testdata/fail.xml")
		if err != nil {
			t.Fatal(err)
		}

		_, err = ParseMetarNOAA(buf)
		if assert.NotNil(err) {
			assert.Contains(err.Error(), "XML syntax")
		}
	})
}

func TestGetTafNOAA(t *testing.T) {
	assert := assert.New(t)

	t.Run("pass", func(t *testing.T) {
		buf, err := ioutil.ReadFile("./testdata/kjfk_taf.xml")
		if err != nil {
			t.Fatal(err)
		}

		out, err := ParseTafNOAA(buf)
		if assert.Nil(err) {
			assert.Equal("KJFK", out[:4])
		}
	})

	t.Run("no_taf", func(t *testing.T) {
		buf, err := ioutil.ReadFile("./testdata/no_taf.xml")
		if err != nil {
			t.Fatal(err)
		}

		_, err = ParseTafNOAA(buf)
		if assert.NotNil(err) {
			assert.Contains(err.Error(), "no TAF")
		}
	})

	t.Run("xml_fail", func(t *testing.T) {
		buf, err := ioutil.ReadFile("./testdata/fail.xml")
		if err != nil {
			t.Fatal(err)
		}

		_, err = ParseTafNOAA(buf)
		if assert.NotNil(err) {
			assert.Contains(err.Error(), "XML syntax")
		}
	})
}
