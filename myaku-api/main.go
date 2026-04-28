package main

import (
	"encoding/json"
	"fmt"
	"myaku/uptime-cli/site"
	"net/http"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func main() {

	var sites = make(map[string]*site.Site)
	var counterID = 0

	http.HandleFunc("/api/sites/", func(w http.ResponseWriter, r *http.Request) {

		id := strings.TrimPrefix(r.URL.Path, "/api/sites/")

		switch r.Method {

		case http.MethodPost:
			if !strings.HasSuffix(r.URL.Path, "/checks") {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}

			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sites/"), "/checks")

			s, exists := sites[id]
			if !exists {
				http.Error(w, "site not found", http.StatusNotFound)
				return
			}

			result, err := s.PerformCheck()

			// update stored site
			s.ResponseTime = result.ResponseTime
			s.IsUp = result.IsUp
			s.CheckCount++

			if err != nil {
				s.Status = "down"
			} else {
				s.Status = fmt.Sprintf("%d", result.StatusCode)
			}

			writeJSON(w, http.StatusOK, result)

		case http.MethodGet:
			s, exists := sites[id]
			if !exists {
				http.Error(w, "site not found", http.StatusNotFound)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"id":            id,
				"url":           s.URL,
				"status":        s.Status,
				"response_time": s.ResponseTime,
				"is_up":         s.IsUp,
				"check_count":   s.CheckCount,
			})

		case http.MethodDelete:
			_, exists := sites[id]
			if !exists {
				http.Error(w, "site not found", http.StatusNotFound)
				return
			}

			delete(sites, id)
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

			id := fmt.Sprintf("site_%d", counterID)

			s := &site.Site{
				URL:    input.URL,
				Status: "Unknown",
			}

			sites[id] = s
			counterID++

			writeJSON(w, http.StatusCreated, map[string]any{
				"id":     id,
				"url":    s.URL,
				"status": s.Status,
			})

		case http.MethodGet:
			result := make([]map[string]any, 0)
			for id, s := range sites {
				result = append(result, map[string]any{
					"id":            id,
					"url":           s.URL,
					"status":        s.Status,
					"response_time": s.ResponseTime,
					"is_up":         s.IsUp,
					"check_count":   s.CheckCount,
				})
			}
			writeJSON(w, http.StatusOK, result)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("Server is running on :8081")
	http.ListenAndServe(":8081", nil)

}
