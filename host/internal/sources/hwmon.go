package sources

import (
	"context"
	"log"
	"time"

	"github.com/datcal/hackintosh/host/internal/store"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// HWWorker samples CPU/RAM/Disk/Net every second and updates the store.
type HWWorker struct {
	S *store.Store

	prevNet   []net.IOCountersStat
	prevNetAt time.Time
}

func (h *HWWorker) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	h.sample(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.sample(ctx)
		}
	}
}

func (h *HWWorker) sample(ctx context.Context) {
	// CPU: 1s averaged percentage (single sample read on each tick)
	cpuPcts, err := cpu.PercentWithContext(ctx, 0, false)
	cpuPct := 0.0
	if err == nil && len(cpuPcts) > 0 {
		cpuPct = cpuPcts[0]
	}

	// RAM
	vm, err := mem.VirtualMemoryWithContext(ctx)
	ramPct := 0.0
	if err == nil {
		ramPct = vm.UsedPercent
	}

	// Disk
	diskPct := 0.0
	if du, err := disk.UsageWithContext(ctx, "/"); err == nil {
		diskPct = du.UsedPercent
	} else if du, err := disk.UsageWithContext(ctx, "C:"); err == nil {
		diskPct = du.UsedPercent
	}

	// Net throughput: diff counters since the last sample
	var upKBs, downKBs float64
	now := time.Now()
	cur, err := net.IOCountersWithContext(ctx, false)
	if err == nil && len(cur) > 0 {
		if len(h.prevNet) > 0 && !h.prevNetAt.IsZero() {
			dt := now.Sub(h.prevNetAt).Seconds()
			if dt > 0 {
				dSent := int64(cur[0].BytesSent) - int64(h.prevNet[0].BytesSent)
				dRecv := int64(cur[0].BytesRecv) - int64(h.prevNet[0].BytesRecv)
				if dSent < 0 { dSent = 0 }
				if dRecv < 0 { dRecv = 0 }
				upKBs = float64(dSent) / 1024.0 / dt
				downKBs = float64(dRecv) / 1024.0 / dt
			}
		}
		h.prevNet = cur
		h.prevNetAt = now
	} else if err != nil {
		log.Printf("hwmon: net counters: %v", err)
	}

	// Uptime
	upSec, _ := host.UptimeWithContext(ctx)
	uptime := time.Duration(upSec) * time.Second

	h.S.SetHW(store.HW{
		Valid:      true,
		CPUPct:     cpuPct,
		RAMPct:     ramPct,
		DiskPct:    diskPct,
		NetUpKBs:   upKBs,
		NetDownKBs: downKBs,
		Uptime:     uptime,
		UpdatedAt:  now,
	})
}
