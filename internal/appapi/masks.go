package appapi

import (
	"fmt"
	"strings"
)

// MainFlags are valid main-attribute letters in filter_by masks.
var MainFlags = map[string]bool{
	"p": true, "m": true, "c": true, "s": true, "i": true, "v": true,
}

// EntityLetters maps entity kind → filter_by letter.
var EntityLetters = map[string]string{
	"actor":    "a",
	"series":   "s",
	"maker":    "m",
	"director": "d",
	"code":     "c",
	"list":     "l",
}

// BuildTagFilter builds filter_by for category browse: {zone}:t:{main}:{tags}:{year}:{month}:
func BuildTagFilter(zone string, main, tags []string, year, month string) (string, error) {
	z, ok := Zones[zone]
	if !ok {
		return "", fmt.Errorf("zone must be one of censored|uncensored|western|fc2")
	}
	mainS := strings.Join(main, ",")
	tagsS := strings.Join(tags, ",")
	return fmt.Sprintf("%d:t:%s:%s:%s:%s:", z, mainS, tagsS, year, month), nil
}

// BuildEntityFilter builds filter_by for entity filmography.
// {zone}:{letter}:{id} or {zone}:{letter}:{id}:{main}::
func BuildEntityFilter(kind, entityID, zone string, main []string) (string, error) {
	z, ok := Zones[zone]
	if !ok {
		return "", fmt.Errorf("zone must be one of censored|uncensored|western|fc2")
	}
	letter, ok := EntityLetters[kind]
	if !ok {
		return "", fmt.Errorf("unknown entity kind: %s", kind)
	}
	base := fmt.Sprintf("%d:%s:%s", z, letter, entityID)
	if len(main) > 0 {
		return base + ":" + strings.Join(main, ",") + "::", nil
	}
	return base, nil
}
