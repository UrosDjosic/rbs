package httpapi

import (
	"oblak/internal/common/ids"
)

func mustID() string {
	s, err := ids.NewToken(16)
	if err != nil {
		// extremely unlikely; if it happens, fall back to empty which will fail insert
		return ""
	}
	return s
}
