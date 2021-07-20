package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

type void struct{}

type User struct {
	Name     []byte
	Email    []byte
	Browsers [][]byte
}

func indexOfSubStr(s *[]byte, subStr *[]byte) int {
	for i := 0; i < len(*s)-len(*subStr)+1; i++ {
		if (*s)[i] == (*subStr)[0] {
			match := true
			for j := 0; j < len(*subStr); j++ {
				if (*s)[i+j] != (*subStr)[j] {
					match = false
					break
				}
			}
			if match {
				return i
			}
		}
	}

	return -1
}

func loadStr(s *[]byte, index *int, t *[]byte, replaceEmail bool) error {
	for {
		if *index == len(*s) {
			return errors.New("index out of range")
		}

		if (*s)[*index] == '"' {
			*index++
			break
		}

		if replaceEmail && (*s)[*index] == '@' {
			*t = append(*t, ' ')
			*t = append(*t, '[')
			*t = append(*t, 'a')
			*t = append(*t, 't')
			*t = append(*t, ']')
			*t = append(*t, ' ')
		} else {
			*t = append(*t, (*s)[*index])
		}
		*index++
	}

	return nil
}

func loadPropertyByName(s *[]byte, propertyName *[]byte, prop *[]byte, replaceEmail bool) error {
	index := indexOfSubStr(s, propertyName)
	if index < 0 {
		return errors.New("loadPropertyByName")
	}

	index += len(*propertyName)
	for {
		if index == len(*s) {
			return errors.New("index out of range")
		}

		if (*s)[index] == '"' {
			index++
			break
		}

		index++
	}

	if index == len(*s) {
		return errors.New("index out of range")
	}

	return loadStr(s, &index, prop, replaceEmail)
}

func loadBrowsers(s *[]byte, browsers *[][]byte, p *sync.Pool) error {
	index := indexOfSubStr(s, &[]byte{'"', 'b', 'r', 'o', 'w', 's', 'e', 'r', 's', '"'})

	if index < 0 {
		return errors.New("browsers")
	}

	index += 10
	for {
		if index == len(*s) {
			return errors.New("browsers index out of range")
		}

		if (*s)[index] == '[' {
			index++
			break
		}

		index++
	}

	if index == len(*s) {
		return errors.New("index out of range")
	}

	for {
		if index == len(*s) {
			return errors.New("browsers index out of range")
		}

		if (*s)[index] == ']' {
			break
		}

		if (*s)[index] == '"' {
			index++
			browser := p.Get().([]byte)
			err := loadStr(s, &index, &browser, false)
			if err != nil {
				return err
			}
			*browsers = append(*browsers, browser)
			if (*s)[index] == ']' {
				break
			}
		}

		index++
	}

	return nil
}

func (user *User) loadFrom(s []byte, p *sync.Pool) error {
	err := loadPropertyByName(&s, &[]byte{'"', 'n', 'a', 'm', 'e', '"'}, &user.Name, false)

	if err != nil {
		return err
	}

	err = loadPropertyByName(&s, &[]byte{'"', 'e', 'm', 'a', 'i', 'l', '"'}, &user.Email, true)
	if err != nil {
		return err
	}

	return loadBrowsers(&s, &user.Browsers, p)
}

func (user *User) clear(p *sync.Pool) {
	user.Name = user.Name[:0]
	user.Email = user.Email[:0]
	for _, browser := range user.Browsers {
		p.Put(browser[:0])
	}
	user.Browsers = user.Browsers[:0]
}

func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	defer func() {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}()

	seenBrowsers := make(map[string]void)
	uniqueBrowsers := 0

	scanner := bufio.NewScanner(file)

	user := User{make([]byte, 0, 512), make([]byte, 0, 512), make([][]byte, 0, 512)}

	bytesPool := sync.Pool{New: func() interface{} {
		return make([]byte, 0, 512)
	}}

	fmt.Fprintln(out, "found users:")

	android := []byte{'A', 'n', 'd', 'r', 'o', 'i', 'd'}
	msie := []byte{'M', 'S', 'I', 'E'}

	i := -1
	for scanner.Scan() {
		i++
		user.clear(&bytesPool)

		err = user.loadFrom(scanner.Bytes(), &bytesPool)
		if err != nil {
			panic(err)
		}

		if len(user.Browsers) == 0 {
			continue
		}

		isAndroid := false
		isMsie := false

		for _, browser := range user.Browsers {
			isAndroidP := indexOfSubStr(&browser, &android) != -1
			isMsieP := indexOfSubStr(&browser, &msie) != -1

			isAndroid = isAndroid || isAndroidP
			isMsie = isMsie || isMsieP

			if isAndroidP || isMsieP {
				browserStr := string(browser)
				if _, exists := seenBrowsers[browserStr]; !exists {
					uniqueBrowsers++
					seenBrowsers[browserStr] = void{}
				}
			}
		}

		if !(isAndroid && isMsie) {
			continue
		}

		fmt.Fprintln(out, fmt.Sprintf("[%d] %s <%s>", i, user.Name, user.Email))
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Total unique browsers", uniqueBrowsers)
}
