package lib

import (
	"fmt"
	"path/filepath"

	"github.com/landlock-lsm/go-landlock/landlock"
)

// SetupSandbox applies Landlock restrictions to the entire process.
// The process can only write to the parent directory of targetFile,
// but can read from everywhere. Network and IPC access is completely blocked.
// Returns an error if Landlock is unavailable.
func SetupSandbox(targetFile string) error {
	// Get absolute path of the target file
	absPath, err := filepath.Abs(targetFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of target file: %w", err)
	}

	// Get the parent directory of the target file
	parentDir := filepath.Dir(absPath)

	// Apply Landlock restrictions using V6:
	// - Read-only access to root filesystem (needed for reading libraries, configs, plugins)
	// - Read-write access to the parent directory of the target file
	// - No network access (bind or connect)
	// - No IPC access (signals and abstract Unix sockets to more privileged domains)
	err = landlock.V6.RestrictPaths(
		landlock.RODirs("/"),       // Read-only access to entire filesystem
		landlock.RWDirs(parentDir), // Read-write access to parent directory
	)

	if err != nil {
		return fmt.Errorf("failed to apply Landlock filesystem sandbox: %w", err)
	}

	// Restrict all network access (no bind or connect allowed)
	// By not specifying any allowed ports, all TCP operations are blocked
	err = landlock.V6.RestrictNet()
	if err != nil {
		return fmt.Errorf("failed to apply Landlock network sandbox: %w", err)
	}

	// Restrict IPC access (signals and abstract Unix sockets to more privileged domains)
	// This prevents the sandboxed process from communicating with processes outside
	// the sandbox domain, providing defense-in-depth against privilege escalation
	err = landlock.V6.RestrictScoped()
	if err != nil {
		return fmt.Errorf("failed to apply Landlock IPC sandbox: %w", err)
	}

	return nil
}
