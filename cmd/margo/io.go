package main

import (
	"context"
	"fmt"
	"io"
	"os"

	margo "github.com/araihu/margo"
)

type SourceReader interface {
	ReadFile(string) ([]byte, error)
}

type osSourceReader struct{}

func (osSourceReader) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

type outputOptions struct {
	Path  string
	Force bool
}

func readInput(ctx context.Context, reader SourceReader, stdin io.Reader, path string) (margo.Source, error) {
	if ctx == nil {
		return margo.Source{}, fmt.Errorf("cli.context_required: input context is required")
	}
	if err := ctx.Err(); err != nil {
		return margo.Source{}, err
	}
	if path == "-" {
		if stdin == nil {
			return margo.Source{}, fmt.Errorf("cli.input_invalid: stdin is unavailable")
		}
		data, err := io.ReadAll(io.LimitReader(stdin, margo.MaxDocumentBytes+1))
		if err != nil {
			return margo.Source{}, fmt.Errorf("cli.input_read: %w", err)
		}
		if int64(len(data)) > margo.MaxDocumentBytes {
			return margo.Source{}, fmt.Errorf("cli.input_too_large: input exceeds %d bytes", margo.MaxDocumentBytes)
		}
		if err := ctx.Err(); err != nil {
			return margo.Source{}, err
		}
		return margo.Source{Name: "<stdin>", Content: data}, nil
	}
	if path == "" {
		return margo.Source{}, fmt.Errorf("cli.input_required: one input path or - is required")
	}
	if reader == nil {
		return margo.Source{}, fmt.Errorf("cli.input_reader_required: file reader is unavailable")
	}
	data, err := reader.ReadFile(path)
	if err != nil {
		return margo.Source{}, fmt.Errorf("cli.input_read: %w", err)
	}
	if int64(len(data)) > margo.MaxDocumentBytes {
		return margo.Source{}, fmt.Errorf("cli.input_too_large: input exceeds %d bytes", margo.MaxDocumentBytes)
	}
	if err := ctx.Err(); err != nil {
		return margo.Source{}, err
	}
	return margo.Source{Name: path, Content: append([]byte(nil), data...)}, nil
}

func publish(ctx context.Context, artifact []byte, options outputOptions, stdout io.Writer) (margo.CommitResult, error) {
	if options.Path == "" {
		return margo.CommitResult{}, fmt.Errorf("cli.output_required: --output is required")
	}
	spool := margo.NewSpool(margo.SpoolOptions{})
	defer spool.Close()
	if err := spool.WriteAll(ctx, artifact); err != nil {
		return margo.CommitResult{}, err
	}
	replay, err := spool.Reader()
	if err != nil {
		return margo.CommitResult{}, err
	}
	defer replay.Close()
	if options.Path == "-" {
		return (margo.StdoutSink{Writer: stdout}).Commit(ctx, replay, spool.Digest())
	}
	return (&margo.AtomicFileSink{Target: options.Path, Force: options.Force}).Commit(ctx, replay, spool.Digest())
}
