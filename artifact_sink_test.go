package margo

import (
	"context"
	"io"
	"testing"
)

func TestCommitOutcomeValuesAreStable(t *testing.T) {
	for value, want := range map[CommitOutcome]string{
		CommitNotCommitted:        "not_committed",
		CommitCommitted:           "committed",
		CommitDurabilityUncertain: "durability_uncertain",
		CommitUnknown:             "unknown",
	} {
		if string(value) != want {
			t.Errorf("CommitOutcome %q = %q, want %q", value, string(value), want)
		}
	}
}

func TestArtifactSinkContractCarriesDigest(t *testing.T) {
	var _ ArtifactSink = recordingSink{}
	digest := ArtifactDigestOf([]byte("artifact"))
	result, err := recordingSink{}.Commit(context.Background(), nil, digest)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != CommitCommitted || result.Digest != digest {
		t.Fatalf("result = %#v, want committed result with digest %s", result, digest)
	}
}

type recordingSink struct{}

func (recordingSink) Commit(_ context.Context, _ io.Reader, digest ArtifactDigest) (CommitResult, error) {
	return CommitResult{Outcome: CommitCommitted, Digest: digest}, nil
}
