package appapi

import "testing"

func TestFilterMagnets(t *testing.T) {
	mags := []map[string]any{
		{"hash": "a", "cnsub": true, "hd": true, "size": float64(100)},
		{"hash": "b", "cnsub": false, "hd": true, "size": float64(2000)},
		{"hash": "c", "cnsub": true, "hd": false, "size": float64(50)},
	}
	out := FilterMagnets(mags, true, false, 0)
	if len(out) != 2 || anyStr(out[0]["hash"]) != "a" {
		t.Fatalf("%v", out)
	}
	out = FilterMagnets(mags, true, true, 0)
	if len(out) != 1 || anyStr(out[0]["hash"]) != "a" {
		t.Fatalf("%v", out)
	}
	out = FilterMagnets(mags, false, false, 1000)
	if len(out) != 1 || anyStr(out[0]["hash"]) != "b" {
		t.Fatalf("%v", out)
	}
}

func TestPickBestMagnet(t *testing.T) {
	mags := []map[string]any{
		{"hash": "a", "cnsub": false, "hd": true, "size": float64(100)},
		{"hash": "b", "cnsub": true, "hd": true, "size": float64(200)},
		{"hash": "c", "cnsub": true, "hd": false, "size": float64(9999)},
	}
	best := PickBestMagnet(mags)
	if anyStr(best["hash"]) != "b" {
		t.Fatalf("%v", best)
	}
	if PickBestMagnet(nil) != nil {
		t.Fatal("empty")
	}
}

func TestMagnetURI(t *testing.T) {
	u := MagnetURI(map[string]any{"hash": "abc"})
	if u != "magnet:?xt=urn:btih:abc" {
		t.Fatal(u)
	}
}
