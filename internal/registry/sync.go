// SPDX-License-Identifier: AGPL-3.0-or-later
// SPDX-FileCopyrightText: 2026 The sourcevault Authors. All rights reserved.
// ===================================================================================================================================== //
// MP""""""`MM MMP"""""YMM M""MMMMM""M MM"""""""`MM MM'""""'YMM MM""""""""`M M""MMMMM""M MMP"""""""MM M""MMMMM""M M""MMMMMMMM M""""""""M //
// M  mmmmm..M M' .mmm. `M M  MMMMM  M MM  mmmm,  M M' .mmm. `M MM  mmmmmmmM M  MMMMM  M M' .mmmm  MM M  MMMMM  M M  MMMMMMMM Mmmm  mmmM //
// M.      `YM M  MMMMM  M M  MMMMM  M M'        .M M  MMMMMooM M`      MMMM M  MMMMP  M M         `M M  MMMMM  M M  MMMMMMMM MMMM  MMMM //
// MMMMMMM.  M M  MMMMM  M M  MMMMM  M MM  MMMb. "M M  MMMMMMMM MM  MMMMMMMM M  MMMM' .M M  MMMMM  MM M  MMMMM  M M  MMMMMMMM MMMM  MMMM //
// M. .MMM'  M M. `MMM' .M M  `MMM'  M MM  MMMMM  M M. `MMM' .M MM  MMMMMMMM M  MMP' .MM M  MMMMM  MM Mb       dM M         M MMMM  MMMM //
// MMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMM //
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
// Private key material is NEVER included — only public metadata is written.
type CAMetadata struct {
	UUID        string    `yaml:"uuid"`
	Name        string    `yaml:"name"`
	Algorithm   string    `yaml:"algorithm"`
	Fingerprint string    `yaml:"fingerprint"`
	ValidFrom   time.Time `yaml:"valid_from"`
	ValidUntil  time.Time `yaml:"valid_until"`
	CreatedAt   time.Time `yaml:"created_at"`
	Revoked     bool      `yaml:"revoked"`
	RevokedAt   time.Time `yaml:"revoked_at,omitempty"`
}

// SaveCAMetadata writes CA public metadata to the registry worktree under
// CertificateAuthority/{uuid}.yaml and commits the change.
// Private key material must never be passed to this function.
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
