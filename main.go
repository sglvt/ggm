package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const MIN_SCRAPE_INTERVAL = 5

var lastScrapeTimestamp int64 = 0

type gpuMetrics struct {
	fanSpeed          uint32
	memoryFree        uint64
	memoryUsed        uint64
	memoryTotal       uint64
	utilizationGPU    uint32
	utilizationMemory uint32
	temperature       uint32
}

type gpuCollector struct {
	fanSpeed          *prometheus.Desc
	memoryFree        *prometheus.Desc
	memoryUsed        *prometheus.Desc
	memoryTotal       *prometheus.Desc
	utilizationGPU    *prometheus.Desc
	utilizationMemory *prometheus.Desc
	temperature       *prometheus.Desc
}

var gpuMetricsList = []gpuMetrics{}

func newGpuCollector() *gpuCollector {
	return &gpuCollector{
		fanSpeed: prometheus.NewDesc("fan_speed",
			"FanSpeed",
			nil, nil,
		),
		memoryFree: prometheus.NewDesc("memory_free",
			"Memory Free",
			nil, nil,
		),
		memoryTotal: prometheus.NewDesc("memory_total",
			"Memory Total",
			nil, nil,
		),
		memoryUsed: prometheus.NewDesc("memory_used",
			"Memory Used",
			nil, nil,
		),
		utilizationGPU: prometheus.NewDesc("utilization_gpu",
			"GPU utilization is the percentage of time when SM(streaming multiprocessor) was busy",
			nil, nil,
		),
		utilizationMemory: prometheus.NewDesc("utilization_memory",
			"Memory utilization is actually the percentage of time the memory controller was busy (percentage of bandwidth used)",
			nil, nil,
		),
		temperature: prometheus.NewDesc("temperature",
			"Temperature",
			nil, nil,
		),
	}
}

func (g *gpuCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- g.utilizationGPU
}

func (g *gpuCollector) Collect(ch chan<- prometheus.Metric) {
	log.Debug("Collect()")
	readMetrics()

	fanSpeed := prometheus.MustNewConstMetric(g.fanSpeed, prometheus.GaugeValue, float64(gpuMetricsList[0].fanSpeed))
	memoryFree := prometheus.MustNewConstMetric(g.memoryFree, prometheus.GaugeValue, float64(gpuMetricsList[0].memoryFree))
	memoryTotal := prometheus.MustNewConstMetric(g.memoryTotal, prometheus.GaugeValue, float64(gpuMetricsList[0].memoryTotal))
	memoryUsed := prometheus.MustNewConstMetric(g.memoryUsed, prometheus.GaugeValue, float64(gpuMetricsList[0].memoryUsed))
	utilizationGPU := prometheus.MustNewConstMetric(g.utilizationGPU, prometheus.GaugeValue, float64(gpuMetricsList[0].utilizationGPU))
	utilizationMemory := prometheus.MustNewConstMetric(g.utilizationMemory, prometheus.GaugeValue, float64(gpuMetricsList[0].utilizationMemory))
	temperature := prometheus.MustNewConstMetric(g.temperature, prometheus.GaugeValue, float64(gpuMetricsList[0].temperature))
	ch <- fanSpeed
	ch <- memoryFree
	ch <- memoryTotal
	ch <- memoryUsed
	ch <- utilizationGPU
	ch <- utilizationMemory
	ch <- temperature
}

func getLogLevel() log.Level {
	level, err := log.ParseLevel(strings.ToLower(os.Getenv("LOG_LEVEL")))
	if err != nil {
		level = log.InfoLevel
	}
	return level
}

func initLogger() {
	log.SetLevel(getLogLevel())
}

