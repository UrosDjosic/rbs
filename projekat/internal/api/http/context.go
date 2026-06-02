package httpapi

import "context"

type ctxKey string

const (
	ctxUserID ctxKey = "userID"
)

func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxUserID, userID)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxUserID)
	s, ok := v.(string)
	return s, ok
}
