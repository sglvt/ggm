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

var utilizationGPU uint32

var lastScrapeTimestamp int64 = 0

type gpuCollector struct {
	// UtilizationMemoryMetric *prometheus.Desc
	utilizationGPUMetric *prometheus.Desc
}

func newGpuCollector() *gpuCollector {
	return &gpuCollector{
		utilizationGPUMetric: prometheus.NewDesc("gpu_utilization",
			"GPU Utilization",
			nil, nil,
		),
	}
}

func (c *gpuCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.utilizationGPUMetric
}

func (c *gpuCollector) Collect(ch chan<- prometheus.Metric) {

	//Write latest value for each metric in the prometheus metric channel.
	//Note that you can pass CounterValue, GaugeValue, or UntypedValue types here.
	log.Debug("Collect()")
	readMetrics()

	m1 := prometheus.MustNewConstMetric(c.utilizationGPUMetric, prometheus.GaugeValue, float64(utilizationGPU))
	ch <- m1
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
		log.Debugf("Got metrics at %v\n", lastScrapeTimestamp)
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
		count, ret := nvml.DeviceGetCount()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get device count: %v", nvml.ErrorString(ret))
		}

		for deviceIndex := 0; deviceIndex < count; deviceIndex++ {
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
			utilization, ret := nvml.DeviceGetUtilizationRates(device)
			if ret == nvml.SUCCESS {
				// GPU utilization is the percentage of time when SM(streaming multiprocessor) was busy
				// Memory utilization is actually the percentage of time the memory controller was busy (percentage of bandwidth used)
				log.Debugf("[%v] GPU Utilization: %v\n", deviceIndex, utilization.Gpu)
				log.Debugf("[%v] Memory Utilization: %v\n", deviceIndex, utilization.Memory)
				utilizationGPU = utilization.Gpu
			} else if ret == nvml.ERROR_NOT_SUPPORTED {
				log.Debug("DeviceGetUtilizationRates - Not supported")
			}

			fanSpeed, ret := nvml.DeviceGetFanSpeed(device)
			if ret == nvml.SUCCESS {
				log.Debugf("[%v] Fan Speed: %v\n", deviceIndex, fanSpeed)
			} else if ret == nvml.ERROR_NOT_SUPPORTED {
				log.Debug("DeviceGetFanSpeed - Not supported")
			}

			_, vgpuUtilization, ret := nvml.DeviceGetVgpuUtilization(device, uint64(t))
			if ret == nvml.SUCCESS {
				log.Debugf("[%v] Vgpu Utilization: %v\n", deviceIndex, vgpuUtilization)
			} else if ret == nvml.ERROR_NOT_SUPPORTED {
				log.Debug("DeviceGetVgpuUtilization - Not supported")
			}
		}

	}
}

func main() {
	initLogger()
	gpuCollector := newGpuCollector()
	prometheus.MustRegister(gpuCollector)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9177", nil))
}
