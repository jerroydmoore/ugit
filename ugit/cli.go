package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"log"
	"os"

	"jerroyd.com/ugit/base"
	"jerroyd.com/ugit/data"
)

func initialize() error {
	return data.Initialize()
}
func hashObject(file string) error {
	if file == "" {
		return errors.New("must specify a -file")
	}
	fh, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fh.Close()
	sha, err := data.HashObject(fh, "blob")
	fmt.Printf("%s %s\n", sha, file)
	return err
}
func catFile(object string) error {
	if object == "" {
		return errors.New("must specify a -object")
	}
	fh, err := data.GetObject(object, "blob")
	if err != nil {
		return err
	}
	defer fh.Close()
	buf := make([]byte, 1024)
	for {
		n, err := fh.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		fmt.Print(string(buf))
	}
	return nil
}

func writeTree() error {
	oid, err := base.WriteTree(".")
	fmt.Println(oid)
	return err
}

func readTree(tree string) (err error) {
	if tree == "" {
		return errors.New("must specify a -tree")
	}
	err = base.ReadTree(tree)
	return err
}

const CMD_INIT string = "init"
const CMD_HASH_OBJECT string = "hash-object"
const CMD_CAT_FILE string = "cat-file"
const CMD_WRITE_TREE string = "write-tree"
const CMD_READ_TREE string = "read-tree"

func main() {
	// init has no options
	// initCmd := flag.NewFlagSet(CMD_INIT, flag.ExitOnError)
	hashObjectCmd := flag.NewFlagSet(CMD_HASH_OBJECT, flag.ExitOnError)
	hashObjectFile := hashObjectCmd.String("file", "", "The file to hash")

	catFileCmd := flag.NewFlagSet(CMD_CAT_FILE, flag.ExitOnError)
	catFileObject := catFileCmd.String("object", "", "The object to print")

	readTreeCmd := flag.NewFlagSet(CMD_READ_TREE, flag.ExitOnError)
	readTreeTree := readTreeCmd.String("tree", "", "The tree to read")

	if len(os.Args) < 2 {
		fmt.Println("expected a subcommand")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case CMD_INIT:
		// initCmd.Parse(os.Args[2:])
		err = initialize()
	case CMD_HASH_OBJECT:
		hashObjectCmd.Parse(os.Args[2:])
		err = hashObject(*hashObjectFile)
	case CMD_CAT_FILE:
		catFileCmd.Parse(os.Args[2:])
		err = catFile(*catFileObject)
	case CMD_WRITE_TREE:
		err = writeTree()
	case CMD_READ_TREE:
		readTreeCmd.Parse(os.Args[2:])
		err = readTree(*readTreeTree)
	default:
		err = errors.New(fmt.Sprintf("unknown subcommand %s", os.Args[1]))

	}
	if err != nil {
		log.Fatalf("[ERROR] %s. See --help", err)
		os.Exit(1)
	}
}
