package site

import (
	"os"
	"strings"
	"testing"
)

func TestComponentDocShellVersionMatchesDependency(t *testing.T) {
	data, err := os.ReadFile("../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "github.com/araihu/goshtoso-app-shells" {
			continue
		}
		if fields[1] != goshtosoComponentDocShellVersion {
			t.Fatalf("shell layout identity %q differs from dependency %q", goshtosoComponentDocShellVersion, fields[1])
		}
		doc, err := os.ReadFile("../CONTRIBUTING.md")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(doc), "`"+goshtosoComponentDocShellVersion+"`") {
			t.Fatal("CONTRIBUTING.md does not name the selected shell version")
		}
		return
	}
	t.Fatal("App Shells dependency missing from go.mod")
}
