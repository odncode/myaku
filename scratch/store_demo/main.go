package main

import (
	"fmt"
	"log"
	"myaku/store"
)

func main() {
	s, err := store.NewStore("postgres://localhost:5432/myaku")
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	// Add site
	id, err := s.AddSite("https://google.com")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted site ID:", id)

	// Get site
	site, err := s.GetSite(id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Got site:", site)

	// List sites
	sites, err := s.ListSites()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("All sites:", sites)

	// Add check
	err = s.AddCheck(id, store.CheckResult{
		StatusCode:   200,
		ResponseTime: 0.3,
		IsUp:         true,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Get checks
	checks, err := s.GetChecks(id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Checks:", checks)
}
