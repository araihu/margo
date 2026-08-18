package browserlaunch

import (
	"os/exec"
	"reflect"
	"testing"
)

func TestAppendMacAppCodeSignCloneFeaturePreservesChromedpDefaults(t *testing.T) {
	command := exec.Command("chromium", "--disable-features=site-per-process,Translate")

	appendMacAppCodeSignCloneFeature(command)

	want := []string{"chromium", "--disable-features=site-per-process,Translate,MacAppCodeSignClone"}
	if !reflect.DeepEqual(command.Args, want) {
		t.Fatalf("args = %q, want %q", command.Args, want)
	}
}

func TestAppendMacAppCodeSignCloneFeatureIsIdempotent(t *testing.T) {
	command := exec.Command("chromium", "--headless", "--disable-features=MacAppCodeSignClone")

	appendMacAppCodeSignCloneFeature(command)

	want := []string{"chromium", "--headless", "--disable-features=MacAppCodeSignClone"}
	if !reflect.DeepEqual(command.Args, want) {
		t.Fatalf("args = %q, want %q", command.Args, want)
	}
}

func TestAppendMacAppCodeSignCloneFeatureAddsMissingSwitch(t *testing.T) {
	command := exec.Command("chromium", "--headless")

	appendMacAppCodeSignCloneFeature(command)

	want := []string{"chromium", "--headless", "--disable-features=MacAppCodeSignClone"}
	if !reflect.DeepEqual(command.Args, want) {
		t.Fatalf("args = %q, want %q", command.Args, want)
	}
}
