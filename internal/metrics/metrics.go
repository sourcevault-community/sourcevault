// SPDX-License-Identifier: AGPL-3.0-or-later
// SPDX-FileCopyrightText: 2026 The sourcevault Authors. All rights reserved.
// ===================================================================================================================================== //
// MP""""""`MM MMP"""""YMM M""MMMMM""M MM"""""""`MM MM'""""'YMM MM""""""""`M M""MMMMM""M MMP"""""""MM M""MMMMM""M M""MMMMMMMM M""""""""M //
// M  mmmmm..M M' .mmm. `M M  MMMMM  M MM  mmmm,  M M' .mmm. `M MM  mmmmmmmM M  MMMMM  M M' .mmmm  MM M  MMMMM  M M  MMMMMMMM Mmmm  mmmM //
// M.      `YM M  MMMMM  M M  MMMMM  M M'        .M M  MMMMMooM M`      MMMM M  MMMMP  M M         `M M  MMMMM  M M  MMMMMMMM MMMM  MMMM //
// MMMMMMM.  M M  MMMMM  M M  MMMMM  M MM  MMMb. "M M  MMMMMMMM MM  MMMMMMMM M  MMMM' .M M  MMMMM  MM M  MMMMM  M M  MMMMMMMM MMMM  MMMM //
// M. .MMM'  M M. `MMM' .M M  `MMM'  M MM  MMMMM  M M. `MMM' .M MM  MMMMMMMM M  MMP' .MM M  MMMMM  MM M  `MMM'  M M  MMMMMMMM MMMM  MMMM //
// Mb.     .dM MMb     dMM Mb       dM MM  MMMMM  M MM.     .dM MM        .M M     .dMMM M  MMMMM  MM Mb       dM M         M MMMM  MMMM //
// MMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMM //
// ===================================================================================================================================== //
// This program is free software: you can redistribute it and/or modify                                                                  //
// it under the terms of the GNU Affero General Public License as                                                                        //
// published by the Free Software Foundation, either version 3 of the                                                                    //
// License, or (at your option) any later version.                                                                                       //
//                                                                                                                                       //
// This program is distributed in the hope that it will be useful,                                                                       //
// but WITHOUT ANY WARRANTY; without even the implied warranty of                                                                        //
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                                                                         //
// GNU Affero General Public License for more details.                                                                                   //
//                                                                                                                                       //
// You should have received a copy of the GNU Affero General Public License                                                              //
// along with this program.  If not, see <https://www.gnu.org/licenses/>.                                                                //
// ===================================================================================================================================== //

package metrics

import (
	"context"
	"log/slog"
	"time"
	"net/http"
	"fmt"
	
	"sourcevault/internal/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

var (
	cpuUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sourcevault_system_cpu_percent",
		Help: "Current CPU utilization of the system as a percentage",
	})

	memUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sourcevault_system_memory_used_bytes",
		Help: "Current memory used by the system in bytes",
	})

	diskFree = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sourcevault_system_disk_free_bytes",
		Help: "Current free disk space on the primary volume in bytes",
	})
)

// StartCollector launches a background goroutine that periodically polls system
// metrics and updates the Prometheus gauges. It respects context cancellation.
func StartCollector(ctx context.Context, rootDir string) {
	ticker := time.NewTicker(15 * time.Second)

	go func() {
		defer ticker.Stop()
		slog.Info("Started background system metrics collector")
		for {
			select {
			case <-ctx.Done():
				slog.Info("Stopping background system metrics collector")
				return
			case <-ticker.C:
				updateMetrics(rootDir)
			}
		}
	}()
}

// Run starts a dedicated HTTP server for Prometheus metrics.
func Run(ctx context.Context, cfg *config.Config) error {
	slog.Info("Starting dedicated metrics server", "host", cfg.Metrics.Host, "port", cfg.Metrics.Port)
	
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Metrics.Host, cfg.Metrics.Port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		slog.Info("Shutting down metrics server")
		srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("metrics server error: %w", err)
	}

	return nil
}

func updateMetrics(rootDir string) {
	// CPU usage (overall system usage)
	percents, err := cpu.Percent(0, false)
	if err == nil && len(percents) > 0 {
		cpuUsage.Set(percents[0])
	} else if err != nil {
		slog.Debug("Failed to collect CPU metrics", "error", err)
	}

	// Memory usage
	v, err := mem.VirtualMemory()
	if err == nil {
		memUsage.Set(float64(v.Used))
	} else {
		slog.Debug("Failed to collect memory metrics", "error", err)
	}

	// Disk free space based on the application's root directory
	d, err := disk.Usage(rootDir)
	if err == nil {
		diskFree.Set(float64(d.Free))
	} else {
		slog.Debug("Failed to collect disk metrics", "error", err)
	}
}
