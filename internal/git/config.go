// Package git provides thin wrappers around the host's git binary.
package git

import "fmt"

// SetConfig writes a key-value pair to the repository's local git config.
func SetConfig(dir, key, value string) error {
	if _, err := Cmd(dir, "config", "--local", key, value); err != nil {
		return fmt.Errorf("setting git config %s: %w", key, err)
	}
	return nil
}

// GetConfig reads a value from the repository's local git config.
// Returns an error if the key is not set.
func GetConfig(dir, key string) (string, error) {
	val, err := Cmd(dir, "config", "--local", key)
	if err != nil {
		return "", fmt.Errorf("reading git config %s: %w", key, err)
	}
	return val, nil
}
