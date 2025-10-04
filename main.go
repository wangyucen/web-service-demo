/*
why not suggest to release this app to the production environment
1. the communication is not encrypted, no TLS/SSL involved.
2. we do not have persistent data storage for the timestamp here, every restart of the service will empty the memory stored string.
3. used non-buffered channel for demonstration purpose, potentially lead to a deadlock
*/
package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

var writes = make(chan writeOp)
var reads = make(chan readOp)
var counter int64 = 0

type readOp struct {
	resp chan string
}
type writeOp struct {
	val  string
	resp chan string
}

func fetchData(w http.ResponseWriter, req *http.Request) {
	read := readOp{
		resp: make(chan string),
	}
	reads <- read
	w.Header().Set("Content-Type", "text/plain")
	_, err := fmt.Fprintf(w, "%s", <-read.resp)
	if err != nil {
		return
	}
}

func saveData(w http.ResponseWriter, req *http.Request) {
	data, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err = strconv.ParseInt(string(data), 10, 64); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	write := writeOp{
		val:  string(data),
		resp: make(chan string),
	}
	writes <- write
	w.Header().Set("Content-Type", "text/plain")
	_, err = fmt.Fprintf(w, "%s", <-write.resp)
	if err != nil {
		return
	}

}
func brokerProcess() {
	go func() {
		var TimeStamp string
		for {
			select {
			case write := <-writes:
				counter++
				TimeStamp = write.val
				write.resp <- strconv.FormatInt(int64(counter), 10)

			case read := <-reads:
				read.resp <- TimeStamp
				counter++
			}
		}
	}()
}
func server() error {
	// initiate a broker process to handle concurrent requests
	brokerProcess()
	http.HandleFunc("/fetch", fetchData)
	http.HandleFunc("/save", saveData)
	if err := http.ListenAndServe(":8090", nil); err != nil {
		return fmt.Errorf("ListenAndServe: %w", err)
	}
	return nil
}

func clientRequest(method string, client *http.Client) error {
	switch method {
	case http.MethodGet:
		request, err := http.NewRequest(method, "http://127.0.0.1:8090/fetch", nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		request.Header.Set("Content-Type", "text/plain")
		response, err := client.Do(request)
		if err != nil {
			return fmt.Errorf("failed to send %v request: %w", http.MethodGet, err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("server returned status code %d", response.StatusCode)
		}
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		fmt.Printf("GET: %s -> %d\n", string(body), response.StatusCode)

	case http.MethodPost:
		unixStr := strconv.FormatInt(time.Now().Unix(), 10)
		request, err := http.NewRequest(method, "http://127.0.0.1:8090/save", bytes.NewBufferString(unixStr))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		request.Header.Set("Content-Type", "text/plain")
		response, err := client.Do(request)
		if err != nil {
			return fmt.Errorf("failed to send %v request: %w", http.MethodPost, err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("server returned status code %d", response.StatusCode)

		}
		_, err = io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
	}

	return nil
}

func main() {
	// server side simulation
	go func() {
		if err := server(); err != nil {
			panic(err)
		}

	}()

	time.Sleep(200 * time.Millisecond)

	// client side simulations
	client := &http.Client{}
	err := clientRequest(http.MethodPost, client)
	if err != nil {
		panic(err)
	}
	err = clientRequest(http.MethodGet, client)
	if err != nil {
		panic(err)
	}

}
