package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

func jobWithWg(j job, in, out chan interface{}, wg *sync.WaitGroup) {
	defer func() {
		close(out)
		wg.Done()
	}()
	j(in, out)
}

func ExecutePipeline(jobs ...job) {
	in := make(chan interface{}, MaxInputDataLen)
	out := make(chan interface{}, MaxInputDataLen)

	wg := &sync.WaitGroup{}

	for _, job := range jobs {
		wg.Add(1)
		go jobWithWg(job, in, out, wg)
		in = out
		out = make(chan interface{}, MaxInputDataLen)
	}

	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	md5Mutex := &sync.Mutex{}

	for item := range in {
		wg.Add(1)
		go singleHashWithWg(item, out, wg, md5Mutex)
	}

	wg.Wait()
}

func singleHashWithWg(item interface{}, out chan interface{}, wg *sync.WaitGroup, mu *sync.Mutex) {
	hash1 := make(chan string)
	hash2 := make(chan string)

	defer func() {
		close(hash1)
		close(hash2)
		wg.Done()
	}()

	strItem := toString(item)

	go crc32Hash(strItem, hash1)

	mu.Lock()
	md5HashSum := md5Hash(strItem)
	mu.Unlock()

	go crc32Hash(md5HashSum, hash2)

	out <- (<-hash1) + "~" + (<-hash2)
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	for item := range in {
		wg.Add(1)
		go multiHashWithWg(item, out, wg)
	}

	wg.Wait()
}

func multiHashWithWg(item interface{}, out chan interface{}, wg *sync.WaitGroup) {
	hashes := make([]chan string, 6)
	for i := 0; i < 6; i++ {
		hashes[i] = make(chan string)
	}

	defer func() {
		for _, ch := range hashes {
			close(ch)
		}
		wg.Done()
	}()

	strBuilder := strings.Builder{}
	strItem := toString(item)

	for i, ch := range hashes {
		go crc32Hash(toString(i)+strItem, ch)
	}

	for _, ch := range hashes {
		strBuilder.WriteString(<-ch)
	}

	out <- strBuilder.String()
}

func CombineResults(in, out chan interface{}) {
	strItems := make([]string, 0, MaxInputDataLen)

	for item := range in {
		strItems = append(strItems, toString(item))
	}

	sort.Slice(strItems, func(i, j int) bool {
		return strItems[i] < strItems[j]
	})

	out <- strings.Join(strItems, "_")
}

func crc32Hash(data string, out chan<- string) {
	out <- DataSignerCrc32(data)
}

func md5Hash(data string) string {
	return DataSignerMd5(data)
}

func toString(item interface{}) string {
	strItem, ok := item.(string)
	if !ok {
		strItem = fmt.Sprintf("%v", item)
	}
	return strItem
}
