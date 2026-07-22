package javdb

import "github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi"

// Re-export magnet helpers for SDK callers.
func FilterMagnets(magnets []map[string]any, cnsub, hd bool, minSize int) []map[string]any {
	return appapi.FilterMagnets(magnets, cnsub, hd, minSize)
}

func PickBestMagnet(magnets []map[string]any) map[string]any {
	return appapi.PickBestMagnet(magnets)
}

func MagnetURI(m map[string]any) string {
	return appapi.MagnetURI(m)
}
