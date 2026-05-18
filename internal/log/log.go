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

package log

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"sourcevault/internal/config"
)

// Init initializes the global slog logger based on the provided configuration.
// It sets up either a file-based logger or a standard error logger,
// configures the log level, and adds source information to the logs
// with an optimized path-trimming strategy.
// It returns a cleanup function that should be called to close any open log files.
func Init(cfg *config.Config) func() {
	var out *os.File
	var err error

	// Determine the output destination
	if cfg.LogFile != "" {
		out, err = os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			slog.Error("failed to open log file", "path", cfg.LogFile, "error", err)
			os.Exit(1)
		}
	} else {
		out = os.Stderr
	}

	level := parseLogLevel(cfg.LogLevelValue())

	// Configure handler options.
	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// If we're looking at the source attribute (file:line), trim it.
			if a.Key == slog.SourceKey {
				source, ok := a.Value.Any().(*slog.Source)
				if ok {
					// Logic: Look for "sourcevault/" and trim everything before it.
					// This ensures logs are concise and independent of the build environment's path.
					fullPath := source.File
					needle := "sourcevault/"
					if idx := strings.Index(fullPath, needle); idx != -1 {
						source.File = fullPath[idx+len(needle):]
					} else {
						// Generic fallback: just keep the last two segments (e.g. log/log.go)
						dir := filepath.Base(filepath.Dir(fullPath))
						base := filepath.Base(fullPath)
						source.File = filepath.Join(dir, base)
					}
				}
			}
			return a
		},
	}

	// Use TextHandler for human-readable console/file logs.
	handler := slog.NewTextHandler(out, opts)
	slog.SetDefault(slog.New(handler))

	// Return a closure to handle safe cleanup of the output stream.
	return func() {
		if out != os.Stderr {
			out.Close()
		}
	}
}

// parseLogLevel converts a string representation of a log level into
// a slog.Level. It defaults to slog.LevelInfo if the input is unrecognized.
func parseLogLevel(value string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "DEBUG":
		return slog.LevelDebug
	case "ERROR":
		return slog.LevelError
	case "WARN", "WARNING":
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
