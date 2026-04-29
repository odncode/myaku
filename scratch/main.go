package main

import (
	"fmt"
	"myaku/uptime-cli/site"
	"time"
)

func checkSite(ch chan site.CheckResult, url string) {
	url = "https://" + url
	s := &site.Site{URL: url}
	result, _ := s.PerformCheck()
	ch <- result
}

func main() {
	urls := []string{"google.com", "github.com", "linkedin.com", "tesla.com", "tesco.com", "thestudentroom.co.uk"}
	ch := make(chan site.CheckResult)

	start := time.Now()

	for _, url := range urls {
		go checkSite(ch, url)
	}

	var upCount int
	var downCount int
	var fastest site.CheckResult
	var slowest site.CheckResult

	for i := range urls {
		result := <-ch

		fmt.Println(result)

		if result.IsUp {
			upCount++
		} else {
			downCount++
		}

		if i == 0 || result.ResponseTime < fastest.ResponseTime {
			fastest = result
		}

		if i == 0 || result.ResponseTime > slowest.ResponseTime {
			slowest = result
		}
	}

	elapsed := time.Since(start)

	fmt.Println("\nSummary")
	fmt.Println("-------")
	fmt.Printf("Total checked: %d\n", len(urls))
	fmt.Printf("Up: %d\n", upCount)
	fmt.Printf("Down: %d\n", downCount)
	fmt.Printf("Fastest: %s (%.3fs)\n", fastest.URL, fastest.ResponseTime)
	fmt.Printf("Slowest: %s (%.3fs)\n", slowest.URL, slowest.ResponseTime)
	fmt.Printf("Concurrent total time: %.3fs\n", elapsed.Seconds())

}
