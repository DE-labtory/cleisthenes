package config

import (
	"testing"
)

func TestInit(t *testing.T) {
	if err := Init(""); err != nil {
		t.Fatalf("unexpected err: %s", err)
	}
	if !fileExists(configPath) {
		t.Fatalf("%s does not exist", configPath)
	}
}
