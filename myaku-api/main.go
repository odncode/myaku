package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"myaku/store"
	"myaku/uptime-cli/site"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func main() {

	db, err := store.NewStore("postgres://localhost:5432/myaku")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	cache := store.NewCache()

	http.HandleFunc("/api/sites/", func(w http.ResponseWriter, r *http.Request) {

		idStr := strings.TrimPrefix(r.URL.Path, "/api/sites/")

		switch r.Method {

		case http.MethodPost:

			// POST /api/sites/{id}/checks
			if !strings.HasSuffix(r.URL.Path, "/checks") {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}

			idStr = strings.TrimSuffix(
				strings.TrimPrefix(r.URL.Path, "/api/sites/"),
				"/checks",
			)

			siteID, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, "invalid id", http.StatusBadRequest)
				return
			}

			s, err := db.GetSite(siteID)
			if err != nil {
				http.Error(w, "site not found", http.StatusNotFound)
				return
			}

			checkSite := &site.Site{
				URL: s.URL,
			}

			result, err := checkSite.PerformCheck()

			status := "down"
			if err == nil {
				status = fmt.Sprintf("%d", result.StatusCode)
			}

			err = db.AddCheck(siteID, store.CheckResult{
				StatusCode:   result.StatusCode,
				ResponseTime: result.ResponseTime,
				IsUp:         result.IsUp,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			err = db.UpdateSiteStatus(
				siteID,
				status,
				result.ResponseTime,
				result.IsUp,
				s.CheckCount+1,
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			updatedSite := store.Site{
				ID:           siteID,
				URL:          s.URL,
				Status:       status,
				IsUp:         result.IsUp,
				ResponseTime: result.ResponseTime,
				CheckCount:   s.CheckCount + 1,
			}

			cache.CacheStatus(siteID, updatedSite, 30*time.Second)

			writeJSON(w, http.StatusOK, result)

		case http.MethodGet:

			siteID, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, "invalid id", http.StatusBadRequest)
				return
			}

			// try cache first
			cachedSite, err := cache.GetCachedStatus(siteID)

			if err == nil {
				writeJSON(w, http.StatusOK, cachedSite)
				return
			}

			// fallback to DB
			s, err := db.GetSite(siteID)
			if err != nil {
				http.Error(w, "site not found", http.StatusNotFound)
				return
			}

			cache.CacheStatus(siteID, s, 30*time.Second)

			writeJSON(w, http.StatusOK, s)

		case http.MethodDelete:

			siteID, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, "invalid id", http.StatusBadRequest)
				return
			}

			err = db.DeleteSite(siteID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			cache.InvalidateCache(siteID)

			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/sites", func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {

		case http.MethodPost:

			var input struct {
				URL string `json:"url"`
			}

			err := json.NewDecoder(r.Body).Decode(&input)
			if err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}

			if !strings.HasPrefix(input.URL, "http") {
				input.URL = "https://" + input.URL
			}

			newID, err := db.AddSite(input.URL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			writeJSON(w, http.StatusCreated, map[string]any{
				"id":     newID,
				"url":    input.URL,
				"status": "unknown",
			})

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

	http.Handle("/", http.FileServer(http.Dir("./static")))

	fmt.Println("Server running on :8081")
	http.ListenAndServe(":8081", nil)
}
