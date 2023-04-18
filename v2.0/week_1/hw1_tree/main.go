package main

import (
	"fmt"
	"io"
	"os"
	pathLib "path"
	"sort"
	"strconv"
	"strings"
)

func filterDirEntries(dirEntries []os.DirEntry, includeFiles bool) []os.DirEntry {
	pointer := 0
	for i := range dirEntries {
		if !includeFiles && !dirEntries[i].IsDir() || strings.Contains(dirEntries[i].Name(), ".DS_Store") {
			continue
		}

		dirEntries[pointer] = dirEntries[i]
		pointer++
	}

	return dirEntries[:pointer]
}

func getFileSize(file os.DirEntry) (string, error) {
	fileInfo, err := file.Info()
	if err != nil {
		return "", err
	}

	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return "(empty)", nil
	}

	return "(" + strconv.FormatInt(fileSize, 10) + "b)", err
}

func dirTreeRecur(prefix string, out io.Writer, path string, printFiles bool) error {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	dirEntries = filterDirEntries(dirEntries, printFiles)
	sort.Slice(dirEntries, func(i, j int) bool {
		return dirEntries[i].Name() < dirEntries[j].Name()
	})

	var dirEntryBeginning, nextPrefix string
	for i, dirEntry := range dirEntries {
		if i+1 == len(dirEntries) {
			dirEntryBeginning = prefix + "└───"
			if dirEntry.IsDir() {
				nextPrefix = prefix + "\t"
			}
		} else {
			dirEntryBeginning = prefix + "├───"
			if dirEntry.IsDir() {
				nextPrefix = prefix + "│\t"
			}
		}

		if !dirEntry.IsDir() {
			fileSize, err := getFileSize(dirEntry)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, dirEntryBeginning+dirEntry.Name()+" "+fileSize)
			if err != nil {
				return err
			}
			continue
		}

		_, err = fmt.Fprintln(out, dirEntryBeginning+dirEntry.Name())
		if err != nil {
			return err
		}

		err = dirTreeRecur(nextPrefix, out, pathLib.Join(path, dirEntry.Name()), printFiles)
		if err != nil {
			return err
		}
	}

	return err
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	return dirTreeRecur("", out, path, printFiles)
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
