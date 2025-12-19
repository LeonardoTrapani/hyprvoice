package injection

import (
	"context"
	"time"
)

// Backend represents a text injection method
type Backend interface {
	Name() string
	Available() error
	Inject(ctx context.Context, text string, timeout time.Duration) error
}
