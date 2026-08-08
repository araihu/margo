package margo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const defaultSpoolMemoryLimit int64 = 1 << 20

// SpoolOptions controls the private staging boundary used before publication.
type SpoolOptions struct {
	MemoryLimit  int64
	MaximumBytes int64
	TempDir      string
}

// Spool accumulates one bounded artifact. It remains invisible to the caller's
// destination and spills to a mode-0600 private file after MemoryLimit.
type Spool struct {
	mu           sync.Mutex
	memory       bytes.Buffer
	tempFile     *os.File
	tempPath     string
	memoryLimit  int64
	maximumBytes int64
	total        int64
	digest       hash.Hash
	closed       bool
	invalid      bool
	tempDir      string
}

// NewSpool creates a private bounded staging buffer. Zero limits select safe
// defaults; invalid negative limits are reported by the first write.
func NewSpool(options SpoolOptions) *Spool {
	memoryLimit := options.MemoryLimit
	if memoryLimit == 0 {
		memoryLimit = defaultSpoolMemoryLimit
	}
	maximumBytes := options.MaximumBytes
	if maximumBytes == 0 {
		maximumBytes = MaxDocumentBytes
	}
	tempDir := options.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	return &Spool{
		memoryLimit:  memoryLimit,
		maximumBytes: maximumBytes,
		digest:       sha256.New(),
		tempDir:      filepath.Clean(tempDir),
		invalid:      memoryLimit < 1 || maximumBytes < 1,
	}
}

// WriteAll appends one bounded byte sequence and observes cancellation before
// any private stage is created or mutated.
func (s *Spool) WriteAll(ctx context.Context, data []byte) error {
	if ctx == nil {
		return errors.New("margo.spool.context_required")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return s.Write(data)
}

// Write appends bytes to the spool without exposing a destination.
func (s *Spool) Write(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("margo.spool.closed")
	}
	if s.invalid {
		return errors.New("margo.spool.options_invalid")
	}
	if int64(len(data)) > s.maximumBytes-s.total {
		return fmt.Errorf("margo.spool.maximum_exceeded: %d > %d", s.total+int64(len(data)), s.maximumBytes)
	}
	if len(data) == 0 {
		return nil
	}
	if s.tempFile == nil && s.total+int64(len(data)) <= s.memoryLimit {
		if _, err := s.memory.Write(data); err != nil {
			return fmt.Errorf("margo.spool.memory_write: %w", err)
		}
	} else {
		if err := s.ensurePrivateFileLocked(); err != nil {
			return err
		}
		if _, err := s.tempFile.Write(data); err != nil {
			return fmt.Errorf("margo.spool.private_write: %w", err)
		}
	}
	if _, err := s.digest.Write(data); err != nil {
		return fmt.Errorf("margo.spool.digest_write: %w", err)
	}
	s.total += int64(len(data))
	return nil
}

func (s *Spool) ensurePrivateFileLocked() error {
	if s.tempFile != nil {
		return nil
	}
	tempFile, err := os.CreateTemp(s.tempDir, ".margo-spool-*")
	if err != nil {
		return fmt.Errorf("margo.spool.private_create: %w", err)
	}
	if err := tempFile.Chmod(0o600); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return fmt.Errorf("margo.spool.private_mode: %w", err)
	}
	if _, err := tempFile.Write(s.memory.Bytes()); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return fmt.Errorf("margo.spool.private_seed: %w", err)
	}
	s.tempFile = tempFile
	s.tempPath = tempFile.Name()
	s.memory.Reset()
	return nil
}

// Reader returns a fresh replay reader positioned at byte zero.
func (s *Spool) Reader() (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, errors.New("margo.spool.closed")
	}
	if s.tempFile == nil {
		return io.NopCloser(bytes.NewReader(append([]byte(nil), s.memory.Bytes()...))), nil
	}
	if err := s.tempFile.Sync(); err != nil {
		return nil, fmt.Errorf("margo.spool.private_sync: %w", err)
	}
	reader, err := os.Open(s.tempPath)
	if err != nil {
		return nil, fmt.Errorf("margo.spool.private_open: %w", err)
	}
	return reader, nil
}

// Digest returns the exact-byte digest accumulated so far.
func (s *Spool) Digest() ArtifactDigest {
	s.mu.Lock()
	defer s.mu.Unlock()
	var digest ArtifactDigest
	copy(digest[:], s.digest.Sum(nil))
	return digest
}

// Size returns the number of staged bytes.
func (s *Spool) Size() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.total
}

// UsesPrivateFile reports whether the memory threshold has been crossed.
func (s *Spool) UsesPrivateFile() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tempFile != nil
}

// Close removes private staging and makes the spool unusable. It is safe to
// call more than once.
func (s *Spool) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	s.memory.Reset()
	var errs []error
	if s.tempFile != nil {
		if err := s.tempFile.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := os.Remove(s.tempPath); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
		}
		s.tempFile = nil
		s.tempPath = ""
	}
	return errors.Join(errs...)
}
