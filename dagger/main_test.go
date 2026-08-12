package main

import "testing"

func TestValidateReleaseTag(t *testing.T) {
	t.Parallel()

	for _, tag := range []string{"v0.0.1", "v1.2.3-rc.1", "v1.2.3.build.4"} {
		if err := validateReleaseTag(tag); err != nil {
			t.Errorf("validateReleaseTag(%q): %v", tag, err)
		}
	}
	for _, tag := range []string{"", "1.2.3", "v1.2", "v1.2.3 bad", "v1.2.3/other"} {
		if err := validateReleaseTag(tag); err == nil {
			t.Errorf("validateReleaseTag(%q) unexpectedly succeeded", tag)
		}
	}
}

func TestValidateExecutionContext(t *testing.T) {
	t.Parallel()

	for _, execution := range []executionContext{
		{CacheDomain: "local", Nonce: "1-1"},
		{CacheDomain: "trusted-main", Nonce: "123-2"},
		{CacheDomain: "trusted-release", Nonce: "123-2", ReleaseTag: "v1.2.3"},
		{CacheDomain: "untrusted-pr-42", Nonce: "123-2"},
	} {
		if err := validateExecutionContext(execution); err != nil {
			t.Errorf("validateExecutionContext(%#v): %v", execution, err)
		}
	}
	for _, execution := range []executionContext{
		{CacheDomain: "trusted-main/../../other", Nonce: "1-1"},
		{CacheDomain: "untrusted-pr-main", Nonce: "1-1"},
		{CacheDomain: "trusted-main", Nonce: "same"},
		{CacheDomain: "trusted-main", Nonce: "1-1; touch bad"},
	} {
		if err := validateExecutionContext(execution); err == nil {
			t.Errorf("validateExecutionContext(%#v) unexpectedly succeeded", execution)
		}
	}
}
