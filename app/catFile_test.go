package catFile_test

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

func TestHashObject(t *testing.T) {
	fileName := "test.txt"
	fileContents := []byte("Hello, go")

	if err := os.WriteFile(fileName, fileContents, 0644); err != nil {
		t.Fatalf("Error writing to test file: %s\n", err)
	}

	wantHash, gitErr := RunGitHashObject(fileName)
	if gitErr != nil {
		t.Fatalf("Error implement git command: %s\n", gitErr)
	}
}

func RunGitHashObject(filePath string) (string, error) {
	cmd := exec.Command("git", "hash-object", "-w", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	// os.MkdirAll when create a directory -> if no err it will return nil
	return out.String(), nil
}
