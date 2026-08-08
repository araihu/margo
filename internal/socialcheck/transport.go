package socialcheck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// FetchHTML reads an explicit file or redirect-free HTTPS URL and verifies an
// optional expected digest before returning bytes.
func FetchHTML(ctx context.Context, location, expectedSHA256 string) ([]byte, error) {
	data, err := fetch(ctx, location)
	if err != nil {
		return nil, err
	}
	if expectedSHA256 != "" {
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != expectedSHA256 {
			return nil, fmt.Errorf("socialcheck: input digest mismatch")
		}
	}
	return data, nil
}

func FetchPreview(ctx context.Context, location, expectedSHA256 string, width, height int, maxBytes int64) ([]byte, error) {
	data, err := FetchHTML(ctx, location, expectedSHA256)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) >= maxBytes {
		return nil, fmt.Errorf("socialcheck: preview exceeds byte bound")
	}
	if err := RequirePNGBytes(data, width, height); err != nil {
		return nil, err
	}
	return data, nil
}

func RequirePNGBytes(data []byte, width, height int) error {
	temporary, err := os.CreateTemp("", "margo-preview-*.png")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return RequirePNG(name, width, height, int64(len(data))+1)
}

func fetch(ctx context.Context, location string) ([]byte, error) {
	if parsed, err := url.Parse(location); err == nil && parsed.Scheme == "https" {
		if parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, fmt.Errorf("socialcheck: unsafe HTTPS input")
		}
		client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
		if err != nil {
			return nil, err
		}
		response, err := client.Do(request)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("socialcheck: HTTP status %d", response.StatusCode)
		}
		return io.ReadAll(io.LimitReader(response.Body, 4<<20))
	}
	if location == "" {
		return nil, fmt.Errorf("socialcheck: input path required")
	}
	return os.ReadFile(location)
}
