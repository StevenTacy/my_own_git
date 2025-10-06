package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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

	case "hash-object":
		if len(os.Args) < 4 {
			handleError(errors.New("usage: mygit hash-object -w <path-file>"))
			os.Exit(1)
		}

		if os.Args[2] != "-w" {
			handleError(errors.New("usage: mygit hash-object -w <path-file>"))
			os.Exit(1)
		}

		hashKey, err := writeFromPath(os.Args[3])
		if err != nil {
			handleError(err)
			os.Exit(1)
		}
		fmt.Printf("Hash: %s\n", hashKey)

	case "ls-tree":
		if len(os.Args) < 3 {
			handleError(errors.New("usage: mygit ls-tree [<args>...]"))
			os.Exit(1)
		}

		if os.Args[2] != "--name-only" {
			fmt.Fprintf(os.Stderr, "usage: mygit ls-tree --name-only [<args>...]\n")
			os.Exit(1)
		}

		printTree(os.Args[3])

	case "write-tree":
		_, hash, err := writeTree(".")
		if err != nil {
			handleError(errors.New("error writing tree"))
		}
		fmt.Println("Hashed tree object: ", hash)

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
	hashedParts := strings.SplitN(buffer.String(), "\x00", 2)
	if len(hashedParts) != 2 {
		return "", fmt.Errorf("invalid object format")
	}
	return hashedParts[1], nil
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

func writeFromPath(pathFile string) (string, error) {
	fileContent, err := os.ReadFile(pathFile)
	if err != nil {
		return "", fmt.Errorf("error reading file: %s", err)
	}

	_, hashKey, err := writeObject("blob", fileContent)
	if err != nil {
		return "", fmt.Errorf("error writing object: %s", err)
	}

	return hashKey, nil
}

func writeObject(contentType string, content []byte) ([20]byte, string, error) {
	header := fmt.Sprintf("%s %d\x00", contentType, len(content))
	storedContent := append([]byte(header), content...)

	hashKeyBytes := sha1.Sum(storedContent)
	hashKey := hex.EncodeToString(hashKeyBytes[:])
	if len(hashKey) != 40 {
		return [20]byte{}, "", fmt.Errorf("length of hashKey = %d is invalid", len(hashKey))
	}

	dir := fmt.Sprintf(".git/objects/%s", hashKey[:2])
	filePath := fmt.Sprintf("%s/%s", dir, hashKey[2:])
	if err := os.MkdirAll(dir, 0755); err != nil {
		return [20]byte{}, "", fmt.Errorf("error creating directory: %s %s", string(dir), err)
	}

	var buffer bytes.Buffer
	zipWriter := zlib.NewWriter(&buffer)
	_, err := zipWriter.Write(storedContent)
	if err != nil {
		return [20]byte{}, "", fmt.Errorf("error occured compressing the data:%s %s", storedContent, err)
	}
	defer zipWriter.Close()

	err = os.WriteFile(filePath, buffer.Bytes(), 0644)
	if err != nil {
		return [20]byte{}, "", fmt.Errorf("error writing content to file: %s, got error: %s", filePath, err)
	}

	return hashKeyBytes, hashKey, nil
}

type GitTree struct {
	Mode []byte
	Name []byte
	SHA  []byte
}

func printTree(hash string) {
	if len(hash) != 40 {
		fmt.Fprintf(os.Stderr, "invalid hash length\n")
		os.Exit(1)
	}

	entries := readTreeEntries(hash)
	dict := []string{}
	for _, item := range entries {
		dict = append(dict, string(item.Name))
	}

	sort.Strings(dict)
	for _, item := range dict {
		fmt.Println(item)
	}
}

/**
 * @param hash
 * seperate hashed parts into header and content
 * check if the object is tree type
 * return detailed tree entries
 */
func readTreeEntries(hash string) []*GitTree {
	buffer := readObject(hash)
	treeBlob := strings.SplitN(buffer.String(), "\x00", 2)

	if len(treeBlob) != 2 {
		fmt.Fprintf(os.Stderr, "invalid object format\n")
		os.Exit(1)
	}

	if !strings.HasPrefix(treeBlob[0], "tree") {
		fmt.Fprintf(os.Stderr, "invalid tree object\n")
		os.Exit(1)
	}

	entries := getDetailedTreeEntries([]byte(treeBlob[1]))
	return entries
}

/** used buffer to read the hashed parts and return the <mode> <name> <sha> of the tree object*/
func getDetailedTreeEntries(treeBlob []byte) []*GitTree {
	entries := []*GitTree{}

	reader := bufio.NewReader(bytes.NewBuffer(treeBlob))

	for {
		modeBytes, err := reader.ReadBytes(' ')
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}

		pathBytes, err := reader.ReadBytes(byte(0))
		if err != nil {
			panic(err)
		}

		shaBytes := make([]byte, 20)
		if _, err := reader.Read(shaBytes); err != nil {
			panic(err)
		}

		newEntry := &GitTree{
			Mode: modeBytes[:len(modeBytes)-1],
			Name: pathBytes[:len(pathBytes)-1],
			SHA:  shaBytes,
		}

		entries = append(entries, newEntry)
	}

	return entries
}

/**
 * read current path and recursively create tree object.
 * divide into 2 parts: directory and file
 */
func writeTree(path string) ([20]byte, string, error) {
	dir, err := os.ReadDir(path)
	if err != nil {
		return [20]byte{}, "", err
	}

	entries := []string{}
	for _, item := range dir {
		if item.Name() == ".git" {
			continue
		}

		if item.IsDir() {
			hash, _, err := writeTree(filepath.Join(path, item.Name()))
			if err != nil {
				return [20]byte{}, "", err
			}

			row := fmt.Sprintf("40000 %s\x00", item.Name(), hash)
			entries = append(entries, row)
			continue
		}

		contentFile, err := os.ReadFile(filepath.Join(path, item.Name()))
		if err != nil {
			panic(err)
		}

		hashKey, _, err := writeObject("blob", contentFile)
		if err != nil {
			return [20]byte{}, "", err
		}

		row := fmt.Sprintf("100644 %s\x00", item.Name(), hashKey)
		entries = append(entries, row)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i][strings.IndexByte(entries[i], ' ')+1:] < entries[j][strings.IndexByte(entries[i], ' ')+1:]
	})

	var buffer bytes.Buffer
	for _, entry := range entries {
		buffer.WriteString(entry)
	}

	return writeObject("tree", buffer.Bytes())
}
