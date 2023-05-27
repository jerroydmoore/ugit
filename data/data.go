package data

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const GIT_DIR string = ".ugit"
const OID_LEN int = 40

func assertInitialized() {
	_, err := os.Stat(GIT_DIR)
	if err != nil {
		panic("ugit not initialized")
	}
}
func checkFileExists(filePath string) bool {
	_, error := os.Stat(filePath)
	return !errors.Is(error, os.ErrNotExist)
}

func Initialize() error {
	err := os.Mkdir(GIT_DIR, os.FileMode(0755))
	if err != nil {
		return err
	}
	err = os.Mkdir(filepath.Join(GIT_DIR, "objects"), os.FileMode(0755))
	return err
}

func GetHead() (oid string, err error) {
	file := filepath.Join(GIT_DIR, "HEAD")
	if !checkFileExists(file) {
		return "", nil
	}
	fh, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer fh.Close()
	// read the type off the beginning, and return the at beginning of data.
	buf := make([]byte, OID_LEN)
	bytesRead, err := fh.Read(buf)
	if bytesRead != OID_LEN {
		return "", errors.New("GetHead failed: unexpected read length")
	} else if err != nil && err != io.EOF {
		return "", err
	}
	return string(buf[:]), nil
}
func SetHead(oid string) error {
	file := filepath.Join(GIT_DIR, "HEAD")
	return os.WriteFile(file, []byte(oid), 0660)
}

func HashObject(fi io.Reader, type_ string) (oid string, err error) {
	assertInitialized()
	hasher := sha1.New()
	buf := make([]byte, 1024)

	fo, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	defer fo.Close()
	// write type
	if type_ == "" {
		type_ = "blob"
	}
	fo.Write([]byte(type_ + string('\000')))
	// write data
	for {
		bytesRead, err := fi.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		} else if bytesRead == 0 {
			break
		}
		hasher.Write(buf[:bytesRead])
		if _, err := fo.Write(buf[:bytesRead]); err != nil {
			return "", err
		}
	}
	oid = hex.EncodeToString(hasher.Sum(nil))

	// Move tempfile to {GIT_DIR}/objects/{oid}
	path := filepath.Join(GIT_DIR, "objects", oid)
	if !checkFileExists(path) {
		err = os.Rename(fo.Name(), path)
		if err != nil {
			fmt.Printf("Err renaming type %s", type_)
		}

	}
	return oid, err
}

func GetObject(oid string, expected_type string) (fh *os.File, err error) {
	assertInitialized()
	file := filepath.Join(GIT_DIR, "objects", oid)
	fh, err = os.Open(file)
	if err != nil {
		return nil, err
	}
	// read the type off the beginning, and return the at beginning of data.
	buf := make([]byte, 1024)
	bytesRead, err := fh.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	} else if bytesRead == 0 {
		return nil, errors.New("GetObject failed: unexpected empty file")
	}
	// find the null
	nullIdx := 0
	for nullIdx < bytesRead && buf[nullIdx] != '\000' {
		nullIdx++
	}

	// enforce type checking
	if expected_type != "" && expected_type != string(buf[0:nullIdx]) {
		return nil, errors.New(fmt.Sprintf("GetObject failed. type %s != %s", expected_type, string(buf[0:nullIdx])))
	}
	fh.Seek(int64(nullIdx+1), 0) // skip the null character
	return fh, nil
}
