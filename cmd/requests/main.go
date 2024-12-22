package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func main() {
	for i := 0; i < 100000; i++ {
		time.Sleep(10 * time.Millisecond)
		reader := strings.NewReader("")
		request, _ := http.NewRequest("GET", "http://localhost:8080/resource1", reader)
		client := &http.Client{}
		resp, _ := client.Do(request)
		resp.Body.Close()
		fmt.Println(resp.StatusCode)
		fmt.Println(resp.Request.URL)
		if resp.StatusCode == http.StatusServiceUnavailable {
			fmt.Println("servers unavailable")
			break
		}
	}
}
