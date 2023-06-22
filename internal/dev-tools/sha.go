package dev_tools

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CreateSHA512File computes the sha512 sum of the specified file the writes
// a sidecar file containing the hash and filename.
func CreateSHA512File(file string) error {
	fmt.Printf("Creating SHA512 hash... Filepath: %s\n", file+".sha512")
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("failed to open file for sha512 summing. Error %s", err)
	}
	defer f.Close()

	sum := sha512.New()
	if _, err := io.Copy(sum, f); err != nil {
		return fmt.Errorf("failed reading from input file. Error %s", err)
	}

	computedHash := hex.EncodeToString(sum.Sum(nil))
	out := fmt.Sprintf("%v  %v", computedHash, filepath.Base(file))

	return os.WriteFile(file+".sha512", []byte(out), 0644)
}
