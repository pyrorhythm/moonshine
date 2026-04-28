package backend

import (
	"context"
	"io"
)

type outputKey struct{}

// WithOutput attaches w to ctx so backends can stream subprocess output to it.
func WithOutput(ctx context.Context, w io.Writer) context.Context {
	return context.WithValue(ctx, outputKey{}, w)
}

// OutputFrom returns the output writer stored in ctx and whether one was set.
// When no writer is set, backends should fall back to their default (os.Stdout).
func OutputFrom(ctx context.Context) (io.Writer, bool) {
	w, ok := ctx.Value(outputKey{}).(io.Writer)
	return w, ok && w != nil
}
