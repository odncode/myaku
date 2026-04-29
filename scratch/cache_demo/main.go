package main

import (
	"fmt"
	"log"
	"time"

	"myaku/store"
)

func main() {
	// --- init DB ---
	s, err := store.NewStore("postgres://localhost:5432/myaku")
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	// --- init cache ---
	c := store.NewCache()

	// 1. Add site to Postgres
	siteID, err := s.AddSite("https://linkedin.com")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted site ID:", siteID)

	// Fetch from DB (source of truth)
	site, err := s.GetSite(siteID)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Cache it (10s TTL)
	err = c.CacheStatus(siteID, site, 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Cached in Redis")

	// 3. Read from cache (HIT)
	cached1, err := c.GetCachedStatus(siteID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Cache HIT:", cached1)

	// 4. Wait for expiry
	fmt.Println("Waiting 11 seconds for TTL expiry...")
	time.Sleep(11 * time.Second)

	// 5. Read again (MISS)
	cached2, err := c.GetCachedStatus(siteID)
	if err == nil {
		fmt.Println("Cache HIT (unexpected):", cached2)
	} else {
		fmt.Println("Cache MISS (expected)")
	}

	// 6. Fall back to DB
	siteFromDB, err := s.GetSite(siteID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Fetched from DB:", siteFromDB)

	// 7. Re-cache it
	err = c.CacheStatus(siteID, siteFromDB, 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Re-cached in Redis")

	// 8. Final cache hit
	final, err := c.GetCachedStatus(siteID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Final Cache HIT:", final)
}
