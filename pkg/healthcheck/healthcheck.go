package healthcheck

import context "context"

type HealthZ interface {
	Ping(ctx context.Context) error
}
