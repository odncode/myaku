package main

import (
	"encoding/json"
	"fmt"
	"log"
	"myaku/store"
	"myaku/uptime-cli/site"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func main() {

	// ================= DB =================
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://localhost:5432/myaku"
	}

	db, err := store.NewStore(dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// ================= CACHE =================
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	cache := store.NewCache(redisURL)

	// ================= CREATE + LIST =================
	http.HandleFunc("/api/sites", func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {

		// CREATE SITE
		case http.MethodPost:
			var input struct {
				URL string `json:"url"`
			}

			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}

			if !strings.HasPrefix(input.URL, "http") {
				input.URL = "https://" + input.URL
			}

			id, err := db.AddSite(input.URL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			writeJSON(w, http.StatusCreated, store.Site{
				ID:     id,
				URL:    input.URL,
				Status: "unknown",
			})

		// LIST SITES
		case http.MethodGet:
			sites, err := db.ListSites()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			writeJSON(w, http.StatusOK, sites)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ================= SINGLE SITE OPS =================
	http.HandleFunc("/api/sites/", func(w http.ResponseWriter, r *http.Request) {

		idStr := strings.TrimPrefix(r.URL.Path, "/api/sites/")

		// ---------------- CHECK SITE ----------------
		if strings.HasSuffix(r.URL.Path, "/checks") && r.Method == http.MethodPost {

			idStr = strings.TrimSuffix(idStr, "/checks")

			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, "invalid id", http.StatusBadRequest)
				return
			}

			siteRow, err := db.GetSite(id)
			if err != nil {
				http.Error(w, "site not found", http.StatusNotFound)
				return
			}

			// 🔥 THIS IS THE ONLY CORRECT FIX
			checker := site.Site{URL: siteRow.URL}
			result, err := checker.PerformCheck()

			status := "down"
			if err == nil {
				status = fmt.Sprintf("%d", result.StatusCode)
			}

			_ = db.AddCheck(id, store.CheckResult{
				StatusCode:   result.StatusCode,
				ResponseTime: result.ResponseTime,
				IsUp:         result.IsUp,
			})

			_ = db.UpdateSiteStatus(
				id,
				status,
				result.ResponseTime,
				result.IsUp,
				siteRow.CheckCount+1,
			)

			updated := store.Site{
				ID:           id,
				URL:          siteRow.URL,
				Status:       status,
				ResponseTime: result.ResponseTime,
				IsUp:         result.IsUp,
				CheckCount:   siteRow.CheckCount + 1,
			}

			_ = cache.CacheStatus(id, updated, 30*time.Second)

			writeJSON(w, http.StatusOK, updated)
			return
		}

		// ---------------- GET SITE ----------------
		if r.Method == http.MethodGet {

			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, "invalid id", http.StatusBadRequest)
				return
			}

			cached, err := cache.GetCachedStatus(id)
			if err == nil {
				writeJSON(w, http.StatusOK, cached)
				return
			}

			siteRow, err := db.GetSite(id)
			if err != nil {
				http.Error(w, "site not found", http.StatusNotFound)
				return
			}

			_ = cache.CacheStatus(id, siteRow, 30*time.Second)

			writeJSON(w, http.StatusOK, siteRow)
			return
		}

		// ---------------- DELETE SITE ----------------
		if r.Method == http.MethodDelete {

			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, "invalid id", http.StatusBadRequest)
				return
			}

			_ = db.DeleteSite(id)
			_ = cache.InvalidateCache(id)

			w.WriteHeader(http.StatusNoContent)
			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	// ================= FRONTEND =================
	http.Handle("/", http.FileServer(http.Dir("./static")))

	fmt.Println("Server running on: 8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
