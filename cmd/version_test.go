package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionCommand_Initialization(t *testing.T) {
	// Test that the version command is properly initialized
	if versionCmd.Use != "version" {
		t.Errorf("Expected version command Use to be 'version', got %s", versionCmd.Use)
	}

	expectedShort := "Print the version number"
	if versionCmd.Short != expectedShort {
		t.Errorf("Expected version command Short to be '%s', got '%s'", expectedShort, versionCmd.Short)
	}

	// Test that Run function is set
	if versionCmd.Run == nil {
		t.Errorf("Expected version command Run function to be set")
	}

	// Test that the command was added to root
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected version command to be added to root command")
	}
}

func TestVersionCommand_Structure(t *testing.T) {
	// Test command structure and properties
	if versionCmd.Parent() != rootCmd {
		t.Errorf("Expected version command parent to be root command")
	}

	// Test that the command has no subcommands (it's a leaf command)
	if len(versionCmd.Commands()) != 0 {
		t.Errorf("Expected version command to have no subcommands, got %d", len(versionCmd.Commands()))
	}

	// Test that the command doesn't have any flags
	localFlags := versionCmd.LocalFlags()
	if localFlags.NFlag() != 0 {
		t.Errorf("Expected version command to have no local flags, got %d", localFlags.NFlag())
	}

	persistentFlags := versionCmd.PersistentFlags()
	if persistentFlags.NFlag() != 0 {
		t.Errorf("Expected version command to have no persistent flags, got %d", persistentFlags.NFlag())
	}
}

func TestVersionCommand_RunFunction(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		expectedOutput string
	}{
		{
			name:           "with version set",
			version:        "1.2.3",
			expectedOutput: "ElasticPHP Agent version 1.2.3\n",
		},
		{
			name:           "with empty version",
			version:        "",
			expectedOutput: "ElasticPHP Agent version \n",
		},
		{
			name:           "with development version",
			version:        "dev",
			expectedOutput: "ElasticPHP Agent version dev\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original version
			originalVersion := Version
			defer func() {
				Version = originalVersion
			}()

			// Set test version
			Version = tt.version

			// Capture stdout
			originalStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create a buffer to capture output
			outputChan := make(chan string)
			go func() {
				var buf bytes.Buffer
				io.Copy(&buf, r)
				outputChan <- buf.String()
			}()

			// Execute the version command
			mockCmd := &cobra.Command{}
			mockArgs := []string{}
			versionCmd.Run(mockCmd, mockArgs)

			// Restore stdout and get output
			w.Close()
			os.Stdout = originalStdout
			output := <-outputChan

			// Verify output
			if output != tt.expectedOutput {
				t.Errorf("Expected output '%s', got '%s'", tt.expectedOutput, output)
			}
		})
	}
}

func TestVersionCommand_Integration(t *testing.T) {
	// Test that version command integrates properly with the root command structure

	// Find the version command in root's subcommands
	var foundVersion *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "version" {
			foundVersion = cmd
			break
		}
	}

	if foundVersion == nil {
		t.Fatalf("version command not found in root commands")
	}

	// Test command hierarchy
	if foundVersion != versionCmd {
		t.Errorf("Found version command is not the same as versionCmd variable")
	}

	// Test that it inherits persistent flags from root
	// Note: We can't test InheritedFlags() directly in tests easily

	// Test some specific inherited flags
	expectedFlags := []string{"config", "debug", "log-level", "autodiscover"}
	for _, flag := range expectedFlags {
		if foundVersion.Flags().Lookup(flag) == nil {
			t.Errorf("Expected version command to inherit flag '%s'", flag)
		}
	}
}

func TestVersionCommand_OutputFormat(t *testing.T) {
	// Test the exact output format
	originalVersion := Version
	defer func() {
		Version = originalVersion
	}()

	Version = "test-version-123"

	// Capture output using a different method
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute the command
	versionCmd.Run(&cobra.Command{}, []string{})

	// Close write end and restore stdout
	w.Close()
	os.Stdout = old

	// Read the output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Test format
	expected := fmt.Sprintf("ElasticPHP Agent version %s\n", Version)
	if output != expected {
		t.Errorf("Expected exact output format '%s', got '%s'", expected, output)
	}

	// Test that it starts with expected prefix
	expectedPrefix := "ElasticPHP Agent version "
	if !strings.HasPrefix(output, expectedPrefix) {
		t.Errorf("Expected output to start with '%s', got '%s'", expectedPrefix, output)
	}

	// Test that it ends with newline
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("Expected output to end with newline, got '%s'", output)
	}

	// Test that version appears in output
	if !strings.Contains(output, Version) {
		t.Errorf("Expected output to contain version '%s', got '%s'", Version, output)
	}
}

func TestVersionCommand_HelpText(t *testing.T) {
	// Test that help text is reasonable
	if len(versionCmd.Short) == 0 {
		t.Errorf("Expected version command to have a short description")
	}

	// Test that Use field is a single word (no spaces)
	if len(versionCmd.Use) == 0 {
		t.Errorf("Expected version command Use to be non-empty")
	}

	hasSpace := false
	for _, char := range versionCmd.Use {
		if char == ' ' {
			hasSpace = true
			break
		}
	}
	if hasSpace {
		t.Errorf("Expected version command Use to be a single word without spaces")
	}

	// Test that Short description mentions version
	shortLower := strings.ToLower(versionCmd.Short)
	if !strings.Contains(shortLower, "version") {
		t.Errorf("Expected Short description to mention 'version', got '%s'", versionCmd.Short)
	}
}

func TestVersionGlobalVariable(t *testing.T) {
	// Test that the Version variable can be set and read
	originalVersion := Version
	defer func() {
		Version = originalVersion
	}()

	testVersions := []string{"1.0.0", "v2.1.0", "dev", "", "1.0.0-beta1"}

	for _, testVersion := range testVersions {
		Version = testVersion
		if Version != testVersion {
			t.Errorf("Expected Version to be '%s', got '%s'", testVersion, Version)
		}
	}
}
