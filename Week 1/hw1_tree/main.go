package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
)

type FileSystemItem interface {
	IsDir() bool
	ToString() string
}

type Directory struct {
	Name string
}

func (dir *Directory) IsDir() bool {
	return true
}

func (dir *Directory) ToString() string {
	return dir.Name
}

type File struct {
	Name string
	Size int64
}

func (f *File) IsDir() bool {
	return false
}

func (f *File) ToString() string {
	var size string
	if f.Size == 0 {
		size = "empty"
	} else {
		size = strconv.FormatInt(f.Size, 10) + "b"
	}
	return f.Name + " (" + size + ")"
}

func GetFsItems(path string, includeFiles bool) ([]FileSystemItem, error) {
	res := make([]FileSystemItem, 0, 16)
	var err error
	if fsItems, err := ioutil.ReadDir(path); err == nil {
		for _, fsInfo := range fsItems {
			if !includeFiles && !fsInfo.IsDir() {
				continue
			}
			if !fsInfo.IsDir() {
				res = append(res, &File{fsInfo.Name(), fsInfo.Size()})
			} else {
				res = append(res, &Directory{fsInfo.Name()})
			}
		}
	}
	return res, err
}

func dirTreeRecur(path string, includeFiles bool, offset string, out io.Writer) error {
	fsItems, err := GetFsItems(path, includeFiles)
	if err != nil {
		return err
	}

	for i, fsItem := range fsItems {
		var off, nextOff string
		if i+1 == len(fsItems) {
			off = offset + `└───`
			nextOff = offset + `	`
		} else {
			off = offset + `├───`
			nextOff = offset + `│	`
		}

		_, err = fmt.Fprintln(out, off+fsItem.ToString())
		if err != nil {
			return err
		}
		if fsItem.IsDir() {
			err = dirTreeRecur(path+string(os.PathSeparator)+fsItem.ToString(), includeFiles, nextOff, out)
		}

		if err != nil {
			return err
		}
	}

	return err
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	currDir, err := os.Getwd()
	if err != nil {
		return err
	}

	dir := currDir + string(os.PathSeparator) + path

	err = dirTreeRecur(dir, printFiles, "", out)

	return err
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
