package appapi

import "testing"

func TestBuildTagFilter(t *testing.T) {
	got, err := BuildTagFilter("censored", nil, nil, "", "")
	if err != nil || got != "0:t:::::" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = BuildTagFilter("censored", nil, []string{"17"}, "", "")
	if err != nil || got != "0:t::17:::" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = BuildTagFilter("censored", nil, []string{"17", "18"}, "2023", "")
	if err != nil || got != "0:t::17,18:2023::" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = BuildTagFilter("censored", []string{"m", "c"}, []string{"17"}, "2023", "1")
	if err != nil || got != "0:t:m,c:17:2023:1:" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestBuildEntityFilter(t *testing.T) {
	got, err := BuildEntityFilter("actor", "9Dqpw", "censored", nil)
	if err != nil || got != "0:a:9Dqpw" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = BuildEntityFilter("actor", "9Dqpw", "censored", []string{"m"})
	if err != nil || got != "0:a:9Dqpw:m::" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = BuildEntityFilter("list", "RZ8Bm", "censored", nil)
	if err != nil || got != "0:l:RZ8Bm" {
		t.Fatalf("%q %v", got, err)
	}
}
