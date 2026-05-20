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
	"strings"
)

// RunMigrations initializes the database schema.
// It is dialect-aware to support different CREATE TABLE syntax across database engines.
func RunMigrations(db *sql.DB, driver string) error {
	driver = strings.ToLower(driver)

	// Ensure the schema_migrations table exists for all dialects.
	if err := ensureMigrationsTable(db, driver); err != nil {
		return err
	}

	// Fetch the currently applied version
	var currentVersion int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to fetch current migration version: %w", err)
	}

	// Retrieve dialect-specific migrations
	migrations := getMigrations(driver)
	if len(migrations) == 0 {
		slog.Debug("No migrations found for driver", "driver", driver)
		return nil
	}

	// Apply pending migrations inside a transaction
	for i, migrationSQL := range migrations {
		version := i + 1
		if version <= currentVersion {
			continue // Already applied
		}

		slog.Info("Applying database migration", "version", version)

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("beginning migration transaction: %w", err)
		}

		if _, err := tx.Exec(migrationSQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("executing migration %d: %w", version, err)
		}

		// Record the applied migration
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			tx.Rollback()
			return fmt.Errorf("recording migration %d: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing migration %d: %w", version, err)
		}
	}

	slog.Info("Database migrations up to date")
	return nil
}

func ensureMigrationsTable(db *sql.DB, driver string) error {
	var query string

	switch driver {
	case "sqlite3", "sqlite":
		query = `CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`
	case "postgres", "postgresql":
		query = `CREATE TABLE IF NOT EXISTS schema_migrations (
			version SERIAL PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
	default:
		return fmt.Errorf("unsupported driver for schema migrations: %s", driver)
	}

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}
	return nil
}

// getMigrations returns a sequential list of SQL statements to apply
// for the given database driver.
func getMigrations(driver string) []string {
	var migrations []string

	switch driver {
	case "sqlite3", "sqlite":
		migrations = []string{
			// Version 1: Core System Tables
			`CREATE TABLE IF NOT EXISTS certificate_authorities (
				uuid TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				algorithm TEXT NOT NULL,
				fingerprint TEXT NOT NULL,
				public_key TEXT NOT NULL,
				encrypted_private_key TEXT NOT NULL,
				is_active BOOLEAN DEFAULT 0,
				revoked BOOLEAN DEFAULT 0,
				revoked_at DATETIME,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				valid_from DATETIME NOT NULL,
				valid_until DATETIME NOT NULL
			);`,
		}
	case "postgres", "postgresql":
		migrations = []string{}
	}

	return migrations
}
