package vm

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func SetupVMMetrics() {
	runningVMsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "cirrinad",
		Subsystem: "VMs",
		Name:      "running",
		Help:      "Number of running VMs",
	})

	totalVMsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "cirrinad",
		Subsystem: "VMs",
		Name:      "defined",
		Help:      "Total Number of VMs defined",
	})

	cpuVMGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "cirrinad",
		Subsystem: "VMs",
		Name:      "CPU",
		Help:      "Number of CPUs allocated to VMs",
	})

	memVMGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "cirrinad",
		Subsystem: "VMs",
		Name:      "Mem",
		Help:      "Megabytes of memory allocated to VMs",
	})
}
