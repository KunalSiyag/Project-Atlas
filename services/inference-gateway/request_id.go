package inference

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}
	return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
}
