package margo

import (
	"context"
	"io"
)

// CommitOutcome describes what is known about a destination after an artifact
// sink returns. Sinks must never collapse an uncertain filesystem state into a
// successful commit or a claim that the destination is unchanged.
type CommitOutcome string

const (
	CommitNotCommitted        CommitOutcome = "not_committed"
	CommitCommitted           CommitOutcome = "committed"
	CommitDurabilityUncertain CommitOutcome = "durability_uncertain"
	CommitUnknown             CommitOutcome = "unknown"
)

// CommitResult is the transport identity returned by an ArtifactSink.
type CommitResult struct {
	Outcome CommitOutcome
	Target  string
	Digest  ArtifactDigest
	Bytes   int64
}

// ArtifactSink publishes a completely staged artifact. Implementations must
// not make destination bytes visible until the input has passed all
// pre-publication validation owned by the caller.
type ArtifactSink interface {
	Commit(context.Context, io.Reader, ArtifactDigest) (CommitResult, error)
}
