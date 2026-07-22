package appapi

import "testing"

func TestActorPeriod(t *testing.T) {
	if ActorPeriod("day") != "daily" || ActorPeriod("week") != "weekly" || ActorPeriod("month") != "monthly" {
		t.Fatal("mapping")
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
