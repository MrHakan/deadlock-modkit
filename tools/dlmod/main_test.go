package main

import (
	"os"
	"testing"
)

// TestMain makes the tests ignore a Deadlock that is open on this computer: they work on fake
// game folders, and a running game would make dlmod refuse to change mods.
func TestMain(m *testing.M) {
	gameRunning = func() bool { return false }
	allowLocalHTTP = true // the tests' stand-in servers run on this computer over plain HTTP
	os.Exit(m.Run())
}
