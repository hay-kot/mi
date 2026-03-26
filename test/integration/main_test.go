//go:build integration

package integration

import (
	"os"
	"os/exec"
	"testing"
)

var miBin string

func TestMain(m *testing.M) {
	if os.Getenv("MI_INTEGRATION") != "1" {
		os.Exit(0)
	}

	path, err := exec.LookPath("mi")
	if err != nil {
		panic("mi binary not found on PATH: " + err.Error())
	}
	miBin = path

	os.Exit(m.Run())
}
