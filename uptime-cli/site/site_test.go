package site

import "testing"

func TestPerformCheckValidURL(t *testing.T) {
	site := Site{URL: "https://google.com"}

	result, err := site.PerformCheck()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.URL != site.URL {
		t.Errorf("expected URL %v, got %v", site.URL, result.URL)
	}

	if result.ResponseTime <= 0 {
		t.Errorf("expected response time > 0")
	}
}

func TestPerformCheckEmptyURL(t *testing.T) {
	site := Site{URL: ""}

	_, err := site.PerformCheck()

	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestPerformCheckBadPrefix(t *testing.T) {
	site := Site{URL: "google.com"}

	_, err := site.PerformCheck()

	if err == nil {
		t.Fatal("expected error for missing http/https")
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name       string
		input      CheckResult
		wantIsUp   bool
		wantStatus string
		wantErr    bool
	}{
		{
			name: "success case",
			input: CheckResult{
				StatusCode: 200,
			},
			wantIsUp:   true,
			wantStatus: "healthy",
			wantErr:    false,
		},
		{
			name: "failure case",
			input: CheckResult{
				StatusCode: 500,
			},
			wantIsUp:   false,
			wantStatus: "down",
			wantErr:    false,
		},
		{
			name: "invalid case",
			input: CheckResult{
				StatusCode: 0,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := Site{}

			err := s.Update(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if s.IsUp != tc.wantIsUp {
				t.Errorf("expected IsUp %v, got %v", tc.wantIsUp, s.IsUp)
			}

			if s.Status != tc.wantStatus {
				t.Errorf("expected Status %v, got %v", tc.wantStatus, s.Status)
			}
		})
	}
}

func TestReset(t *testing.T) {
	site := Site{
		URL:    "https://google.com",
		Status: "healthy",
		IsUp:   true,
	}

	site.Reset()

	if site.Status != "unknown" {
		t.Errorf("expected unknown, got %v", site.Status)
	}

	if site.IsUp != false {
		t.Errorf("expected false, got %v", site.IsUp)
	}

	if site.ResponseTime != 0 {
		t.Errorf("expected 0, got %v", site.ResponseTime)
	}

	if site.CheckCount != 0 {
		t.Errorf("expected 0, got %v", site.CheckCount)
	}
}