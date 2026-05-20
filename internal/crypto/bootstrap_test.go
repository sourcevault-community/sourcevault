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

package crypto

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"sourcevault/internal/config"
	"sourcevault/internal/db"
)

func TestEnsureCA(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sourcevault-bootstrap-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup registry worktree
	worktree := filepath.Join(tmpDir, "data", "registry", "worktree")
	caDir := filepath.Join(worktree, "CertificateAuthority")
	if err := os.MkdirAll(caDir, 0755); err != nil {
		t.Fatalf("failed to create ca dir: %v", err)
	}

	// Initialize git repo in worktree so registry commits succeed
	runCmd := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	runCmd(worktree, "init")
	// Create initial commit and push to set upstream
	if err := os.WriteFile(filepath.Join(worktree, "README"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	runCmd(worktree, "add", "README")
	runCmd(worktree, "commit", "-m", "initial")
	runCmd(worktree, "remote", "add", "origin", worktree)
	runCmd(worktree, "push", "--set-upstream", "origin", "main")

	runCmd(worktree, "config", "user.email", "test@example.com")
	runCmd(worktree, "config", "user.name", "Test User")

	cfg := &config.Config{
		RootDir: tmpDir,
		CA: config.CAConfig{
			DefaultKeyType:   "ed25519",
			DefaultRSABits:   2048,
			DefaultValidDays: 1,
			Passphrase:       "test-passphrase",
		},
		Database: config.DatabaseConfig{
			Driver: "sqlite3",
			DSN:    filepath.Join(tmpDir, "test.db"),
		},
	}

	// Initialize and Migrate DB
	dbConn, err := db.Initialize(cfg)
	if err != nil {
		t.Fatalf("failed to initialize db: %v", err)
	}
	defer dbConn.Close()

	if err := db.RunMigrations(dbConn, cfg.Database.Driver); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	signer := &CASigner{}

	// 1. Test System Uninitialized (EnsureCA should not create)
	if err := EnsureCA(cfg, dbConn, signer); err != nil {
		t.Fatalf("EnsureCA failed: %v", err)
	}
	if signer.IsUnsealed() {
		t.Error("expected signer to be sealed when uninitialized")
	}

	// 2. Test Explicit Force Creation (manual step)
	if err := ForceCreateCA(cfg, dbConn, signer, "Test CA"); err != nil {
		t.Fatalf("ForceCreateCA failed: %v", err)
	}

	if !signer.IsUnsealed() {
		t.Error("expected signer to be unsealed after explicit ForceCreateCA")
	}

	// Verify local files exist
	localCaDir := filepath.Join(tmpDir, "data", "ca")
	files, _ := os.ReadDir(localCaDir)
	if len(files) < 2 { // uuid and uuid.pub
		t.Errorf("expected local CA files, found %d", len(files))
	}

	// 3. Test Restoration (local files gone, registry syncs to DB, then restores files from DB)
	// Clear local files
	os.RemoveAll(localCaDir)
	os.MkdirAll(localCaDir, 0700)
	signer.Seal()

	// EnsureCA should sync from registry to DB, detect missing files, 
	// and restore from the DB cache.
	if err := EnsureCA(cfg, dbConn, signer); err != nil {
		t.Fatalf("EnsureCA failed for restoration: %v", err)
	}

	if signer.IsUnsealed() {
		t.Error("expected signer to be sealed after EnsureCA restoration")
	}

	files, _ = os.ReadDir(localCaDir)
	if len(files) < 2 {
		t.Errorf("expected local CA files to be restored, found %d", len(files))
	}

	// 4. Test Re-use (local files exist)
	signer.Seal()
	if err := EnsureCA(cfg, dbConn, signer); err != nil {
		t.Fatalf("EnsureCA failed for re-use: %v", err)
	}

	if signer.IsUnsealed() {
		t.Error("expected signer to remain sealed after re-use check")
	}
}
