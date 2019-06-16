package log_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DE-labtory/cleisthenes/log"
)

func TestDebug(t *testing.T) {
	log.SetToDebug()
	log.Debug("level", "debug")
}

func TestInfo(t *testing.T) {
	log.SetToInfo()
	log.Debug("level", "debug") // not printed
	log.Info("level", "info")
}

// printed console and file
func TestEnableFileLogger(t *testing.T) {
	os.RemoveAll("./test")
	absPath, _ := filepath.Abs("./test/cleisthenes.log")
	defer os.RemoveAll("./test")
	err := log.EnableFileLogger(true, absPath)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("level", "error", "filepath", absPath)
}

// printed only file
func TestEnableOnlyFileLogger(t *testing.T) {
	os.RemoveAll("./test")
	absPath, _ := filepath.Abs("./test/cleisthenes.log")
	defer os.RemoveAll("./test")
	err := log.EnableOnlyFileLogger(true, absPath)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("level", "error", "filepath", absPath)
}
