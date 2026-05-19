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

// Package registry provides helpers for executing Git commands required by the
// SourceVault system registry bootstrap process. All functions log the command
// being run at DEBUG level and capture stderr on failure for clear error messages.
package registry

import (
	"bytes"
	"fmt"
	"log/slog"
	"os/exec"
)

// runGit executes a git command in the given working directory. It logs the
// command at DEBUG level before running, and returns a wrapped error containing
// stderr output if the command fails.
func runGit(workDir string, args ...string) error {
	slog.Debug("Running git command", "args", args, "dir", workDir)

	cmd := exec.Command("git", args...)
	cmd.Dir = workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %v: %w\nstderr: %s", args, err, stderr.String())
	}
	return nil
}

// gitInitBare initializes a new bare Git repository at the given path.
func gitInitBare(path string) error {
	return runGit(".", "init", "--bare", path)
}

// gitConfigSet sets a local git config value inside the given repository.
func gitConfigSet(repoPath, key, value string) error {
	return runGit(repoPath, "config", key, value)
}

// gitClone clones src into dst, checking out the specified branch.
func gitClone(src, dst, branch string) error {
	return runGit(".", "clone", "--branch", branch, src, dst)
}

// gitCloneNoCheckout clones src into dst without checking out any branch.
// Used when the bare repo has no commits yet and we need to push an initial one.
func gitCloneNoCheckout(src, dst string) error {
	return runGit(".", "clone", src, dst)
}

// gitAdd stages files matching the given pattern in the given worktree.
func gitAdd(worktree, pattern string) error {
	return runGit(worktree, "add", pattern)
}

// gitCommit creates a commit in the given worktree with the provided message.
func gitCommit(worktree, message string) error {
	return runGit(worktree, "commit", "--allow-empty", "-m", message)
}

// gitPush pushes the current branch to the given remote.
func gitPush(worktree, remote string) error {
	return runGit(worktree, "push", remote)
}

// gitFetch fetches the latest state from the remote in the given worktree.
func gitFetch(worktree string) error {
	return runGit(worktree, "fetch", "origin")
}

// gitResetHard resets the worktree to exactly match the remote branch.
// This is intentionally not a merge — it guarantees no conflicts on startup.
func gitResetHard(worktree, branch string) error {
	return runGit(worktree, "reset", "--hard", "origin/"+branch)
}
