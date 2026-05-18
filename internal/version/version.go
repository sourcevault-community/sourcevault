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


package version

import (
	"runtime"
)

// VersionInfo holds all build and version details about the application.
// This information is primarily populated at build time using ldflags
// in the Makefile, allowing the binary to report its own identity.
type VersionInfo struct {
	AppName      string // AppName is the formal name of the application (e.g., "sourcevault").
	AppVersion   string // AppVersion is the semantic version string (e.g., "0.1.0").
	GitCommit    string // GitCommit is the short git commit hash from which the app was built.
	GitBranch    string // GitBranch is the name of the git branch used during the build.
	BuildDate    string // BuildDate is the UTC timestamp indicating when the binary was compiled.
	Architecture string // Architecture is the target CPU architecture (e.g., amd64, arm64).
}

var (
	// appName is the default application name, used if not overridden during build.
	appName = "sourcevault"
	// appVersion is the default semantic version.
	appVersion = "0.1.0"
	// gitCommit is a placeholder for the git commit hash.
	gitCommit = "UNKNOWN"
	// gitBranch is a placeholder for the git branch name.
	gitBranch = "UNKNOWN"
	// buildDate is a placeholder for the build timestamp.
	buildDate = "UNKNOWN"
)

// Current contains the version information for the running application instance.
// The fields are initialized using the variables defined above, which are
// expected to be overridden by Makefile LDFLAGS at compile time.
var Current = VersionInfo{
	AppName:      appName,
	AppVersion:   appVersion,
	GitBranch:    gitBranch,
	GitCommit:    gitCommit,
	BuildDate:    buildDate,
	Architecture: runtime.GOARCH,
}
