package appapi

import "testing"

func TestBuildSearchParamsDefaults(t *testing.T) {
	p := BuildSearchParams("SSIS", SearchOptions{})
	if p["q"] != "SSIS" || p["page"] != "1" {
		t.Fatalf("%v", p)
	}
	if _, ok := p["movie_type"]; ok {
		t.Fatalf("empty zone must omit movie_type: %v", p)
	}
}

func TestBuildSearchParamsZoneAndFilters(t *testing.T) {
	p := BuildSearchParams("SSIS", SearchOptions{
		Page: 2, Limit: 5, Zone: "censored", Sort: "release",
		FilterBy: "magnets", Type: "movie",
	})
	want := map[string]string{
		"q": "SSIS", "page": "2", "limit": "5",
		"movie_type": "0", "movie_sort_by": "release",
		"movie_filter_by": "magnets", "type": "movie",
	}
	for k, v := range want {
		if p[k] != v {
			t.Fatalf("%s: got %q want %q full=%v", k, p[k], v, p)
		}
	}
}

func TestBuildSearchParamsZoneAllOmitsMovieType(t *testing.T) {
	p := BuildSearchParams("x", SearchOptions{Zone: "all", Page: 1})
	if _, ok := p["movie_type"]; ok {
		t.Fatalf("all must omit movie_type: %v", p)
	}
}

func TestBuildSearchParamsUncensored(t *testing.T) {
	p := BuildSearchParams("heyzo", SearchOptions{Zone: "uncensored"})
	if p["movie_type"] != "1" {
		t.Fatalf("%v", p)
	}
}
