package prometheus

import (
	"Zminio/console"
	logger "Zminio/log"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	uploadObject   *prometheus.GaugeVec
	downloadObject *prometheus.GaugeVec
	WorkerIp       string
)

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

	prometheus.MustRegister(uploadObject, downloadObject)
	http.Handle("/metrics", promhttp.Handler())
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
	}
}

func IsValidIpv4(ip string) bool {
	ipaddr := net.ParseIP(ip)
	if ipaddr == nil {
		return false
	}
	return true
}
