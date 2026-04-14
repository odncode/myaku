package site

import (
	"errors"
	"net/http"
	"time"
)

type Site struct {
	URL          string  `json:"url"`
	Status       string  `json:"status"`
	ResponseTime float64 `json:"response_time"`
	IsUp         bool    `json:"is_up"`
	CheckCount   int     `json:"check_count"`
}

// CheckResult is the raw output of an HTTP check operation.
//
// DESIGN ROLE:
// This struct represents the "data layer" of the system.
// It is intentionally free of business rules so it can be:
// - safely passed around between functions
// - stored in memory or disk
// - used for reporting without recomputation
//
// SYSTEM FLOW:
// HTTP request → CheckResult → Site.Update() → Site state + reporting
//
// KEY IDEA:
// This is immutable "evidence" of what happened during a check.
type CheckResult struct {
	URL          string    `json:"url"`
	StatusCode   int       `json:"status_code"`
	ResponseTime float64   `json:"response_time"`
	IsUp         bool      `json:"is_up"`
	CheckedAt    time.Time `json:"checked_at"`
}

// PerformCheck is the boundary between the system and the outside world (HTTP).
//
// DESIGN ROLE:
// This function is responsible for I/O only:
// - sending network requests
// - measuring latency
// - capturing raw HTTP responses
//
// It MUST NOT:
// - decide business logic (healthy/down rules)
// - modify Site state
//
// SYSTEM FLOW:
// Site (input) → HTTP request → CheckResult (output)
//
// WHY THIS EXISTS:
// It isolates unpredictable external systems (network) from application logic.
// This makes testing and reasoning about the system easier.
func (s *Site) PerformCheck() (CheckResult, error) {
	start := time.Now()

	resp, err := http.Get(s.URL)
	elapsed := time.Since(start)

	if err != nil {
		return CheckResult{
			URL:          s.URL,
			StatusCode:   0,
			ResponseTime: elapsed.Seconds(),
			IsUp:         false,
			CheckedAt:    time.Now(),
		}, err
	}

	defer resp.Body.Close()

	status := resp.StatusCode
	isUp := status >= 200 && status < 300

	return CheckResult{
		URL:          s.URL,
		StatusCode:   status,
		ResponseTime: elapsed.Seconds(),
		IsUp:         isUp,
		CheckedAt:    time.Now(),
	}, nil
}

// Update applies a CheckResult to the internal Site state.
//
// DESIGN ROLE:
// This is the "business logic layer" of the system.
// It interprets raw data and updates the system's current state.
//
// RESPONSIBILITIES:
// - convert StatusCode → IsUp
// - update human-readable Status
// - track number of checks performed
// - store latest response time
//
// SYSTEM FLOW:
// CheckResult (raw truth) → Update → Site (current state)
//
// WHY THIS EXISTS:
// Separates "what happened" (CheckResult) from "what it means" (Site state).
// This allows consistent logic regardless of where the result came from.

func (s *Site) Update(result CheckResult) error {
	if result.StatusCode == 0 {
		return errors.New("request failed")
	}

	s.ResponseTime = result.ResponseTime
	s.IsUp = result.StatusCode >= 200 && result.StatusCode < 300
	s.CheckCount++

	if s.IsUp {
		s.Status = "healthy"
	} else {
		s.Status = "down"
	}

	return nil
}

func (s *Site) Reset() {
	s.Status = "unknown"
	s.IsUp = false
	s.ResponseTime = 0
	s.CheckCount = 0
}