func readMetrics() {
	if time.Now().Unix()-lastScrapeTimestamp > MIN_SCRAPE_INTERVAL {
		lastScrapeTimestamp = time.Now().Unix()
		gpuMetricsList = gpuMetricsList[:0]
		log.Debugf("Got metrics at %v\n", lastScrapeTimestamp)

		count, ret := nvml.DeviceGetCount()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get device count: %v", nvml.ErrorString(ret))
		}

		// Loop through devices
		for deviceIndex := 0; deviceIndex < count; deviceIndex++ {
			var deviceGpuMetrics gpuMetrics
			device, ret := nvml.DeviceGetHandleByIndex(deviceIndex)
			if ret != nvml.SUCCESS {
				log.Fatalf("Unable to get device at index %d: %v", deviceIndex, nvml.ErrorString(ret))
			}

			uuid, ret := device.GetUUID()
			if ret != nvml.SUCCESS {
				log.Fatalf("Unable to get uuid of device at index %d: %v", deviceIndex, nvml.ErrorString(ret))
			}

			log.Debugf("%v,%v\n", uuid, deviceIndex)

			t := time.Now().UnixMicro() - 5000000
			log.Debugf("Timestamp: %v\n", uint64(t))

			memory, ret := nvml.DeviceGetMemoryInfo(device)
			log.Debugf("[%v] MemoryInfo: total=%v free=%v used=%v\n", deviceIndex, memory.Total, memory.Free, memory.Used)

			deviceGpuMetrics.memoryFree = memory.Free
			deviceGpuMetrics.memoryTotal = memory.Total
			deviceGpuMetrics.memoryUsed = memory.Used

			processes, ret := nvml.DeviceGetProcessUtilization(device, uint64(t))
			for k := range processes {
				p := processes[k]
				if p.Pid > 0 {
					log.Debugf("[%v] Process: pid=%v ts=%v memutil=%v smutil=%v\n", deviceIndex, p.Pid, p.TimeStamp, p.MemUtil, p.SmUtil)
				}
			}

			temperature, ret := nvml.DeviceGetTemperature(device, nvml.TEMPERATURE_GPU)
			if ret == nvml.SUCCESS {
				log.Debugf("[%v] Temperature: %v\n", deviceIndex, temperature)
			} else if ret == nvml.ERROR_NOT_SUPPORTED {
				log.Debug("DeviceGetTemperature - Not supported")
			}
			deviceGpuMetrics.temperature = temperature

			utilization, ret := nvml.DeviceGetUtilizationRates(device)
			if ret == nvml.SUCCESS {
				log.Debugf("[%v] GPU Utilization: %v\n", deviceIndex, utilization.Gpu)
				log.Debugf("[%v] Memory Utilization: %v\n", deviceIndex, utilization.Memory)
			} else if ret == nvml.ERROR_NOT_SUPPORTED {
				log.Debug("DeviceGetUtilizationRates - Not supported")
			}
			deviceGpuMetrics.utilizationGPU = utilization.Gpu
			deviceGpuMetrics.utilizationMemory = utilization.Memory

			fanSpeed, ret := nvml.DeviceGetFanSpeed(device)
			if ret == nvml.SUCCESS {
				log.Debugf("[%v] Fan Speed: %v\n", deviceIndex, fanSpeed)
			} else if ret == nvml.ERROR_NOT_SUPPORTED {
				log.Debug("DeviceGetFanSpeed - Not supported")
			}
			deviceGpuMetrics.fanSpeed = fanSpeed

			// powerUsage, ret := nvml.DeviceGetPowerUsage(device)
			// if ret == nvml.SUCCESS {
			// 	log.Debugf("[%v] Power Usage: %v\n", deviceIndex, powerUsage)
			// } else if ret == nvml.ERROR_NOT_SUPPORTED {
			// 	log.Debug("DeviceGetPowerUsage - Not supported")
			// }

			// _, driverModel, ret := nvml.DeviceGetDriverModel(device)
			// if ret == nvml.SUCCESS {
			// 	log.Debugf("[%v] Power Usage: %v\n", deviceIndex, driverModel)
			// } else if ret == nvml.ERROR_NOT_SUPPORTED {
			// 	log.Debug("DeviceGetDriverModel - Not supported")
			// }

			gpuMetricsList = append(gpuMetricsList, deviceGpuMetrics)
		}

	}
}

func main() {
	initLogger()
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		log.Fatalf("Unable to initialize NVML: %v", nvml.ErrorString(ret))
	}
	defer func() {
		ret := nvml.Shutdown()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to shutdown NVML: %v", nvml.ErrorString(ret))
		}
	}()
	gpuCollector := newGpuCollector()
	prometheus.MustRegister(gpuCollector)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9177", nil))
}
