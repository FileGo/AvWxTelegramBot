package main

import (
	"context"
	"crypto/tls"
	"io"
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
			f, err := os.Open("./testdata/kjfk_data.json")
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
		go env.getData("KJFK", data, &wg)
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
		go env.getData("KJFK", data, &wg)
		wg.Wait()

		out := <-data
		if assert.NotNil(out.err) {
			assert.Contains(out.err.Error(), "http")
		}
	})
}

func TestParseDataNOAA(t *testing.T) {
	assert := assert.New(t)

	t.Run("pass", func(t *testing.T) {
		buf, err := os.ReadFile("./testdata/kjfk_data.json")
		if err != nil {
			t.Fatal(err)
		}

		metar, taf, err := ParseNOAAData(buf)
		if assert.Nil(err) {
			assert.Equal("METAR KJFK", metar[:10])
			assert.Equal("TAF KJFK", taf[:8])
		}
	})

	t.Run("json_fail", func(t *testing.T) {
		buf, err := os.ReadFile("./testdata/fail.json")
		if err != nil {
			t.Fatal(err)
		}

		_, _, err = ParseNOAAData(buf)
		if assert.NotNil(err) {
			assert.Contains(err.Error(), "json")
		}
	})
}
