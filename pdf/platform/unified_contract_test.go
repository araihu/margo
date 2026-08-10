package platform

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRepositoryPlatformContractsUseUnifiedRootIdentity(t *testing.T) {
	contractsData, err := os.ReadFile("runner-contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var contracts struct {
		SchemaVersion string `json:"schemaVersion"`
		Runners       map[string]struct {
			Command          []string `json:"command"`
			OwnedSourcePaths []string `json:"ownedSourcePaths"`
			OwnedProbePaths  []string `json:"ownedProbePaths"`
		} `json:"runners"`
	}
	if err := json.Unmarshal(contractsData, &contracts); err != nil {
		t.Fatal(err)
	}
	if contracts.SchemaVersion != "margo/pdf-platform-contracts/v2" {
		t.Fatalf("schema = %q", contracts.SchemaVersion)
	}
	for id, runner := range contracts.Runners {
		if len(runner.Command) < 3 || runner.Command[2] != "./pdf/platform" {
			t.Fatalf("runner %q command = %v", id, runner.Command)
		}
		for _, path := range append(runner.OwnedSourcePaths, runner.OwnedProbePaths...) {
			if !strings.HasPrefix(path, "pdf/platform/") {
				t.Fatalf("runner %q path = %q", id, path)
			}
		}
	}
	lockData, err := os.ReadFile("../platform-toolchain.lock")
	if err != nil {
		t.Fatal(err)
	}
	lockText := string(lockData)
	for _, forbidden := range []string{"/private/tmp/", `"path": "github.com/araihu/margo",\n      "version"`, "v0.0.0-20260808231103-771f44908d14"} {
		if strings.Contains(lockText, forbidden) {
			t.Fatalf("lock contains forbidden identity %q", forbidden)
		}
	}
	if !strings.Contains(lockText, `"schemaVersion": "margo/pdf-platform-toolchain/v2"`) || !strings.Contains(lockText, `"modulePath": "github.com/araihu/margo"`) {
		t.Fatal("lock does not declare the unified root module")
	}
}
