package main

import (
	"fmt"
	"os"
	"uptime-cli/site"
	"encoding/json"

) 








// MAIN SYSTEM FLOW:
//
// 1. Read input URLs from CLI
// 2. For each URL:
//    a. Create a Site instance
//    b. Perform HTTP check (external system)
//    c. Store raw result (history tracking)
//    d. Update internal state (business logic)
//    e. Print real-time status
// 3. After loop:
//    a. Compute summary statistics
//    b. Persist results to JSON file
//
// DESIGN GOAL:
// main.go is intentionally thin — it only orchestrates flow.
// All real logic lives in the site package.


func main() {
	
	var results []site.CheckResult
	
	for _, url := range os.Args[1:] {
		// Step 1: Create isolated Site instance for this URL
		s := site.Site{URL: url}
		
		// Step 2: Execute external HTTP request (side-effect boundary)(inside 				//		PerformCheck)
		result, err := s.PerformCheck()
		if err != nil {
			fmt.Printf("Error on checking %v:, %v\n", url, err)
			continue
		}
		
		// Step 3: Store raw result for reporting + persistence
		results = append(results, result)
		
		// Step 4: Update internal system state from raw result
		err = s.Update(result)
		if err != nil {
			fmt.Printf("Error on updating %v:, %v\n", url, err)
			continue
		}
		
		// Step 5: Immediate user feedback (CLI output)
		fmt.Printf("%s: %s (%.2fms)\n", s.URL, s.Status, result.ResponseTime*1000)

	}
		
	// SUMMARY COMPUTATION:
	//
	// This section transforms raw results into insights:
	// - total sites checked
	// - number of successful vs failed checks
	// - average latency of successful requests
	//
	// DESIGN IDEA:
	// We separate "data collection" (loop above) from "analysis" (this block)
	// to keep the system readable and easy to extend.

	total_sites := len(results)
	up := 0
	up_response := 0.0

	for _, r := range results {
    		if r.StatusCode >= 200 && r.StatusCode < 300 {
        		up++
        		up_response += r.ResponseTime
    		}
	}
	
	down := total_sites - up
	
	fmt.Println("\n--- SUMMARY ---")
	fmt.Printf("Total Sites: %d\n", total_sites)
	fmt.Printf("Up: %d\n", up)
	fmt.Printf("Down: %d\n", down)

	if up > 0 {
		avg := up_response / float64(up)
		fmt.Printf("Avg Response Time (up sites): %.2fms\n", avg*1000)
	} else {
		fmt.Printf("Avg Response Time (up sites): N/A\n")
	}
	

	
	// PERSISTENCE LAYER:
	//
	// Converts in-memory results into a stable external format (JSON).
	// This allows:
	// - historical tracking
	// - debugging past runs
	// - future analytics
	//
	// DESIGN IDEA:
	// Your CLI is not just a tool — it's also a data generator.	

	data, err := json.MarshalIndent(results, "", " ")
	if err != nil {
		fmt.Printf("Error creating JSON: %v\n", err)
    		return
	}
	
	err = os.WriteFile("results.json", data, 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
    		return

	}

	
}










