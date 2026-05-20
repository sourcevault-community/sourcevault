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

package crypto

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"

	"sourcevault/internal/config"
	"sourcevault/internal/db"
	"sourcevault/internal/registry"
)

// EnsureCA guarantees that the local system is in sync with the authoritative
// CA defined in the registry.
// 1. If an active CA exists in the registry, it ensures local files match.
// 2. It DOES NOT force-create a CA if missing (manual initialization required).
// 3. It DOES NOT automatically unseal the signer.
func EnsureCA(cfg *config.Config, dbConn *sql.DB, signer *CASigner) error {
	caDir := filepath.Join(cfg.RootDir, "data", "ca")
	if err := os.MkdirAll(caDir, 0700); err != nil {
		return fmt.Errorf("creating CA directory: %w", err)
	}

	// Step 1: Check registry for the current authoritative CA.
	activeMeta, err := registry.GetActiveCA(cfg)
	if err != nil {
		return fmt.Errorf("checking registry for active CA: %w", err)
	}

	if activeMeta != nil {
		slog.Debug("Found active CA in registry", "uuid", activeMeta.UUID)
		localPrivPath := filepath.Join(caDir, activeMeta.UUID)
		localPubPath := localPrivPath + ".pub"

		// Ensure DB cache is up to date with registry
		if err := db.UpsertCA(dbConn, *activeMeta, true); err != nil {
			slog.Warn("Failed to sync CA metadata to database cache", "uuid", activeMeta.UUID, "error", err)
		}

		// If local files are missing, restore them from registry so the node is "ready" to be unsealed.
		if _, err := os.Stat(localPrivPath); os.IsNotExist(err) {
			slog.Info("Local CA missing, restoring from registry", "uuid", activeMeta.UUID)
			if err := restoreCA(activeMeta, localPrivPath, localPubPath); err != nil {
				return fmt.Errorf("restoring CA from registry: %w", err)
			}
		}
	} else {
		slog.Info("No active CA found in registry. System is in uninitialized state.")
	}

	return nil
}

// ForceCreateCA generates a new CA keypair, saves it locally, uploads it to the
// registry, caches it in the DB, and marks it as the active CA.
func ForceCreateCA(cfg *config.Config, dbConn *sql.DB, signer *CASigner) error {
	caDir := filepath.Join(cfg.RootDir, "data", "ca")
	
	passphrase := []byte(cfg.CA.Passphrase)
	if len(passphrase) == 0 {
		return fmt.Errorf("cannot force-create CA: no passphrase provided (use SOURCEVAULT_CA_PASSPHRASE or provide one via interactive prompt)")
	}

	privPEM, pubAuth, err := GenerateCAKey(cfg.CA.DefaultKeyType, cfg.CA.DefaultRSABits, passphrase)
	if err != nil {
		return fmt.Errorf("generating CA key: %w", err)
	}

	caUUID := uuid.New().String()
	privPath := filepath.Join(caDir, caUUID)
	pubPath := privPath + ".pub"

	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		return fmt.Errorf("writing private key: %w", err)
	}
	if err := os.WriteFile(pubPath, pubAuth, 0644); err != nil {
		return fmt.Errorf("writing public key: %w", err)
	}

	// Parse fingerprint for metadata
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(pubAuth)
	if err != nil {
		return fmt.Errorf("parsing public key for fingerprint: %w", err)
	}
	fingerprint := ssh.FingerprintSHA256(pubKey)

	now := time.Now().UTC()
	meta := registry.CAMetadata{
		UUID:                caUUID,
		Name:                "Initial System CA",
		Algorithm:           cfg.CA.DefaultKeyType,
		Fingerprint:         fingerprint,
		EncryptedPrivateKey: string(privPEM),
		PublicKey:           string(pubAuth),
		CreatedAt:           now,
		ValidFrom:           now,
		ValidUntil:          now.Add(time.Duration(cfg.CA.DefaultValidDays) * 24 * time.Hour),
	}

	if err := registry.SaveCAMetadata(cfg, meta); err != nil {
		return fmt.Errorf("saving CA metadata to registry: %w", err)
	}

	if err := registry.SetActiveCA(cfg, caUUID); err != nil {
		return fmt.Errorf("setting active CA in registry: %w", err)
	}

	// Cache in DB
	if err := db.UpsertCA(dbConn, meta, true); err != nil {
		slog.Warn("Failed to cache new CA metadata in database", "uuid", caUUID, "error", err)
	}

	slog.Info("New CA created and registered", "uuid", caUUID, "fingerprint", fingerprint)

	// Auto-unseal the newly created CA
	return signer.UnsealFromPath(privPath, passphrase)
}

func restoreCA(meta *registry.CAMetadata, privPath, pubPath string) error {
	if err := os.WriteFile(privPath, []byte(meta.EncryptedPrivateKey), 0600); err != nil {
		return err
	}
	if err := os.WriteFile(pubPath, []byte(meta.PublicKey), 0644); err != nil {
		return err
	}
	return nil
}
