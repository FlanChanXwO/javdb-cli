package appapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRankingPeriod(t *testing.T) {
	if RankingPeriod("day") != "daily" || RankingPeriod("week") != "weekly" || RankingPeriod("month") != "monthly" {
		t.Fatal("RankingPeriod mapping")
	}
}

func TestBuildTop250Params(t *testing.T) {
	p, err := BuildTop250Params("", "", 1, 1, 20, false)
	if err != nil || p["type"] != "all" || p["type_value"] != "all" {
		t.Fatalf("%v %v", p, err)
	}
	p, err = BuildTop250Params("censored", "", 51, 1, 20, true)
	if err != nil || p["type"] != "video_type" || p["type_value"] != "0" || p["start_rank"] != "51" || p["ignore_watched"] != "true" {
		t.Fatalf("%v %v", p, err)
	}
	p, err = BuildTop250Params("censored", "2023", 1, 1, 20, false)
	if err != nil || p["type"] != "year" || p["type_value"] != "2023" {
		t.Fatalf("year should win: %v %v", p, err)
	}
}

func TestRankingsMoviesQueryMapping(t *testing.T) {
	var gotType, gotPeriod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotType = r.URL.Query().Get("type")
		gotPeriod = r.URL.Query().Get("period")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"movies":[]}}`))
	}))
	defer server.Close()

	client, err := New(Options{Host: server.URL})
	if err != nil {
		t.Fatalf("New client: %v", err)
	}

	tests := []struct {
		input      string
		period     string
		wantType   string
		wantPeriod string
	}{
		{"censored", "day", "0", "daily"},
		{"uncensored", "week", "1", "weekly"},
		{"western", "month", "2", "monthly"},
		{"fc2", "day", "3", "daily"},
		{"2", "daily", "2", "daily"},
	}

	for _, tt := range tests {
		_, err := client.RankingsMovies(tt.input, tt.period)
		if err != nil {
			t.Fatalf("RankingsMovies(%q, %q) error: %v", tt.input, tt.period, err)
		}
		if gotType != tt.wantType || gotPeriod != tt.wantPeriod {
			t.Errorf("RankingsMovies(%q, %q) got type=%q, period=%q; want type=%q, period=%q",
				tt.input, tt.period, gotType, gotPeriod, tt.wantType, tt.wantPeriod)
		}
	}
}

func TestRankingsPlaybackQueryMapping(t *testing.T) {
	var gotFilterBy, gotPeriod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFilterBy = r.URL.Query().Get("filter_by")
		gotPeriod = r.URL.Query().Get("period")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"movies":[]}}`))
	}))
	defer server.Close()

	client, err := New(Options{Host: server.URL})
	if err != nil {
		t.Fatalf("New client: %v", err)
	}

	tests := []struct {
		input      string
		period     string
		wantFilter string
		wantPeriod string
	}{
		{"western", "day", "2", "daily"},
		{"censored", "week", "0", "weekly"},
		{"1", "month", "1", "monthly"},
	}

	for _, tt := range tests {
		_, err := client.RankingsPlayback(tt.input, tt.period)
		if err != nil {
			t.Fatalf("RankingsPlayback(%q, %q) error: %v", tt.input, tt.period, err)
		}
		if gotFilterBy != tt.wantFilter || gotPeriod != tt.wantPeriod {
			t.Errorf("RankingsPlayback(%q, %q) got filter_by=%q, period=%q; want filter_by=%q, period=%q",
				tt.input, tt.period, gotFilterBy, gotPeriod, tt.wantFilter, tt.wantPeriod)
		}
	}
}
