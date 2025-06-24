package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestServeCommand_Initialization(t *testing.T) {
	// Test that the serve command is properly initialized
	if serveCmd.Use != "serve" {
		t.Errorf("Expected serve command Use to be 'serve', got %s", serveCmd.Use)
	}

	expectedShort := "Start agent HTTP server with metrics and control endpoints"
	if serveCmd.Short != expectedShort {
		t.Errorf("Expected serve command Short to be '%s', got '%s'", expectedShort, serveCmd.Short)
	}

	// Test that Run function is set
	if serveCmd.Run == nil {
		t.Errorf("Expected serve command Run function to be set")
	}

	// Test that the command was added to root
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "serve" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected serve command to be added to root command")
	}
}

func TestServeCommand_Structure(t *testing.T) {
	// Test command structure and properties
	if serveCmd.Parent() != rootCmd {
		t.Errorf("Expected serve command parent to be root command")
	}

	// Test that the command has no subcommands (it's a leaf command)
	if len(serveCmd.Commands()) != 0 {
		t.Errorf("Expected serve command to have no subcommands, got %d", len(serveCmd.Commands()))
	}

	// Test that the command doesn't have any flags specific to it
	// (it should inherit from root's persistent flags)
	localFlags := serveCmd.LocalFlags()
	if localFlags.NFlag() != 0 {
		t.Errorf("Expected serve command to have no local flags, got %d", localFlags.NFlag())
	}
}

func TestServeCommand_RunFunction(t *testing.T) {
	// We can't easily test the actual Run function without starting a server
	// and dealing with logging initialization, but we can verify it exists
	// and has the right signature

	if serveCmd.Run == nil {
		t.Errorf("Expected Run function to be set")
		return
	}

	// Test that the function signature is correct by creating a mock call
	// We won't actually execute it to avoid starting a server in tests
	mockCmd := &cobra.Command{}
	mockArgs := []string{}

	// This should not panic (we're just testing the function signature)
	defer func() {
		if r := recover(); r != nil {
			// If it panics due to uninitialized config or logging, that's expected
			// We're just testing that the function signature is correct
			t.Logf("Function panicked as expected in test environment: %v", r)
		}
	}()

	// We don't actually call serveCmd.Run(mockCmd, mockArgs) here
	// because it would try to start a server and require proper initialization
	// The fact that we can reference it without compilation errors proves
	// the signature is correct
	_ = mockCmd
	_ = mockArgs
}

func TestServeCommand_Integration(t *testing.T) {
	// Test that serve command integrates properly with the root command structure

	// Find the serve command in root's subcommands
	var foundServe *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "serve" {
			foundServe = cmd
			break
		}
	}

	if foundServe == nil {
		t.Fatalf("serve command not found in root commands")
	}

	// Test command hierarchy
	if foundServe != serveCmd {
		t.Errorf("Found serve command is not the same as serveCmd variable")
	}

	// Test that it inherits persistent flags from root
	persistentFlags := foundServe.PersistentFlags()

	// Should not have its own persistent flags
	if persistentFlags.NFlag() != 0 {
		t.Errorf("Expected serve command to have no persistent flags of its own")
	}

	// Test some specific inherited flags
	expectedFlags := []string{"config", "debug", "log-level", "autodiscover"}
	for _, flag := range expectedFlags {
		if foundServe.Flags().Lookup(flag) == nil {
			t.Errorf("Expected serve command to inherit flag '%s'", flag)
		}
	}
}

func TestServeCommand_HelpText(t *testing.T) {
	// Test that help text is reasonable
	if len(serveCmd.Short) == 0 {
		t.Errorf("Expected serve command to have a short description")
	}

	// Long description is optional, but if present should be reasonable
	if serveCmd.Long != "" && len(serveCmd.Long) < 10 {
		t.Errorf("If Long description is set, it should be meaningful")
	}

	// Test that Use field is a single word (no spaces)
	if len(serveCmd.Use) == 0 {
		t.Errorf("Expected serve command Use to be non-empty")
	}

	hasSpace := false
	for _, char := range serveCmd.Use {
		if char == ' ' {
			hasSpace = true
			break
		}
	}
	if hasSpace {
		t.Errorf("Expected serve command Use to be a single word without spaces")
	}
}
