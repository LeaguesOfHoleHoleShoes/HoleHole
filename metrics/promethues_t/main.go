package main

import(
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"time"
)

func main() {
	//初始一个http handler
	http.Handle("/metrics", promhttp.Handler())
	//初始化一个容器
	diskPercent := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "memory_percent",
		Help: "memory use percent",
	},
	[]string {"percent"},
	)
	xx := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "hahaha_a",
		Help: "ooo",
	})

	prometheus.MustRegister(diskPercent)
	prometheus.MustRegister(xx)

	go func() {
		err := http.ListenAndServe(":9110", nil)
		if err != nil {
			panic(err)
		}
	}()
	for i := 0; i < 13; i++ {
		x := i
		go func() {
			for {
				if x % 2 == 0 {
					xx.Inc()
				} else {
					diskPercent.WithLabelValues("usedMemory").Set(float64(x % 13))
				}
				time.Sleep(time.Second)
			}
		}()
	}
	select {}
}

