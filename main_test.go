package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
)

func newTestServer() *httptest.Server {
	brokerProcess()
	mux := http.NewServeMux()
	mux.HandleFunc("/fetch", fetchData)
	mux.HandleFunc("/save", saveData)
	return httptest.NewServer(mux)
}

func doPostRequest(t *testing.T, data string, URL string, client *http.Client) ([]byte, int) {
	method := http.MethodPost
	request, err := http.NewRequest(method, URL+"/save", bytes.NewBufferString(data))
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}
	request.Header.Set("Content-Type", "text/plain")
	response, err := client.Do(request)
	if err != nil {
		t.Errorf("failed to send %v request: %v", method, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Errorf("failed to read response body: %v", err)
	}
	return body, response.StatusCode
}

func doGetRequest(t *testing.T, URL string, client *http.Client) ([]byte, int) {
	method := http.MethodGet
	request, err := http.NewRequest(method, URL+"/fetch", nil)
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}
	request.Header.Set("Content-Type", "text/plain")
	response, err := client.Do(request)
	if err != nil {
		t.Errorf("failed to send %v request: %v", method, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Errorf("failed to read response body: %v", err)
	}
	return body, response.StatusCode
}

func TestSaveThenFetch(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	want := "88888888" // fixed timestamp
	client := &http.Client{}

	// POST
	doPostRequest(t, want, ts.URL, client)

	// GET
	body, _ := doGetRequest(t, ts.URL, client)
	if string(body) != want {
		t.Errorf("save and fetch not matching, want %v but get %v", want, string(body))
	}
}

func TestBadRequest(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()
	invalidTimestamp := "AABB"
	want := http.StatusBadRequest
	client := &http.Client{}

	// POST
	_, statusCode := doPostRequest(t, invalidTimestamp, ts.URL, client)

	if statusCode != want {
		t.Errorf("expect 400 error code, but get %d", statusCode)
	}

}

func TestConcurrentReadWrite(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()
	client := &http.Client{}
	body, _ := doPostRequest(t, strconv.FormatInt(int64(0), 10), ts.URL, client)
	before, _ := strconv.ParseInt(string(body), 10, 64)

	wg := new(sync.WaitGroup)
	for i := 1; i <= 50; i++ {
		wg.Go(func() {
			client := &http.Client{}
			doPostRequest(t, strconv.FormatInt(int64(1000+i), 10), ts.URL, client)
		})

	}
	for i := 1; i <= 50; i++ {
		wg.Go(func() {
			client := &http.Client{}
			doGetRequest(t, ts.URL, client)
		})

	}
	wg.Wait()
	body, _ = doPostRequest(t, strconv.FormatInt(int64(0), 10), ts.URL, client)
	after, _ := strconv.ParseInt(string(body), 10, 64)
	if after-before != 101 {
		t.Errorf("expect concurrent read and write to be handled")
	}

}
