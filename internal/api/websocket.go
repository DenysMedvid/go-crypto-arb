package api

import "context"

type SnapshotBroadcaster interface {
	BroadcastSnapshot(ctx context.Context) error
}

// TODO: Add a WebSocket broadcaster for frontend clients after the REST snapshot
// schema settles. The TUI intentionally uses REST only in v1.
