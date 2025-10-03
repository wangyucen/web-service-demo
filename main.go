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
	_, err := fmt.Fprintf(w, <-read.resp)
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
	write := writeOp{
		val:  string(data),
		resp: make(chan string),
	}
	writes <- write
	w.Header().Set("Content-Type", "text/plain")
	_, err = fmt.Fprintf(w, <-write.resp)
	if err != nil {
		return
	}

}

func server() error {
	go func() {
		var TimeStamp string
		for {
			select {
			case write := <-writes:
				TimeStamp = write.val
				write.resp <- "unix time stamp stored successfully\n"
			case read := <-reads:
				read.resp <- TimeStamp
			}
		}
	}()
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
		body, _ := io.ReadAll(response.Body)
		fmt.Println("GET response:", string(body))

	case http.MethodPost:
		unixStr := strconv.FormatInt(time.Now().Unix(), 10)
		request, err := http.NewRequest("POST", "http://127.0.0.1:8090/save", bytes.NewBufferString(unixStr))
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
		body, _ := io.ReadAll(response.Body)
		fmt.Println("POST response:", string(body))
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
