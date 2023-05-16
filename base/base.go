package base

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"google.golang.org/protobuf/proto"
	// pb "jerroyd.com/ugit/base/basepb"
	"jerroyd.com/ugit/data"
)

func isIgnored(path string) bool {
	for _, part := range filepath.SplitList(path) {
		if part == ".ugit" || part == ".git" {
			return true
		}
	}
	return false
}

func emptyCwd() error {
	directory := "."
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		full := filepath.Join(directory, entry.Name())
		if isIgnored(full) {
			continue
		}
		err := os.RemoveAll(full)
		if err != nil {
			return err
		}
	}
	return nil
}

func uint64ToByteArray(num uint64) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, num)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), err
}

var sizeOfUint64 uint64 = uint64(unsafe.Sizeof(uint64(1)))

func byteArrayToUint64(b []byte) (num uint64, size uint64, err error) {
	buf := bytes.NewBuffer(b)
	num, err = binary.ReadUvarint(buf)
	buf2 := new(bytes.Buffer)
	binary.Write(buf2, binary.LittleEndian, uint64(5))
	// fmt.Printf("byteArrayToUint64 %d\n[% x]\n[% x]\n", num, b[:sizeOfUint64], buf2.Bytes())
	return num, sizeOfUint64, err

}
func ugitObjectMarshal(obj *UgitObject) ([]byte, error) {
	buf, err := proto.Marshal(obj)
	if err != nil {
		return nil, err
	}
	sizeBuf, err := uint64ToByteArray(uint64(len(buf)))
	if err != nil {
		return nil, err
	}
	sizeBuf = append(sizeBuf, buf...)
	return sizeBuf, nil
}
func ugitObjectUnmarshal(buf []byte, obj *UgitObject) (end uint64, err error) {
	len, start, err := byteArrayToUint64(buf)
	if err != nil {
		return 0, err
	}
	end = start + len
	err = proto.Unmarshal(buf[start:end], obj)
	// fmt.Printf(":%s %s %s -> %d\n", obj.Type_, obj.Oid, obj.Name, end)
	return end, err
}

func iterTreeEntries(oid string) ([]UgitObject, error) {
	objectList := make([]UgitObject, 0)
	if oid == "" {
		return nil, errors.New("iterTreeEntities requires an oid")
	}
	fh, err := data.GetObject(oid, "tree")
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	// read item count
	buf := make([]byte, sizeOfUint64)
	_, err = fh.Read(buf)
	if err != nil {
		return nil, err
	}
	count, _, err := byteArrayToUint64(buf)
	for i := uint64(0); i < count; i++ {
		// read length of current item
		_, err = fh.Read(buf)
		if err != nil {
			return nil, err
		}
		len, _, err := byteArrayToUint64(buf)
		if err != nil {
			return nil, err
		}
		buf2 := make([]byte, len)
		object := UgitObject{}
		fh.Read(buf2)
		if err != nil {
			return nil, err
		}
		err = proto.Unmarshal(buf2, &object)
		if err != nil {
			return nil, err
		}
		// fmt.Printf("%s %s %s\n", object.Type_, object.Oid, object.Name)
		objectList = append(objectList, UgitObject{ // Avoid "copy" complaint?
			Type_: object.Type_,
			Oid:   object.Oid,
			Name:  object.Name,
		})
	}
	return objectList, nil
}

type tupleOidPath struct {
	oid  string
	path string
}

