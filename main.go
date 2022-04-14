package main

import (
	"fmt"
	"log"
	"time"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

func main() {
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

		fmt.Printf("%v,%v\n", uuid, deviceIndex)
		memory, ret := nvml.DeviceGetMemoryInfo(device)
		fmt.Printf("%v\n%v\n%v\n", memory.Total, memory.Free, memory.Used)

		t := time.Now().UnixMicro() - 5000000
		fmt.Printf("Timestamp is %v\n", uint64(t))

		var deviceTotalGPUUtilization uint32 = 0
		processes, ret := nvml.DeviceGetProcessUtilization(device, uint64(t))
		for k := range processes {
			p := processes[k]
			if p.Pid > 0 {
				fmt.Printf("%v %v %v %v\n", p.Pid, p.TimeStamp, p.MemUtil, p.SmUtil)
				deviceTotalGPUUtilization += p.SmUtil
			}
		}

		fmt.Printf("%v - Total GPU Utilization: %v\n", deviceIndex, deviceTotalGPUUtilization)

		// processName, ret := nvml.SystemGetProcessName(1)
		// if ret != nvml.SUCCESS {
		// 	fmt.Printf("SystemGetProcessName: %v", ret)
		// } else {
		// 	fmt.Printf("SystemGetProcessName: %v", ret)
		// 	fmt.Printf("  name: %v", processName)
		// }

		temperature, ret := nvml.DeviceGetTemperature(device, nvml.TEMPERATURE_GPU)
		fmt.Printf("%v\n", temperature)

		utilization, ret := nvml.DeviceGetUtilizationRates(device)
		if ret == nvml.SUCCESS {
			fmt.Printf("%v\n", utilization)
		} else if ret == nvml.ERROR_NOT_SUPPORTED {
			fmt.Println("DeviceGetTemperature - Not supported")
		}

		fanSpeed, ret := nvml.DeviceGetFanSpeed(device)
		if ret == nvml.SUCCESS {
			fmt.Printf("Fan Speed: %v\n", fanSpeed)
		} else if ret == nvml.ERROR_NOT_SUPPORTED {
			fmt.Println("DeviceGetFanSpeed - Not supported")
		}

		_, vgpuUtilization, ret := nvml.DeviceGetVgpuUtilization(device, uint64(t))
		if ret == nvml.SUCCESS {
			fmt.Printf("Vgpu Utilization: %v\n", vgpuUtilization)
		} else if ret == nvml.ERROR_NOT_SUPPORTED {
			fmt.Println("DeviceGetVgpuUtilization - Not supported")
		}
	}

}
