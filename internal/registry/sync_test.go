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

package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
	"sourcevault/internal/config"
)

func TestCAMetadataSync(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sourcevault-registry-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup directory structure
	worktree := filepath.Join(tmpDir, "data", "registry", "worktree")
	caDir := filepath.Join(worktree, "CertificateAuthority")
	if err := os.MkdirAll(caDir, 0755); err != nil {
		t.Fatalf("failed to create ca dir: %v", err)
	}

	// Initialize git repo in worktree
	if err := runGit(worktree, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := gitConfigSet(worktree, "user.email", "test@example.com"); err != nil {
		t.Fatalf("git config user.email failed: %v", err)
	}
	if err := gitConfigSet(worktree, "user.name", "Test User"); err != nil {
		t.Fatalf("git config user.name failed: %v", err)
	}

	cfg := &config.Config{
		RootDir: tmpDir,
	}

	meta := CAMetadata{
		UUID:        "test-uuid",
		Name:        "Test CA",
		Algorithm:   "ed25519",
		Fingerprint: "SHA256:abc",
		ValidFrom:   time.Now().UTC().Truncate(time.Second),
		ValidUntil:  time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second),
		CreatedAt:   time.Now().UTC().Truncate(time.Second),
		Revoked:     false,
	}

	// Test SaveCAMetadata
	if err := SaveCAMetadata(cfg, meta); err != nil {
		t.Fatalf("SaveCAMetadata failed: %v", err)
	}

	filePath := filepath.Join(caDir, meta.UUID+".yaml")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("metadata file %s was not created", filePath)
	}

	// Test RevokeCAMetadata
	if err := RevokeCAMetadata(cfg, meta.UUID); err != nil {
		t.Fatalf("RevokeCAMetadata failed: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read metadata file: %v", err)
	}

	var updatedMeta CAMetadata
	if err := yaml.Unmarshal(data, &updatedMeta); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}

	if !updatedMeta.Revoked {
		t.Error("expected metadata to be revoked")
	}
	if updatedMeta.RevokedAt.IsZero() {
		t.Error("expected RevokedAt to be set")
	}
}
