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

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"sourcevault/internal/crypto"
	"sourcevault/internal/db"
	"sourcevault/internal/metrics"
	"sourcevault/internal/registry"
	"sourcevault/internal/version"
	sv_web "sourcevault/internal/web"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start service",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Print the ASCII banner to stdout.
		fmt.Fprint(cmd.OutOrStdout(), banner)
		fmt.Fprint(cmd.OutOrStdout(), "\n\n")

		cfg := appCfg

		// Step 3: Log application metadata.
		slog.Info("Application is starting up", "application_name", version.Current.AppName, "application_version", version.Current.AppVersion)

		slog.Info("Application information",
			"application_name", version.Current.AppName,
			"application_version", version.Current.AppVersion,
			"git_branch", version.Current.GitBranch,
			"git_commit", version.Current.GitCommit,
			"build_date", version.Current.BuildDate,
			"architecture", version.Current.Architecture,
		)

		// Step 4: Log the parsed configuration at DEBUG level.
		slog.Debug("Base configuration",
			"root_dir", cfg.RootDir,
			"log_file", cfg.LogFile,
			"log_level", cfg.LogLevel,
		)
		slog.Debug("Web server configuration",
			"enabled", cfg.Web.Enabled,
			"host", cfg.Web.Host,
			"port", cfg.Web.Port,
		)
		slog.Debug("SSH server configuration",
			"enabled", cfg.Ssh.Enabled,
			"host", cfg.Ssh.Host,
			"port", cfg.Ssh.Port,
		)

		// Set up signal handling for graceful shutdown.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Define and create required directory structure. data/ groups all internal
		// system state; volumes/ stays at root as it holds user repository data.
		dataDir := filepath.Join(cfg.RootDir, "data")
		dirs := []string{
			cfg.RootDir,
			dataDir,
			filepath.Join(dataDir, "cache"),
			filepath.Join(dataDir, "ca"),
			filepath.Join(dataDir, "registry"),
			filepath.Join(cfg.RootDir, "volumes"),
		}

		if cfg.Database.Driver == "sqlite3" || cfg.Database.Driver == "sqlite" {
			dirs = append(dirs, filepath.Join(dataDir, "database"))
		}

		for _, dir := range dirs {
			if err := os.MkdirAll(dir, 0750); err != nil {
				slog.Error("Failed to create required directory", "dir", dir, "error", err)
				return fmt.Errorf("creating directory %s: %w", dir, err)
			}
			slog.Debug("Ensured required directory exists", "dir", dir)
		}

		// Ensure the Git-based system registry is initialized and the worktree is up to date.
		// This runs before database initialization so a fresh node can seed the DB from registry state.
		if err := registry.EnsureRegistry(cfg); err != nil {
			slog.Error("Failed to ensure system registry", "error", err)
			return fmt.Errorf("ensuring registry: %w", err)
		}

		// Initialize Database Connection
		dbConn, err := db.Initialize(cfg)
		if err != nil {
			slog.Error("Failed to initialize database", "error", err)
			return fmt.Errorf("initializing database: %w", err)
		}
		defer dbConn.Close()

		// Run Database Migrations
		if err := db.RunMigrations(dbConn, cfg.Database.Driver); err != nil {
			slog.Error("Failed to run database migrations", "error", err)
			return fmt.Errorf("running migrations: %w", err)
		}

		// Ensure a valid Certificate Authority (CA) exists for SSH certificate signing.
		// If missing, it will attempt to restore from registry or force-create a new one.
		// This runs after registry sync and DB migration to ensure the CA can be cached.
		if err := crypto.EnsureCA(cfg, dbConn, appSigner); err != nil {
			slog.Error("Failed to ensure Certificate Authority", "error", err)
			return fmt.Errorf("ensuring CA: %w", err)
		}

		// Use an errgroup to manage background services.
		g, ctx := errgroup.WithContext(ctx)

		// Start the Prometheus metrics collector in the background.
		metrics.StartCollector(ctx, cfg.RootDir)

		// Keep track of how many services were successfully started.
		var started int

		// Launch the dedicated metrics server if enabled.
		if cfg.Metrics.Enabled {
			started++
			g.Go(func() error {
				return metrics.Run(ctx, cfg)
			})
		}

		// Launch the Web server if enabled.
		if cfg.Web.Enabled {
			started++
			g.Go(func() error {
				return sv_web.Run(ctx, cfg)
			})
		}
		// Launch the SSH server if enabled.
		if cfg.Ssh.Enabled {
			// started++
			// g.Go(func() error { ... })
		}

		// If no services were enabled to start, wait for an interrupt signal
		if started == 0 {
			slog.Warn("No services (Web/SSH) are enabled; waiting for interrupt signal")
			<-ctx.Done()
		} else {
			// Wait for all background services to complete or for a termination signal.
			if err := g.Wait(); err != nil {
				return fmt.Errorf("application error: %w", err)
			}
		}

		slog.Info("Application shut down gracefully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
