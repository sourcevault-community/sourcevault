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

// Package registry provides helpers for reading and writing YAML metadata
// to the Git-based system registry worktree.
package registry

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"sourcevault/internal/config"
)

// CAMetadata is the registry representation of a local or trusted CA.
// The private key is stored in its encrypted (sealed) form.
type CAMetadata struct {
	UUID                string    `yaml:"uuid"`
	Name                string    `yaml:"name"`
	Algorithm           string    `yaml:"algorithm"`
	Fingerprint         string    `yaml:"fingerprint"`
	EncryptedPrivateKey string    `yaml:"encrypted_private_key"`
	PublicKey           string    `yaml:"public_key"`
	ValidFrom           time.Time `yaml:"valid_from"`
	ValidUntil          time.Time `yaml:"valid_until"`
	CreatedAt           time.Time `yaml:"created_at"`
	Revoked             bool      `yaml:"revoked"`
	RevokedAt           time.Time `yaml:"revoked_at,omitempty"`
}

// ActiveCAMetadata tracks which CA is currently authoritative for the node.
type ActiveCAMetadata struct {
	UUID      string    `yaml:"uuid"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

// GetActiveCA retrieves the metadata of the currently active CA from the registry.
// Returns nil, nil if no CA is currently active.
func GetActiveCA(cfg *config.Config) (*CAMetadata, error) {
	worktree := filepath.Join(cfg.RootDir, "data", "registry", "worktree")
	activePath := filepath.Join(worktree, "CertificateAuthority", "ActiveCA.yaml")

	if _, err := os.Stat(activePath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(activePath)
	if err != nil {
		return nil, fmt.Errorf("reading ActiveCA metadata: %w", err)
	}

	var active ActiveCAMetadata
	if err := yaml.Unmarshal(data, &active); err != nil {
		return nil, fmt.Errorf("parsing ActiveCA metadata: %w", err)
	}

	// Now load the actual CA metadata
	caPath := filepath.Join(worktree, "CertificateAuthority", active.UUID+".yaml")
	caData, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("reading CA metadata for %s: %w", active.UUID, err)
	}

	var meta CAMetadata
	if err := yaml.Unmarshal(caData, &meta); err != nil {
		return nil, fmt.Errorf("parsing CA metadata for %s: %w", active.UUID, err)
	}

	return &meta, nil
}

// SetActiveCA marks a specific CA UUID as the active one in the registry.
func SetActiveCA(cfg *config.Config, uuid string) error {
	worktree := filepath.Join(cfg.RootDir, "data", "registry", "worktree")
	caDir := filepath.Join(worktree, "CertificateAuthority")
	activePath := filepath.Join(caDir, "ActiveCA.yaml")

	slog.Info("Setting active CA in registry", "uuid", uuid)

	active := ActiveCAMetadata{
		UUID:      uuid,
		UpdatedAt: time.Now().UTC(),
	}

	data, err := yaml.Marshal(&active)
	if err != nil {
		return fmt.Errorf("marshaling ActiveCA metadata: %w", err)
	}

	if err := os.WriteFile(activePath, data, 0o640); err != nil {
		return fmt.Errorf("writing ActiveCA metadata file: %w", err)
	}

	if err := gitAdd(worktree, activePath); err != nil {
		return fmt.Errorf("staging ActiveCA update: %w", err)
	}
	if err := gitCommit(worktree, "feat(ca): set active CA to "+uuid); err != nil {
		return fmt.Errorf("committing ActiveCA update: %w", err)
	}

	return nil
}

// SaveCAMetadata writes CA metadata to the registry worktree under
// CertificateAuthority/{uuid}.yaml and commits the change.
func SaveCAMetadata(cfg *config.Config, meta CAMetadata) error {
	worktree := filepath.Join(cfg.RootDir, "data", "registry", "worktree")
	caDir := filepath.Join(worktree, "CertificateAuthority")
	filePath := filepath.Join(caDir, meta.UUID+".yaml")

	slog.Info("Writing CA metadata to registry", "uuid", meta.UUID, "path", filePath)

	data, err := yaml.Marshal(&meta)
	if err != nil {
		return fmt.Errorf("marshaling CA metadata: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o640); err != nil {
		return fmt.Errorf("writing CA metadata file: %w", err)
	}

	// Commit the change to the bare repo so it is version-controlled.
	if err := gitAdd(worktree, filePath); err != nil {
		return fmt.Errorf("staging CA metadata: %w", err)
	}
	if err := gitCommit(worktree, "feat(ca): add CA "+meta.UUID+" to registry"); err != nil {
		return fmt.Errorf("committing CA metadata: %w", err)
	}

	slog.Info("CA metadata committed to registry", "uuid", meta.UUID)
	return nil
}

// RevokeCAMetadata marks a CA as revoked in the registry by updating its YAML
// file and committing the change.
func RevokeCAMetadata(cfg *config.Config, uuid string) error {
	worktree := filepath.Join(cfg.RootDir, "data", "registry", "worktree")
	filePath := filepath.Join(worktree, "CertificateAuthority", uuid+".yaml")

	slog.Info("Revoking CA in registry", "uuid", uuid)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading CA metadata for %s: %w", uuid, err)
	}

	var meta CAMetadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("parsing CA metadata for %s: %w", uuid, err)
	}

	meta.Revoked = true
	meta.RevokedAt = time.Now().UTC()

	updated, err := yaml.Marshal(&meta)
	if err != nil {
		return fmt.Errorf("marshaling updated CA metadata: %w", err)
	}

	if err := os.WriteFile(filePath, updated, 0o640); err != nil {
		return fmt.Errorf("writing updated CA metadata: %w", err)
	}

	if err := gitAdd(worktree, filePath); err != nil {
		return fmt.Errorf("staging CA revocation: %w", err)
	}
	if err := gitCommit(worktree, "feat(ca): revoke CA "+uuid); err != nil {
		return fmt.Errorf("committing CA revocation: %w", err)
	}

	slog.Info("CA revoked in registry", "uuid", uuid)
	return nil
}
