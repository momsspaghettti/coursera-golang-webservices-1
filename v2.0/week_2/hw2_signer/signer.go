package main

// сюда писать код

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

func processJob(job job, in, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	job(in, out)
	close(out)
}

func ExecutePipeline(jobs ...job) {
	in := make(chan interface{}, 1)
	out := make(chan interface{}, 1)
	wg := &sync.WaitGroup{}
	for _, job := range jobs {
		wg.Add(1)
		go processJob(job, in, out, wg)
		in = out
		out = make(chan interface{}, 1)
	}
	close(out)
	wg.Wait()
}

func computeMd5Safe(data string, mu *sync.Mutex) string {
	mu.Lock()
	defer mu.Unlock()
	return DataSignerMd5(data)
}

func computeMd5Hash(data string, mu *sync.Mutex) chan string {
	chRes := make(chan string, 1)
	go (func(ch chan string) {
		ch <- computeMd5Safe(data, mu)
	})(chRes)
	return chRes
}

func computeCrc32Hash(data string) chan string {
	chRes := make(chan string, 1)
	go (func(ch chan string) {
		ch <- DataSignerCrc32(data)
	})(chRes)
	return chRes
}

func computeSingleHash(data string, out chan interface{}, wg *sync.WaitGroup, mu *sync.Mutex) {
	md5Ch := computeMd5Hash(data, mu)
	firstCrc32Ch := computeCrc32Hash(data)
	secondSrc32Ch := computeCrc32Hash(<-md5Ch)

	out <- (<-firstCrc32Ch) + "~" + (<-secondSrc32Ch)
	wg.Done()
}

func SingleHash(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	for inputItem := range in {
		wg.Add(1)
		go computeSingleHash(strconv.Itoa(inputItem.(int)), out, wg, mu)
	}
	wg.Wait()
}

func computeMultiHash(data string, out chan interface{}, wg *sync.WaitGroup) {
	channels := make([]chan string, 6)
	for i := 0; i < 6; i++ {
		channels[i] = computeCrc32Hash(strconv.Itoa(i) + data)
	}

	res := strings.Builder{}
	for i := 0; i < 6; i++ {
		res.WriteString(<-channels[i])
	}

	out <- res.String()
	wg.Done()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for inputItem := range in {
		wg.Add(1)
		go computeMultiHash(inputItem.(string), out, wg)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	accumulator := make([]string, 0)
	var itemStr string
	for item := range in {
		itemStr = item.(string)
		accumulator = append(accumulator, itemStr)
	}

	sort.Slice(accumulator, func(i, j int) bool {
		return accumulator[i] < accumulator[j]
	})

	out <- strings.Join(accumulator, "_")
}
