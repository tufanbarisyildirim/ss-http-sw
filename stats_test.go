package ss_http_sw

import (
	"log"
	"testing"
	"time"
)

//those dummy package level variables are for preventing compiler optimisation in our benchmarks.
var rTotal int64
var rAvg float64

func BenchmarkStatsWorker_Avg(b *testing.B) {
	log.SetFlags(0)
	worker := NewStatsWorker("benchmark.txt", 500, 5*time.Minute)
	worker.Start()
	for i := 0; i < b.N; i++ {
		worker.CountRequest()
		rAvg = worker.Avg()
	}
	worker.Stop()
}

func BenchmarkStatsWorker_Total(b *testing.B) {
	log.SetFlags(0)
	worker := NewStatsWorker("benchmark.txt", 500, 5*time.Minute)
	worker.Start()
	for i := 0; i < b.N; i++ {
		worker.CountRequest()
		rTotal = worker.Total()
	}
	worker.Stop()
}
