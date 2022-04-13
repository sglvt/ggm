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

	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get device at index %d: %v", i, nvml.ErrorString(ret))
		}

		uuid, ret := device.GetUUID()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get uuid of device at index %d: %v", i, nvml.ErrorString(ret))
		}

		fmt.Printf("%v,%v\n", uuid, i)
		memory, ret := nvml.DeviceGetMemoryInfo(device)
		fmt.Printf("%v\n%v\n%v\n", memory.Total, memory.Free, memory.Used)

		t := time.Now().UnixMicro() - 5000000
		fmt.Printf("Timestamp is %v\n", uint64(t))

		processes, ret := nvml.DeviceGetProcessUtilization(device, uint64(t))
		for k := range processes {
			p := processes[k]
			if p.Pid > 0 {
				fmt.Printf("%v %v %v\n", p.Pid, p.TimeStamp, p.MemUtil)
			}
		}

		// processName, ret := nvml.SystemGetProcessName(1)
		// if ret != nvml.SUCCESS {
		// 	fmt.Printf("SystemGetProcessName: %v", ret)
		// } else {
		// 	fmt.Printf("SystemGetProcessName: %v", ret)
		// 	fmt.Printf("  name: %v", processName)
		// }
	}

}
