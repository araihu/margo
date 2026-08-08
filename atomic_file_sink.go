package margo

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AtomicFileSink stages one complete artifact beside its destination and
// publishes it with the platform's atomic no-replace primitive. O2 never
// replaces an existing destination; force replacement is owned by O3.
type AtomicFileSink struct {
	Target string
	Force  bool
	ops    atomicOps
}

type atomicOps interface {
	createStage(dir string) (*os.File, error)
	publishNoReplace(stage, target string) (visible bool, err error)
	remove(path string) error
	syncParent(dir string) error
}

type targetSnapshot struct {
	exists bool
	digest ArtifactDigest
}

// Commit implements ArtifactSink. Before the visibility linearization point,
// every failure is not_committed and the destination is left untouched. An
// ambiguous platform result is classified with a read-back instead of being
// guessed as a failed publication.
func (s *AtomicFileSink) Commit(ctx context.Context, r io.Reader, expected ArtifactDigest) (CommitResult, error) {
	result := CommitResult{Outcome: CommitNotCommitted, Target: s.Target, Digest: expected}
	if ctx == nil {
		return result, errors.New("margo.atomic.context_required")
	}
	if r == nil {
		return result, errors.New("margo.atomic.reader_required")
	}
	if s.Target == "" {
		return result, errors.New("margo.atomic.target_required")
	}
	if s.Force {
		return result, errors.New("margo.atomic.force_not_ready")
	}

	target, err := filepath.Abs(s.Target)
	if err != nil {
		return result, fmt.Errorf("margo.atomic.target_absolute: %w", err)
	}
	result.Target = target
	dir := filepath.Dir(target)

	prior, err := snapshotTarget(target)
	if err != nil {
		return result, fmt.Errorf("margo.atomic.prior_snapshot: %w", err)
	}
	if prior.exists {
		return result, errors.New("margo.atomic.destination_exists")
	}

	ops := s.ops
	if ops == nil {
		ops = defaultAtomicOps()
	}
	stage, err := ops.createStage(dir)
	if err != nil {
		return result, fmt.Errorf("margo.atomic.stage_create: %w", err)
	}
	stagePath := stage.Name()
	stageOwned := true
	defer func() {
		if stageOwned {
			_ = ops.remove(stagePath)
		}
	}()

	digest, size, err := writeStage(ctx, stage, r)
	result.Bytes = size
	if err != nil {
		_ = stage.Close()
		return result, err
	}
	if err := stage.Sync(); err != nil {
		_ = stage.Close()
		return result, fmt.Errorf("margo.atomic.stage_sync: %w", err)
	}
	if err := stage.Close(); err != nil {
		return result, fmt.Errorf("margo.atomic.stage_close: %w", err)
	}
	if digest != expected {
		return result, fmt.Errorf("margo.atomic.digest_mismatch: got %s want %s", digest, expected)
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}

	// From this call through read-back and parent synchronization, cancellation
	// is observed only after the actual filesystem outcome is known.
	visible, publishErr := ops.publishNoReplace(stagePath, target)
	if publishErr != nil || !visible {
		outcome := classifyPublishFailure(prior, target, expected, visible)
		result.Outcome = outcome
		if publishErr != nil {
			return result, fmt.Errorf("margo.atomic.publish: %w", publishErr)
		}
		return result, errors.New("margo.atomic.publish_not_visible")
	}
	stageOwned = false

	actual, err := snapshotTarget(target)
	if err != nil {
		result.Outcome = CommitUnknown
		return result, fmt.Errorf("margo.atomic.readback: %w", err)
	}
	if !actual.exists || actual.digest != expected {
		result.Outcome = CommitUnknown
		return result, errors.New("margo.atomic.readback_mismatch")
	}
	if err := ops.syncParent(dir); err != nil {
		result.Outcome = CommitDurabilityUncertain
		return result, fmt.Errorf("margo.atomic.parent_sync: %w", err)
	}
	result.Outcome = CommitCommitted
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func writeStage(ctx context.Context, stage io.Writer, r io.Reader) (ArtifactDigest, int64, error) {
	hash := sha256.New()
	buffer := make([]byte, 32*1024)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return ArtifactDigest{}, total, err
		}
		n, readErr := r.Read(buffer)
		if n > 0 {
			if err := ctx.Err(); err != nil {
				return ArtifactDigest{}, total, err
			}
			written, writeErr := stage.Write(buffer[:n])
			if writeErr != nil {
				return ArtifactDigest{}, total, fmt.Errorf("margo.atomic.stage_write: %w", writeErr)
			}
			if written != n {
				return ArtifactDigest{}, total, io.ErrShortWrite
			}
			if _, err := hash.Write(buffer[:n]); err != nil {
				return ArtifactDigest{}, total, fmt.Errorf("margo.atomic.digest_write: %w", err)
			}
			total += int64(n)
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return ArtifactDigest(hash.Sum(nil)), total, nil
			}
			return ArtifactDigest{}, total, fmt.Errorf("margo.atomic.reader: %w", readErr)
		}
	}
}

func snapshotTarget(path string) (targetSnapshot, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return targetSnapshot{}, nil
		}
		return targetSnapshot{}, err
	}
	if !info.Mode().IsRegular() {
		return targetSnapshot{exists: true}, fmt.Errorf("margo.atomic.destination_not_regular: %s", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return targetSnapshot{exists: true}, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return targetSnapshot{exists: true}, err
	}
	return targetSnapshot{exists: true, digest: ArtifactDigest(hash.Sum(nil))}, nil
}

func classifyPublishFailure(prior targetSnapshot, target string, expected ArtifactDigest, visible bool) CommitOutcome {
	actual, err := snapshotTarget(target)
	if err != nil {
		return CommitUnknown
	}
	if visible {
		if actual.exists && actual.digest == expected {
			return CommitDurabilityUncertain
		}
		return CommitUnknown
	}
	if actual.exists == prior.exists && (!actual.exists || actual.digest == prior.digest) {
		return CommitNotCommitted
	}
	if actual.exists && actual.digest == expected {
		return CommitUnknown
	}
	return CommitUnknown
}
