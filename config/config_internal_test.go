package config

import (
	"os"
	"testing"
)

func TestGetConfiguration(t *testing.T) {
	configPath = os.Getenv("GOPATH") + "/src/github.com/DE-labtory/cleisthenes/config/testdata/config.golden.yml"

	c := Get()
	if c.HoneyBadger.NetworkSize != 10 {
		t.Fatalf("expected N is %d, but got %d", 10, c.HoneyBadger.NetworkSize)
	}
	if c.HoneyBadger.Byzantine != 3 {
		t.Fatalf("expected F is %d, but got %d", 3, c.HoneyBadger.Byzantine)
	}
	if c.Members.Addresses[0] != "localhost:8080" {
		t.Fatalf("expected address[0] is %s, but got %s", "localhost:8080", c.Members.Addresses[0])
	}
	if c.Members.Addresses[1] != "localhost:8081" {
		t.Fatalf("expected address[1] is %s, but got %s", "localhost:8081", c.Members.Addresses[1])
	}
	if c.Tpke.MasterPublicKey != "asdf" {
		t.Fatalf("expected master public key is %s, but got %s", "asdf", c.Tpke.MasterPublicKey)
	}
	if c.Tpke.Threshold != 3 {
		t.Fatalf("expected threshold is %d, but got %d", 3, c.Tpke.Threshold)
	}
}
