package margo

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// StdoutSink copies only a completely validated spool to its writer. Unlike a
// filesystem sink, stdout cannot revoke bytes after a downstream short write.
type StdoutSink struct {
	Writer io.Writer
}

func (s StdoutSink) Commit(ctx context.Context, r io.Reader, expected ArtifactDigest) (CommitResult, error) {
	result := CommitResult{Outcome: CommitNotCommitted, Target: "stdout", Digest: expected}
	if ctx == nil {
		return result, errors.New("margo.stdout.context_required")
	}
	if r == nil {
		return result, errors.New("margo.stdout.reader_required")
	}
	if s.Writer == nil {
		return result, errors.New("margo.stdout.writer_required")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}

	spool := NewSpool(SpoolOptions{MaximumBytes: MaxDocumentBytes})
	defer func() { _ = spool.Close() }()
	if err := fillSpool(ctx, spool, r); err != nil {
		return result, err
	}
	if got := spool.Digest(); got != expected {
		return result, fmt.Errorf("margo.stdout.digest_mismatch: got %s want %s", got, expected)
	}

	reader, err := spool.Reader()
	if err != nil {
		return result, fmt.Errorf("margo.stdout.replay: %w", err)
	}
	defer reader.Close()
	result, err = copyValidatedStdout(ctx, s.Writer, reader, result)
	return result, err
}

func fillSpool(ctx context.Context, spool *Spool, r io.Reader) error {
	buffer := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := r.Read(buffer)
		if n > 0 {
			if err := spool.WriteAll(ctx, buffer[:n]); err != nil {
				return err
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return fmt.Errorf("margo.stdout.reader: %w", readErr)
		}
	}
}

func copyValidatedStdout(ctx context.Context, writer io.Writer, reader io.Reader, result CommitResult) (CommitResult, error) {
	buffer := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			if result.Bytes == 0 {
				result.Outcome = CommitNotCommitted
			} else {
				result.Outcome = CommitUnknown
			}
			return result, err
		}
		n, readErr := reader.Read(buffer)
		if n > 0 {
			written, writeErr := writer.Write(buffer[:n])
			result.Bytes += int64(written)
			if writeErr != nil {
				result.Outcome = CommitUnknown
				return result, fmt.Errorf("margo.stdout.partial_write: wrote %d of %d: %w", written, n, writeErr)
			}
			if written != n {
				result.Outcome = CommitUnknown
				return result, fmt.Errorf("margo.stdout.partial_write: wrote %d of %d: %w", written, n, io.ErrShortWrite)
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				result.Outcome = CommitCommitted
				if err := ctx.Err(); err != nil {
					return result, err
				}
				return result, nil
			}
			result.Outcome = CommitUnknown
			return result, fmt.Errorf("margo.stdout.replay_read: %w", readErr)
		}
	}
}