func GetTree(oid string, basePath string) ([]tupleOidPath, error) {
	tree, err := iterTreeEntries(oid)
	fmt.Printf("iterTreeEntries ret %d\n", len(tree))
	list := make([]tupleOidPath, 0)
	if err != nil {
		return nil, err
	}

	if basePath == "" {
		basePath = "./"
	}
	for i := 0; i < len(tree); i++ {
		fmt.Printf("%s %s %s\n", tree[i].Type_, tree[i].Oid, tree[i].Name)
		full := filepath.Join(basePath, tree[i].Name)
		if strings.Index(tree[i].Name, "/") >= 0 {
			return nil, errors.New(fmt.Sprintf("GetTree error: unexpected '/' in %s [oid: %s]", tree[i].Name, tree[i].Oid))
		} else if tree[i].Name == "." || tree[i].Name == ".." {
			return nil, errors.New(fmt.Sprintf("GetTree error: unexpected object \"%s\" [oid: %s]", tree[i].Name, tree[i].Oid))
		} else if tree[i].Type_ == "blob" {
			tuple := tupleOidPath{
				path: full,
				oid:  tree[i].Oid,
			}
			list = append(list, tuple)
		} else if tree[i].Type_ == "tree" {
			fmt.Println("not okay")
			localList, err := GetTree(tree[i].Oid, full)
			if err != nil {
				return nil, err
			}
			list = append(list, localList...)
		} else {
			return nil, errors.New(fmt.Sprintf("GetTree error: unexpected type %s in %s [oid: %s]", tree[i].Type_, tree[i].Name, tree[i].Oid))
		}
	}
	fmt.Printf("GetTree %d\n", len(list))
	return list, nil
}
func ReadTree(tree_oid string) error {
	list, err := GetTree(tree_oid, "./")
	if err != nil {
		return err
	}
	fmt.Printf("ReadTree %d\n", len(list))

	emptyCwd()

	for _, tuple := range list {
		fmt.Printf("%s %s\n", tuple.oid, tuple.path)
		basedir, _ := filepath.Split(tuple.path)
		os.MkdirAll(basedir, os.FileMode(0755))
		fo, err := data.GetObject(tuple.oid, "")
		if err != nil {
			return err
		}
		defer fo.Close()
		fi, err := os.Create(tuple.path)
		if err != nil {
			return err
		}
		defer fi.Close()
		buf := make([]byte, 1024)
		for {
			bytesRead, err := fo.Read(buf)
			if err != nil && err != io.EOF {
				return err
			} else if bytesRead == 0 {
				break
			}
			fi.Write(buf[:bytesRead])
		}
	}
	return nil
}
func WriteTree(directory string) (oid string, err error) {
	if directory == "" {
		directory = "."
	}
	entries, err := os.ReadDir(directory)

	if err != nil {
		return "", err
	}

	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		var buf []byte
		list := &UgitObjectList{}
		list2 := []*UgitObject{}
		for _, entry := range entries {
			full := filepath.Join(directory, entry.Name())
			var type_ string
			var oid string
			// fmt.Println(full)
			if isIgnored(full) {
				continue
			} else if entry.IsDir() {
				// filepath.EvalSymlinks()
				oid, err = WriteTree(full)
				if err != nil {
					panic(err)
				}
				type_ = "tree"

			} else {
				// fmt.Println(full)
				type_ = "blob"
				fh, err := os.Open(full)
				if err != nil {
					panic(err)
				}
				defer fh.Close()
				oid, err = data.HashObject(fh, "blob")
			}

			var tuple2 = &UgitObject{
				Name:  entry.Name(),
				Oid:   oid,
				Type_: type_,
			}
			// fmt.Printf("%s %s %s\n", tuple2.Oid, tuple2.Type_, tuple2.Name)
			list.Objects = append(list.Objects, tuple2)
			list2 = append(list2, tuple2)
			encoded, err := proto.Marshal(tuple2)
			if err != nil {
				panic(err)
			}
			buf = append(buf, encoded...)

		}
		// write data
		// 1. write number of types
		buf2, err := uint64ToByteArray(uint64(len(list2)))
		if err != nil {
			panic(err)
		}
		writer.Write(buf2)
		// then write every item
		for i := 0; i < len(list2); i++ {
			// fmt.Printf("%s %s %s\n", list2[i].Oid, list2[i].Type_, list2[i].Name)
			buf2, err = ugitObjectMarshal(list2[i])
			if err != nil {
				panic(err)
			}
			writer.Write(buf2)
		}
	}()

	// creat the tree object
	oid, err = data.HashObject(reader, "tree")
	// fmt.Printf("%s %s %s\n", oid, "tree", directory)
	// fmt.Println(oid)
	return oid, err
}
