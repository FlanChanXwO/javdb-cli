// Package signature implements the JavDB app jdsignature header.
//
// Reverse-engineered from JavDB.apk 1.9.28; golden vector verified 2026-07-16.
//
//	jdsignature = "{ts}.{suffix}.{md5(ts + prefix)}"
package signature

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
)

// Precomputed from access key "30820" + CONST_PREFIX/CONST_SUFFIX in the app.
// Only the timestamp changes per request.
const (
	Prefix = "71cf27bb3c0bcdf207b64abecddc970098c7421ee7203b9cdae54478478a199e7d5a6e1a57691123c1a931c057842fb73ba3b3c83bcd69c17ccf174081e3d8aa"
	Suffix = "lpw6vgqzsp"
)

// Sign returns a jdsignature header value for the given unix second.
// If ts <= 0, the current time is used.
func Sign(ts int64) string {
	if ts <= 0 {
		ts = time.Now().Unix()
	}
	sum := md5.Sum([]byte(fmt.Sprintf("%d%s", ts, Prefix)))
	return fmt.Sprintf("%d.%s.%s", ts, Suffix, hex.EncodeToString(sum[:]))
}
