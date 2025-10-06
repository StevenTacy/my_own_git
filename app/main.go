package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

/** convert target file content into hash code */
func calculateGitObjectHash(content []byte) string {
	header := fmt.Sprintf("blob %d\x00", len(content))
	data := append([]byte(header), content...)
	hashedData := sha1.Sum(data)
	return fmt.Sprintf("%x", hashedData)
}

// func writeCompressedObject(filePath string, content []byte) error {

// }

// Usage: your_program.sh <command> <arg1> <arg2> ...
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Fprintf(os.Stderr, "Logs from your program will appear here!\n")

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		// 0755 in shell is rwxr-xr-x
		// 0644 in shell is rw-r--r--

		for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			}
		}

		headFileContents := []byte("ref: refs/heads/main\n")
		if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		}

		fmt.Println("Initialized git directory")

	case "cat-file":
		if len(os.Args) < 4 {
			handleError(errors.New("usage: mygit cat-file -p [<args>...]"))
			os.Exit(1)
		}

		if os.Args[2] != "-p" {
			handleError(errors.New("usage: mygit cat-file -p [<args>...]"))
			os.Exit(1)
		}

		fileContent, err := readContentObject(os.Args[3])
		if err != nil {
			handleError(err)
			os.Exit(1)
		}

		fmt.Printf("%s\n", fileContent)

	// case "hash-object":
	// 	if len(os.Args) != 4 {
	// 		fmt.Fprintf(os.Stderr, "usage: mygit hash-object -w <file-path>\n")
	// 		os.Exit(1)
	// 	}

	// 	writeFlag := os.Args[2]
	// 	if writeFlag != "-w" {
	// 		fmt.Fprintf(os.Stderr, "usage: mygit hash-object -w <file-path>\n")
	// 		os.Exit(1)
	// 	}

	// 	fileName = os.Args[3]
	// 	fileContents, err := os.ReadFile(fileName)
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
	// 		os.Exit(1)
	// 	}

	// 	objectHash := calculateGitObjectHash(fileContents)
	// 	dirName := objectHash[:2]
	// 	hashedFileName := objectHash[2:]
	// 	dirPath := fmt.Sprintf(".mygit/objects/%s", dirName)
	// 	dirErr := os.MkdirAll(dirPath, 0755)
	// 	if dirErr != nil {
	// 		fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", dirErr)
	// 		os.Exit(1)
	// 	}

	// 	filePath := fmt.Sprintf(".mygit/objects/%s/%s", dirName, hashedFileName)
	// 	writeErr := writeCompressedObject(filePath, fileContents)
	// 	if writeErr != nil {
	// 		fmt.Fprintf(os.Stderr, "Error writing compressed object: %s\n", writeErr)
	// 		os.Exit(1)
	// 	}

	// 	fmt.Printf("%s\n", objectHash)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}

func handleError(err error) {
	fmt.Fprintf(os.Stderr, err.Error()+"\n")
}

/**
 * read the content of hashed object
 * @param hash
 * @return content of the object
 */
func readContentObject(hash string) (string, error) {
	if len(hash) != 40 {
		return "", fmt.Errorf("invalid hash length")
	}

	buffer := readObject(hash)
	contentParts := strings.SplitN(buffer.String(), "\x00", 2)
	if len(contentParts) != 2 {
		return "", fmt.Errorf("invalid object format")
	}
	return contentParts[1], nil
}

/**
 * 1. first read the compressed object check if the file exists
 * 2. then decompress the content and return the buffer
 */
func readObject(hash string) bytes.Buffer {
	dir := fmt.Sprintf(".git/objects/%s", hash[:2])
	fileName := fmt.Sprintf("%s/%s", dir, hash[2:])

	fileContents, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
		os.Exit(1)
	}

	reader, err := zlib.NewReader(bytes.NewReader(fileContents))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decompressing the file: %s\n", err)
		os.Exit(1)
	}
	defer reader.Close()

	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, reader); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading decompressed data: %s\n", err)
		os.Exit(1)
	}

	return buffer
}
