// Copyright (c) 2017, NVIDIA CORPORATION. All rights reserved.

package nvidia

import (
	"fmt"
	"log"
	"strings"

	"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"

	"golang.org/x/net/context"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

func check(err error) {
	if err != nil {
		log.Panicln("Fatal:", err)
	}
}

func getDevices() []*pluginapi.Device {
	n, err := nvml.GetDeviceCount()
	check(err)

	var devs []*pluginapi.Device
	for i := uint(0); i < n; i++ {
		d, err := nvml.NewDeviceLite(i)
		check(err)
		devs = append(devs, &pluginapi.Device{
			ID:     d.UUID,
			Health: pluginapi.Healthy,
		})
	}

	return devs
}

func getGpuTopology() gpuTopology {
	n, err := nvml.GetDeviceCount()
	check(err)
	
	// init gpuTopology
	topology := make([][]gpuTopologyType, n)
	for i := 0; i < int(n); i++ {
		topology[i] = make([]gpuTopologyType, n)
	}
	
	var devs []*nvml.Device

	for i := uint(0); i < n; i++ {
		d, err := nvml.NewDeviceLite(i)
		check(err)
		devs = append(devs, d)
	}

	for i := 0; i < int(n); i++ {
		for j := 0; j < int(n); j++ {
			if (i != j) {
				p2plink, err := nvml.GetP2PLink(devs[i], devs[j])
				check(err)
				if p2plink != nvml.P2PLinkUnknown {
					topology[i][j] = gpuTopologyType(p2plink)
				}
				nvlink, err := nvml.GetNVLink(devs[i], devs[j])
				check(err)
				if nvlink != nvml.P2PLinkUnknown {
					topology[i][j] = gpuTopologyType(nvlink)
				}
				log.Printf("Warning: gpu%v == gpu%v topogoloy is: %v, description is %v", i, j, gpuTopologyType(nvlink).Abbreviation(), gpuTopologyType(nvlink).String())
			}
		}
	}

	return topology
}

func getDevNameMap() map[string]uint {
	n, err := nvml.GetDeviceCount()
	check(err)
	realDevNameMap := map[string]uint{}

	for i := uint(0); i < n; i++ {
		d, err := nvml.NewDevice(i)
		check(err)
		var id uint
		_, err = fmt.Sscanf(d.Path, "/dev/nvidia%d", &id)
		check(err)
		realDevNameMap[d.UUID] = id
	}
	return realDevNameMap
}

func deviceExists(devs []*pluginapi.Device, id string) bool {
	for _, d := range devs {
		if d.ID == id {
			return true
		}
	}
	return false
}

func watchXIDs(ctx context.Context, devs []*pluginapi.Device, xids chan<- *pluginapi.Device) {
	eventSet := nvml.NewEventSet()
	defer nvml.DeleteEventSet(eventSet)

	for _, d := range devs {
		err := nvml.RegisterEventForDevice(eventSet, nvml.XidCriticalError, d.ID)
		if err != nil && strings.HasSuffix(err.Error(), "Not Supported") {
			log.Printf("Warning: %s is too old to support healthchecking: %s. Marking it unhealthy.", d.ID, err)

			xids <- d
			continue
		}

		if err != nil {
			log.Panicln("Fatal:", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		e, err := nvml.WaitForEvent(eventSet, 5000)
		if err != nil && e.Etype != nvml.XidCriticalError {
			continue
		}

		// FIXME: formalize the full list and document it.
		// http://docs.nvidia.com/deploy/xid-errors/index.html#topic_4
		// Application errors: the GPU should still be healthy
		if e.Edata == 31 || e.Edata == 43 || e.Edata == 45 {
			continue
		}

		if e.UUID == nil || len(*e.UUID) == 0 {
			// All devices are unhealthy
			for _, d := range devs {
				xids <- d
			}
			continue
		}

		for _, d := range devs {
			if d.ID == *e.UUID {
				xids <- d
			}
		}
	}
}

//patchGpuTopology(gpuTopology)
