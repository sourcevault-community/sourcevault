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

package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"sourcevault/internal/config"

	// Register sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

// Initialize creates a new database connection pool based on the configuration.
func Initialize(cfg *config.Config) (*sql.DB, error) {
	driver := strings.ToLower(cfg.Database.Driver)
	dsn := cfg.Database.DSN

	switch driver {
	case "sqlite3", "sqlite":
		// For SQLite, if the DSN isn't absolute, assume it's relative to RootDir.
		if !filepath.IsAbs(dsn) && !strings.HasPrefix(dsn, "file:") {
			dsn = filepath.Join(cfg.RootDir, dsn)
		}
		
		// Append high-performance PRAGMAs to the connection string
		// WAL mode ensures high concurrency without database locking.
		// busy_timeout=5000 ensures queries wait 5s for locks before throwing 'database is locked'.
		// foreign_keys=1 enforces relationships.
		if !strings.Contains(dsn, "?") {
			dsn += "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=1&_synchronous=NORMAL"
		}

		slog.Debug("Connecting to SQLite database", "dsn", dsn)
		db, err := sql.Open("sqlite3", dsn)
		if err != nil {
			return nil, fmt.Errorf("opening sqlite3 database: %w", err)
		}

		// SQLite only allows one writer at a time. To prevent 'database is locked' errors,
		// we configure the connection pool to have a single active connection,
		// or allow a few but rely on the busy_timeout.
		// For maximum safety with SQLite, MaxOpenConns = 1 is often recommended,
		// but with WAL mode, multiple readers and 1 writer are supported.
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)

		return db, nil

	case "postgres", "postgresql":
		// Placeholder for future Postgres support
		return nil, fmt.Errorf("postgres driver is not yet implemented")

	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}
