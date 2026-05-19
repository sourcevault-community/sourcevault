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
//  1. Fresh install: initializes the bare repo and pushes the initial directory structure.
//  2. Bare repo exists but has no commits (e.g. previous run crashed): re-runs the bootstrap.
//  3. Existing install: clones the worktree if missing, or force-syncs if already present.
func EnsureRegistry(cfg *config.Config) error {
	bareRepo := filepath.Join(cfg.RootDir, "registry", "system.git")
	worktree := filepath.Join(cfg.RootDir, "registry", "worktree")
	branch := cfg.Registry.Branch

	// Step 1: Initialize the bare repository if it does not already exist.
	if _, err := os.Stat(bareRepo); os.IsNotExist(err) {
		slog.Info("Initializing new system registry bare repository", "path", bareRepo)
		if err := gitInitBare(bareRepo); err != nil {
			return fmt.Errorf("git init --bare: %w", err)
		}
		// Set a consistent Git identity for all automated commits.
		// This avoids "Please tell me who you are" errors in environments
		// without a global git config present.
		if err := gitConfigSet(bareRepo, "user.email", "noreply@sourcevault"); err != nil {
			return err
		}
		if err := gitConfigSet(bareRepo, "user.name", "SourceVault"); err != nil {
			return err
		}
	} else {
		slog.Debug("Registry bare repository already exists", "path", bareRepo)
	}

	// Step 2: Bootstrap an initial commit if the bare repo has no commits yet.
	// This handles both fresh installs and partially-initialized repos (e.g. a
	// previous run that crashed after git init but before the push succeeded).
	if !bareRepoHasCommits(bareRepo) {
		if err := bootstrapInitialCommit(bareRepo, branch); err != nil {
			return fmt.Errorf("bootstrapping registry: %w", err)
		}
	}

	// Step 3: Clone or sync the worktree.
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

// bareRepoHasCommits checks whether a bare repository has at least one commit.
// A freshly initialized bare repo has no HEAD ref, so rev-parse will fail.
func bareRepoHasCommits(bareRepo string) bool {
	err := runGit(bareRepo, "rev-parse", "HEAD")
	return err == nil
}

// bootstrapInitialCommit creates the initial directory structure and pushes the
// first commit to the bare repository, which establishes the default branch.
func bootstrapInitialCommit(bareRepo, branch string) error {
	slog.Info("Bootstrapping initial registry directory structure")

	// Use a temporary clone to build and push the initial commit, since you
	// cannot commit directly to a bare repository.
	tmpDir, err := os.MkdirTemp("", "sourcevault-registry-bootstrap-*")
	if err != nil {
		return fmt.Errorf("creating temp dir for bootstrap: %w", err)
	}
	defer func() {
		os.RemoveAll(tmpDir)
		slog.Debug("Cleaned up bootstrap temp directory", "dir", tmpDir)
	}()

	tmpWorktree := filepath.Join(tmpDir, "worktree")

	// Clone the bare repo without specifying a branch — there are no branches yet.
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

	// Create each top-level directory with a .gitkeep so Git tracks them.
	for _, dir := range worktreeDirs {
		dirPath := filepath.Join(tmpWorktree, dir)
		if err := os.MkdirAll(dirPath, 0o750); err != nil {
			return fmt.Errorf("creating registry dir %s: %w", dir, err)
		}
		keepPath := filepath.Join(dirPath, ".gitkeep")
		if err := os.WriteFile(keepPath, []byte{}, 0o640); err != nil {
			return fmt.Errorf("writing .gitkeep for %s: %w", dir, err)
		}
		slog.Debug("Created registry directory", "dir", dir)
	}

	// Rename the default branch to match the configured branch name before committing.
	if err := runGit(tmpWorktree, "checkout", "-b", branch); err != nil {
		return fmt.Errorf("checking out branch %s: %w", branch, err)
	}

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
