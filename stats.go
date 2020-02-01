package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type StatsWorker struct {
	Samples    map[int64]int64
	Precision  time.Duration
	WindowSize time.Duration
	GotReq     chan time.Time
	Close      chan struct{}
	wg         *sync.WaitGroup
	mu         *sync.RWMutex
	fileLoaded chan struct{}
	dbFile     string
	startedAt  time.Time
}

func NewStatsWorker(file string, bufferSize int, windowSize time.Duration) *StatsWorker {
	return &StatsWorker{
		Samples:    map[int64]int64{},
		Precision:  5 * time.Second,
		WindowSize: windowSize,
		GotReq:     make(chan time.Time, bufferSize),
		Close:      make(chan struct{}),
		wg:         &sync.WaitGroup{},
		mu:         &sync.RWMutex{},
		dbFile:     file,
		startedAt:  time.Now(),
	}
}

func (sw *StatsWorker) CountRequest() {
	sw.wg.Add(1)
	sw.GotReq <- time.Now()
}

func (sw *StatsWorker) LoadFile() {
	log.Printf("loading file %s \n", sw.dbFile)
	file, err := os.Open(sw.dbFile)
	if err != nil {
		log.Printf("error opening file %s\n", err)
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

fileScanning:
	for scanner.Scan() {
		str := strings.Split(scanner.Text(), " ")
		if len(str) > 1 {

			reqUnix, err := strconv.ParseInt(str[0], 10, 64)
			if err != nil {
				log.Printf("error loading stat file,starting with a clean db: %s\n", err)
				break fileScanning
			}

			reqTime := time.Unix(reqUnix, 0)

			reqCount, err := strconv.ParseInt(str[1], 10, 64)
			if err != nil {
				log.Printf("error loading stat file %s\n", err)
				break fileScanning
			}
			sw.startedAt = time.Unix(int64(math.Min(float64(sw.startedAt.Unix()), float64(reqUnix))), 0)
			sw.IncrSample(reqTime, reqCount)
		}
	}
}

func (sw *StatsWorker) precisionLower(theTime time.Time) int64 {
	return theTime.Unix() - (theTime.Unix() % int64(sw.Precision.Seconds()))
}

func (sw *StatsWorker) Start() {
	sw.LoadFile() //block until file loaded into memory
	log.Println("starting statistics worker")
	go func() {
	workerLoop:
		for {
			select {
			case reqTime := <-sw.GotReq:
				sw.IncrSample(reqTime, 1)
				sw.wg.Done()
			case <-sw.Close:
				break workerLoop
			}
		}
	}()

	go func() { //cleanup after every 2 samples addition to keep memory clean
		for {
			<-time.After(sw.Precision * 2)
			sw.Cleanup()
		}
	}()
}

func (sw *StatsWorker) IncrSample(timeStamp time.Time, incrBy int64) {
	position := sw.precisionLower(timeStamp)
	sw.mu.Lock()
	if _, ok := sw.Samples[position]; ok {
		sw.Samples[position] = sw.Samples[position] + incrBy
	} else {
		sw.Samples[position] = incrBy
	}
	sw.mu.Unlock()
}

func (sw *StatsWorker) Stop() {
	go func() {
		sw.wg.Wait()
		log.Println("stopping logger gracefully")
		sw.dumpFile()
		close(sw.Close)
	}()

	select {
	case <-time.After(10 * time.Second):
		log.Println("timeout! killing logging process")
		close(sw.Close)
	case <-sw.Close:
		break
	}
}

func (sw *StatsWorker) dumpFile() {

	f, err := os.Create(sw.dbFile)
	if err != nil {
		log.Printf("error creating db file : %s\n", err)
		return
	}

	sw.mu.Lock()

	//find the pisitions we need to write into file
	lines := make([]int64, 0)
	for pos, _ := range sw.Samples {
		if sw.isInWindow(pos) {
			lines = append(lines, pos)
		}
	}
	//sort timestamps since range gives them us in a random order
	sort.Sort(SortableInt64(lines))

	for _, pos := range lines {
		_, err = f.WriteString(fmt.Sprintf("%d %d\n", pos, sw.Samples[pos]))
		if err != nil {
			log.Printf("error writing a db line : %s\n", err)
			break
		}
	}
	sw.mu.Unlock()
}

func (sw *StatsWorker) isInWindow(pos int64) bool {
	return sw.isInWindowOf(pos, sw.WindowSize)
}

func (sw *StatsWorker) isInWindowOf(pos int64, duration time.Duration) bool {
	now := time.Now()
	firstTime := now.Add(-duration)
	firstPosition := sw.precisionLower(firstTime)
	return pos <= now.Unix() && pos >= firstPosition
}

func (sw *StatsWorker) Cleanup() {
	sw.mu.Lock()
	now := time.Now()
	for pos, _ := range sw.Samples {
		if pos < now.Add(-sw.WindowSize).Unix() {
			delete(sw.Samples, pos)
		}
	}
	sw.mu.Unlock()
}

func (sw *StatsWorker) Total() int64 {
	return sw.TotalOf(sw.WindowSize)
}

func (sw *StatsWorker) TotalOf(duration time.Duration) int64 {
	var total int64 = 0
	sw.mu.Lock()
	for pos, count := range sw.Samples {
		if sw.isInWindowOf(pos, duration) {
			total = total + count
		}
	}
	sw.mu.Unlock()
	return total
}

func (sw *StatsWorker) Avg() float64 {
	return sw.AvgOf(sw.WindowSize)
}

func (sw *StatsWorker) AvgOf(duration time.Duration) float64 {
	total := sw.TotalOf(duration)
	return float64(total) / math.Min(time.Now().Sub(sw.startedAt).Seconds(), duration.Seconds())
}
