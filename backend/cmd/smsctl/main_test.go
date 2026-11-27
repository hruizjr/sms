package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRefusesOutsideLocalEnvironments(t *testing.T) {
	t.Parallel()
	for _, env := range []string{"", "production", "staging"} {
		lookup := func(key string) (string, bool) {
			if key == "APP_ENV" && env != "" {
				return env, true
			}
			return "", false
		}
		var out, errOut bytes.Buffer
		code := run(context.Background(), []string{"tenant", "list"}, lookup, &out, &errOut)
		if code != exitUsage || !strings.Contains(errOut.String(), "refusing to run") || out.Len() != 0 {
			t.Errorf("APP_ENV=%q: exit %d, stderr %q", env, code, errOut.String())
		}
	}
}

func TestUsage(t *testing.T) {
	t.Parallel()
	var out, errOut bytes.Buffer
	if code := run(context.Background(), nil, func(string) (string, bool) { return "", false }, &out, &errOut); code != exitUsage || !strings.Contains(errOut.String(), "usage:") {
		t.Errorf("no arguments: exit %d, stderr %q", code, errOut.String())
	}
}
