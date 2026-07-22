package appapi

// FilterMagnets applies client-side cnsub/hd/min_size filters.
func FilterMagnets(magnets []map[string]any, cnsub, hd bool, minSize int) []map[string]any {
	out := make([]map[string]any, 0, len(magnets))
	for _, m := range magnets {
		if cnsub && !truthy(m["cnsub"]) {
			continue
		}
		if hd && !truthy(m["hd"]) {
			continue
		}
		if minSize > 0 && anyInt(m["size"]) < minSize {
			continue
		}
		out = append(out, m)
	}
	return out
}

// PickBestMagnet prefers cnsub, then hd, then larger size, then files_count.
func PickBestMagnet(magnets []map[string]any) map[string]any {
	if len(magnets) == 0 {
		return nil
	}
	best := magnets[0]
	for _, m := range magnets[1:] {
		if magnetBetter(m, best) {
			best = m
		}
	}
	return best
}

func magnetBetter(a, b map[string]any) bool {
	ac, bc := boolScore(a["cnsub"]), boolScore(b["cnsub"])
	if ac != bc {
		return ac > bc
	}
	ah, bh := boolScore(a["hd"]), boolScore(b["hd"])
	if ah != bh {
		return ah > bh
	}
	as, bs := anyInt(a["size"]), anyInt(b["size"])
	if as != bs {
		return as > bs
	}
	return anyInt(a["files_count"]) > anyInt(b["files_count"])
}

func boolScore(v any) int {
	if truthy(v) {
		return 1
	}
	return 0
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case int:
		return t != 0
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}

func anyInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case string:
		n := 0
		for _, c := range t {
			if c < '0' || c > '9' {
				break
			}
			n = n*10 + int(c-'0')
		}
		return n
	default:
		return 0
	}
}

// MagnetURI builds magnet:?xt=urn:btih:… from a row.
func MagnetURI(m map[string]any) string {
	if m == nil {
		return ""
	}
	h := anyStr(m["hash"])
	if h == "" {
		return ""
	}
	return "magnet:?xt=urn:btih:" + h
}
