package prometheus

import (
	"Zminio/console"
	logger "Zminio/log"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	uploadObject, downloadObject, syncCurrent, sideloader *prometheus.GaugeVec
	WorkerIp                                              string
)

var Uptime prometheus.Gauge

func ControllerPrometheusInit(prometheusHost string) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.ErrorLogger.Printf("error with get worker ip%v\n", err)
	}
	if len(addrs) > 1 {
		WorkerIp = strings.Split(addrs[1].String(), "/")[0]
	}

	uploadObject = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "zminio",
			Name:      "zminio_upload",
			Help:      "total uploaded objects",
		},
		[]string{
			"WorkerIp",
			"Upload_path",
			"Version",
		}, // labels
	)
	downloadObject = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "zminio",
			Name:      "zminio_download",
			Help:      "total downloaded objects",
		},
		[]string{
			"WorkerIp",
			"Download_path",
			"Version",
		}, // labels
	)
	syncCurrent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "zminio",
			Name:      "sync_status",
			Help:      "on progress sync objects",
		},
		[]string{
			"WorkerIp",
			"Sync_Interval",
			"Version",
		}, // labels
	)
	sideloader = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "zminio",
			Name:      "sideloader_status",
			Help:      "on progress of sideloader action",
		},
		[]string{
			"WorkerIp",
			"Sideloader",
			"Version",
		}, // labels
	)
	Uptime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "zsync_uptime",
		Help: "worker uptime in seconds",
	})

	prometheus.MustRegister(uploadObject, downloadObject, Uptime, syncCurrent, sideloader)
	http.Handle("/metrics", promhttp.Handler())
	go recordMetrics()
	if err := http.ListenAndServe(prometheusHost, nil); err != nil {
		logger.ErrorLogger.Fatalf("could not run prometheus server: %s", err.Error())
	}
}

func IncreasePrometheusCount(pType string) {
	if pType == "upload" {
		uploadObject.With(prometheus.Labels{
			"WorkerIp":    WorkerIp,
			"Upload_path": fmt.Sprintf("%v", console.Pathfile),
			"Version":     console.AppVersion,
		}).Add(1)
	} else if pType == "download" {
		downloadObject.With(prometheus.Labels{
			"WorkerIp":      WorkerIp,
			"Download_path": fmt.Sprintf("%v", console.OutPutFile),
			"Version":       console.AppVersion,
		}).Add(1)
	} else if pType == "sync" {
		downloadObject.With(prometheus.Labels{
			"WorkerIp":      WorkerIp,
			"Download_path": fmt.Sprintf("downloaded from %v", console.Url),
			"Version":       console.AppVersion,
		}).Add(1)
		syncCurrent.With(prometheus.Labels{
			"WorkerIp":      WorkerIp,
			"Sync_Interval": fmt.Sprintf("%v hour", console.Interval),
			"Version":       console.AppVersion,
		}).Add(1)
		uploadObject.With(prometheus.Labels{
			"WorkerIp":    WorkerIp,
			"Upload_path": fmt.Sprintf("uploaded into the %v", console.Url_sync),
			"Version":     console.AppVersion,
		}).Add(1)
	}
}

func SetSideloader(message string) {
	sideloader.With(prometheus.Labels{
		"WorkerIp":   WorkerIp,
		"Sideloader": message,
		"Version":    console.AppVersion,
	}).Add(1)
}

func ResetSideloader() {
	sideloader.Reset()
}

func DecreasePrometheusCount() {
	time.Sleep(10 * time.Second)
	syncCurrent.With(prometheus.Labels{
		"WorkerIp":      WorkerIp,
		"Sync_Interval": fmt.Sprintf("%v hour", console.Interval),
		"Version":       console.AppVersion,
	}).Dec()
}

func IsValidIpv4(ip string) bool {
	ipaddr := net.ParseIP(ip)
	if ipaddr == nil {
		return false
	}
	return true
}

func recordMetrics() {
	startTime := time.Now()
	for {
		Uptime.Set(time.Since(startTime).Seconds())
		time.Sleep(1 * time.Second)
	}
}
