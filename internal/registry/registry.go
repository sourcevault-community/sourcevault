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

// Package registry manages the Git-based system registry — SourceVault's source of truth.
// It maintains a bare repository (system.git) and a checked-out worktree so that the
// application can read YAML configuration at runtime without touching the bare repo directly.
package registry

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"sourcevault/internal/config"
)

// worktreeDirs lists the top-level directories that must exist inside the registry worktree.
// Each represents a distinct domain of configuration state stored as {uuid}.yaml files.
var worktreeDirs = []string{
	"Users",
	"Volumes",
	"Repositories",
	"Organizations",
	"CertificateAuthority",
}

// EnsureRegistry guarantees the system registry is in a consistent, readable state before
// the rest of the application starts. It handles three distinct scenarios:
//
//  1. Fresh install: initializes the bare repo and clones a worktree.
//  2. Existing install, first boot: clones the worktree from the existing bare repo.
//  3. Existing install: force-syncs the worktree from the bare repo (no merges, no conflicts).
func EnsureRegistry(cfg *config.Config) error {
	bareRepo := filepath.Join(cfg.RootDir, "registry", "system.git")
	worktree := filepath.Join(cfg.RootDir, "registry", "worktree")
	branch := cfg.Registry.Branch

	// Step 1: Initialize the bare repository if it does not already exist.
	if _, err := os.Stat(bareRepo); os.IsNotExist(err) {
		if err := initBareRepo(bareRepo, branch); err != nil {
			return fmt.Errorf("initializing registry bare repo: %w", err)
		}
	} else {
		slog.Debug("Registry bare repository already exists, skipping init", "path", bareRepo)
	}

	// Step 2: Clone or sync the worktree.
	if _, err := os.Stat(worktree); os.IsNotExist(err) {
		slog.Info("Registry worktree not found, cloning from bare repository", "branch", branch)
		if err := gitClone(bareRepo, worktree, branch); err != nil {
			return fmt.Errorf("cloning registry worktree: %w", err)
		}
		slog.Info("Registry worktree cloned successfully")
	} else {
		// Worktree exists — force-sync it to exactly match the bare repo.
		// We use fetch + reset --hard instead of pull to guarantee no merge conflicts on startup.
		slog.Info("Registry worktree found, force-syncing with bare repository", "branch", branch)
		if err := gitFetch(worktree); err != nil {
			return fmt.Errorf("fetching registry updates: %w", err)
		}
		if err := gitResetHard(worktree, branch); err != nil {
			return fmt.Errorf("resetting registry worktree: %w", err)
		}
		slog.Info("Registry worktree synced successfully")
	}

	return nil
}

// initBareRepo initializes a new bare Git repository and pushes an initial commit
// containing the required top-level directory structure.
func initBareRepo(bareRepo, branch string) error {
	slog.Info("Initializing new system registry bare repository", "path", bareRepo)

	// Create the bare repo.
	if err := gitInitBare(bareRepo); err != nil {
		return fmt.Errorf("git init --bare: %w", err)
	}

	// Set a consistent Git identity for all automated commits made by SourceVault.
	// This avoids "Please tell me who you are" errors in environments without a global git config.
	if err := gitConfigSet(bareRepo, "user.email", "noreply@sourcevault"); err != nil {
		return err
	}
	if err := gitConfigSet(bareRepo, "user.name", "SourceVault"); err != nil {
		return err
	}

	// Bootstrap the initial commit via a temporary clone.
	// A freshly initialised bare repo has no commits and no branch yet, so we
	// cannot clone it with --branch. We clone without checkout, build the structure,
	// commit, and push — which creates the branch inside the bare repo.
	slog.Info("Bootstrapping initial registry directory structure")

	tmpDir, err := os.MkdirTemp("", "sourcevault-registry-bootstrap-*")
	if err != nil {
		return fmt.Errorf("creating temp dir for bootstrap: %w", err)
	}
	defer func() {
		os.RemoveAll(tmpDir)
		slog.Debug("Cleaned up bootstrap temp directory", "dir", tmpDir)
	}()

	// Clone the empty bare repo into the temp dir.
	tmpWorktree := filepath.Join(tmpDir, "worktree")
	if err := gitCloneNoCheckout(bareRepo, tmpWorktree); err != nil {
		return fmt.Errorf("cloning for bootstrap: %w", err)
	}

	// Set identity on the temp clone so commits succeed without a global git config.
	if err := gitConfigSet(tmpWorktree, "user.email", "noreply@sourcevault"); err != nil {
		return err
	}
	if err := gitConfigSet(tmpWorktree, "user.name", "SourceVault"); err != nil {
		return err
	}

	// Create each top-level directory with a .gitkeep so it is tracked by Git.
	for _, dir := range worktreeDirs {
		dirPath := filepath.Join(tmpWorktree, dir)
		if err := os.MkdirAll(dirPath, 0o750); err != nil {
			return fmt.Errorf("creating registry dir %s: %w", dir, err)
		}
		// Write a .gitkeep so the empty directory is tracked by Git.
		keepPath := filepath.Join(dirPath, ".gitkeep")
		if err := os.WriteFile(keepPath, []byte{}, 0o640); err != nil {
			return fmt.Errorf("writing .gitkeep for %s: %w", dir, err)
		}
		slog.Debug("Created registry directory", "dir", dir)
	}

	// Stage all new files, commit, and push to the bare repo to create the branch.
	if err := gitAdd(tmpWorktree, "."); err != nil {
		return fmt.Errorf("staging initial registry structure: %w", err)
	}
	if err := gitCommit(tmpWorktree, "chore: initialize system registry structure"); err != nil {
		return fmt.Errorf("committing initial registry structure: %w", err)
	}
	if err := gitPush(tmpWorktree, "origin"); err != nil {
		return fmt.Errorf("pushing initial registry structure: %w", err)
	}

	slog.Info("System registry initialized successfully")
	return nil
}
