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
	"time"

	"sourcevault/internal/registry"
)

// CA models the database representation of a Certificate Authority.
type CA struct {
	UUID                string
	Name                string
	Algorithm           string
	Fingerprint         string
	PublicKey           string
	EncryptedPrivateKey string
	IsActive            bool
	Revoked             bool
	RevokedAt           sql.NullTime
	CreatedAt           time.Time
	ValidFrom           time.Time
	ValidUntil          time.Time
}

// UpsertCA inserts or updates a CA's metadata in the database cache.
func UpsertCA(db *sql.DB, meta registry.CAMetadata, isActive bool) error {
	slog.Debug("Caching CA metadata in database", "uuid", meta.UUID, "active", isActive)

	query := `
		INSERT INTO certificate_authorities (
			uuid, name, algorithm, fingerprint, public_key, encrypted_private_key, is_active, revoked, revoked_at, created_at, valid_from, valid_until
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(uuid) DO UPDATE SET
			name = excluded.name,
			is_active = excluded.is_active,
			revoked = excluded.revoked,
			revoked_at = excluded.revoked_at
	`

	_, err := db.Exec(query,
		meta.UUID,
		meta.Name,
		meta.Algorithm,
		meta.Fingerprint,
		meta.PublicKey,
		meta.EncryptedPrivateKey,
		isActive,
		meta.Revoked,
		meta.RevokedAt,
		meta.CreatedAt,
		meta.ValidFrom,
		meta.ValidUntil,
	)

	if err != nil {
		return fmt.Errorf("upserting CA %s: %w", meta.UUID, err)
	}

	// If we just marked one as active, ensure all others are marked inactive
	if isActive {
		if _, err := db.Exec("UPDATE certificate_authorities SET is_active = 0 WHERE uuid != ?", meta.UUID); err != nil {
			return fmt.Errorf("resetting other active CAs: %w", err)
		}
	}

	return nil
}

// GetCAByUUID retrieves a single CA's metadata from the database cache.
func GetCAByUUID(db *sql.DB, uuid string) (*registry.CAMetadata, error) {
	query := `SELECT uuid, name, algorithm, fingerprint, public_key, encrypted_private_key, revoked, revoked_at, created_at, valid_from, valid_until 
	          FROM certificate_authorities WHERE uuid = ?`
	
	var meta registry.CAMetadata
	err := db.QueryRow(query, uuid).Scan(
		&meta.UUID,
		&meta.Name,
		&meta.Algorithm,
		&meta.Fingerprint,
		&meta.PublicKey,
		&meta.EncryptedPrivateKey,
		&meta.Revoked,
		&meta.RevokedAt,
		&meta.CreatedAt,
		&meta.ValidFrom,
		&meta.ValidUntil,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying CA %s: %w", uuid, err)
	}

	return &meta, nil
}

// GetActiveCA retrieves the metadata of the currently active CA from the database cache.
func GetActiveCA(db *sql.DB) (*registry.CAMetadata, error) {
	query := `SELECT uuid, name, algorithm, fingerprint, public_key, encrypted_private_key, revoked, revoked_at, created_at, valid_from, valid_until 
	          FROM certificate_authorities WHERE is_active = 1`
	
	var meta registry.CAMetadata
	err := db.QueryRow(query).Scan(
		&meta.UUID,
		&meta.Name,
		&meta.Algorithm,
		&meta.Fingerprint,
		&meta.PublicKey,
		&meta.EncryptedPrivateKey,
		&meta.Revoked,
		&meta.RevokedAt,
		&meta.CreatedAt,
		&meta.ValidFrom,
		&meta.ValidUntil,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying active CA: %w", err)
	}

	return &meta, nil
}
