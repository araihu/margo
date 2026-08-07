package authority

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/araihu/margo/internal/canonicaljson"
)

type CanonicalOrigin string

type AuthorityRoutes struct {
	Homepage       string `json:"homepage"`
	Representative string `json:"representative"`
	Preview        string `json:"preview"`
}

type AuthorityOwner struct {
	Principal string `json:"principal"`
	Contact   string `json:"contact"`
}

type AuthorityAsset struct {
	URL            string `json:"url"`
	MIMEType       string `json:"mimeType"`
	SHA256         string `json:"sha256"`
	Bytes          int64  `json:"bytes"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	EvidenceURL    string `json:"evidenceURL"`
	EvidenceSHA256 string `json:"evidenceSHA256"`
}

type AuthorityEvidence struct {
	Provider       string `json:"provider"`
	Account        string `json:"account"`
	Resource       string `json:"resource"`
	EvidenceURL    string `json:"evidenceURL"`
	EvidenceSHA256 string `json:"evidenceSHA256"`
}

type AuthorityDeployment struct {
	Provider       string `json:"provider"`
	Project        string `json:"project"`
	DeploymentID   string `json:"deploymentID"`
	Commit         string `json:"commit"`
	EvidenceURL    string `json:"evidenceURL"`
	EvidenceSHA256 string `json:"evidenceSHA256"`
}

type AuthorityReceipt struct {
	Transport  string `json:"transport"`
	Source     string `json:"source"`
	SHA256     string `json:"sha256"`
	Verifier   string `json:"verifier"`
	VerifiedAt string `json:"verifiedAt"`
}

type AuthorityRecord struct {
	SchemaVersion   string              `json:"schemaVersion"`
	RecordDigest    string              `json:"recordDigest"`
	CanonicalOrigin CanonicalOrigin     `json:"canonicalOrigin"`
	Routes          AuthorityRoutes     `json:"routes"`
	Owner           AuthorityOwner      `json:"owner"`
	Asset           AuthorityAsset      `json:"asset"`
	Host            AuthorityEvidence   `json:"host"`
	Deployment      AuthorityDeployment `json:"deployment"`
	Receipt         AuthorityReceipt    `json:"receipt"`
}

// Validate checks the structural authority contract and recordDigest.
func (record AuthorityRecord) Validate() error {
	if err := validateRecord(record); err != nil {
		return err
	}
	preimage, err := digestPreimage(record)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(preimage)
	if record.RecordDigest != hex.EncodeToString(hash[:]) {
		return fmt.Errorf("authority.record_digest_mismatch")
	}
	return nil
}

type AuthoritySource interface {
	Read(context.Context, string) ([]byte, AuthorityReceipt, error)
}

// FileSource reads only an explicit file path and computes its receipt digest.
type FileSource struct{}

func (FileSource) Read(_ context.Context, source string) ([]byte, AuthorityReceipt, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, AuthorityReceipt{}, err
	}
	hash := sha256.Sum256(data)
	return data, AuthorityReceipt{Transport: "file", Source: source, SHA256: hex.EncodeToString(hash[:]), Verifier: "margo/file-source-v1"}, nil
}

// HTTPSSource fetches one URL with redirects disabled and a bounded response.
type HTTPSSource struct{ Client *http.Client }

func (s HTTPSSource) Read(ctx context.Context, source string) ([]byte, AuthorityReceipt, error) {
	parsed, err := url.Parse(source)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, AuthorityReceipt{}, fmt.Errorf("authority.transport_url_invalid: %s", source)
	}
	client := s.Client
	if client == nil {
		client = &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, AuthorityReceipt{}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, AuthorityReceipt{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, AuthorityReceipt{}, fmt.Errorf("authority.transport_status: %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, AuthorityReceipt{}, err
	}
	hash := sha256.Sum256(data)
	return data, AuthorityReceipt{Transport: "https", Source: source, SHA256: hex.EncodeToString(hash[:]), Verifier: "margo/https-source-v1"}, nil
}

func VerifyAuthorityRecord(data []byte) (AuthorityRecord, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var record AuthorityRecord
	if err := decoder.Decode(&record); err != nil {
		return AuthorityRecord{}, fmt.Errorf("authority.record_invalid: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return AuthorityRecord{}, fmt.Errorf("authority.record_trailing_data")
	}
	if err := validateRecord(record); err != nil {
		return AuthorityRecord{}, err
	}
	preimage, err := digestPreimage(record)
	if err != nil {
		return AuthorityRecord{}, err
	}
	hash := sha256.Sum256(preimage)
	if record.RecordDigest != hex.EncodeToString(hash[:]) {
		return AuthorityRecord{}, fmt.Errorf("authority.record_digest_mismatch")
	}
	return record, nil
}

func LoadAuthorityRecord(ctx context.Context, source AuthoritySource, location string) (AuthorityRecord, error) {
	if source == nil || strings.TrimSpace(location) == "" {
		return AuthorityRecord{}, fmt.Errorf("authority.source_required")
	}
	data, receipt, err := source.Read(ctx, location)
	if err != nil {
		return AuthorityRecord{}, err
	}
	record, err := VerifyAuthorityRecord(data)
	if err != nil {
		return AuthorityRecord{}, err
	}
	if err := verifyReceiptIdentity(record.Receipt, receipt); err != nil {
		return AuthorityRecord{}, err
	}
	// The record receipt describes independently verified authority evidence,
	// not the record bytes themselves; requiring equality would be circular.
	return record, nil
}

func validateRecord(record AuthorityRecord) error {
	if record.SchemaVersion != "margo/canonical-authority/v1" {
		return fmt.Errorf("authority.schema_version_invalid")
	}
	origin, err := parseHTTPSOrigin(string(record.CanonicalOrigin))
	if err != nil {
		return fmt.Errorf("authority.origin_invalid: %w", err)
	}
	if record.Routes.Homepage != "/" || !isRoutePath(record.Routes.Representative) || record.Routes.Representative == "/" || record.Routes.Preview != "/assets/social/margo-v0.0.1.png" {
		return fmt.Errorf("authority.route_invalid")
	}
	assetURL, err := url.Parse(record.Asset.URL)
	if err != nil || assetURL.Scheme != "https" || assetURL.User != nil || assetURL.RawQuery != "" || assetURL.Fragment != "" || assetURL.Host != origin.Host || assetURL.Path != record.Routes.Preview {
		return fmt.Errorf("authority.origin_unlisted: asset is outside canonical origin")
	}
	if record.Asset.MIMEType != "image/png" && record.Asset.MIMEType != "image/jpeg" {
		return fmt.Errorf("authority.asset_mime_invalid")
	}
	if record.Asset.Bytes <= 0 || record.Asset.Bytes >= 1_000_000 || record.Asset.Width != 1280 || record.Asset.Height != 640 || !isSHA256(record.Asset.SHA256) || !isSHA256(record.Asset.EvidenceSHA256) {
		return fmt.Errorf("authority.asset_invalid")
	}
	if err := validateEvidence(record.Host); err != nil {
		return fmt.Errorf("authority.host_invalid: %w", err)
	}
	if record.Deployment.Provider == "" || record.Deployment.Project == "" || record.Deployment.DeploymentID == "" || record.Deployment.Commit == "" || !isHTTPS(record.Deployment.EvidenceURL) || !isSHA256(record.Deployment.EvidenceSHA256) {
		return fmt.Errorf("authority.deployment_invalid")
	}
	if record.Owner.Principal == "" || record.Owner.Contact == "" {
		return fmt.Errorf("authority.owner_invalid")
	}
	if record.Receipt.Transport != "file" && record.Receipt.Transport != "https" {
		return fmt.Errorf("authority.receipt_transport_invalid")
	}
	if record.Receipt.Source == "" || !isSHA256(record.Receipt.SHA256) || record.Receipt.Verifier == "" || record.Receipt.VerifiedAt == "" {
		return fmt.Errorf("authority.receipt_invalid")
	}
	return nil
}

func validateEvidence(evidence AuthorityEvidence) error {
	if evidence.Provider == "" || evidence.Account == "" || evidence.Resource == "" || !isHTTPS(evidence.EvidenceURL) || !isSHA256(evidence.EvidenceSHA256) {
		return fmt.Errorf("evidence fields are incomplete")
	}
	return nil
}

func parseHTTPSOrigin(value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
		return nil, fmt.Errorf("must be an origin-only HTTPS URL")
	}
	return parsed, nil
}

func isHTTPS(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == ""
}

func isRoutePath(value string) bool {
	return strings.HasPrefix(value, "/") && !strings.ContainsAny(value, "?#") && !strings.Contains(value, "//")
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func digestPreimage(record AuthorityRecord) ([]byte, error) {
	data, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, err
	}
	delete(object, "recordDigest")
	return canonicaljson.Marshal(object)
}
