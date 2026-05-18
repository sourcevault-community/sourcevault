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

package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sourcevault"
	"sourcevault/internal/config"
	"time"
)

// responseWriter is a minimal wrapper around http.ResponseWriter that captures
// the HTTP status code. This allows the logging middleware to report the
// outcome of each request.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before passing it to the underlying
// ResponseWriter.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware provides structured logging for every HTTP request.
// It records the method, URL path, remote client address, resulting status code,
// and the total processing duration.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Default to 200 OK in case WriteHeader is never called explicitly.
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		args := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"status", rw.statusCode,
			"duration", duration,
		}

		// Include X-Forwarded-For if it exists to help identify the original client IP.
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			args = append(args, "x_forwarded_for", xff)
		}

		slog.Info("HTTP request", args...)
	})
}

func Handler() http.Handler {
	mux := http.NewServeMux()

	// Root handler serves the modern Coming Soon template.
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		content, err := sourcevault.Templates.ReadFile("templates/coming_soon.html")
		if err != nil {
			slog.Error("failed to read coming_soon template", "error", err)
			http.Error(w, "Template not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(content)
	})

	// Serve the embedded logo image.
	mux.HandleFunc("GET /logo.png", func(w http.ResponseWriter, r *http.Request) {
		content, err := sourcevault.Templates.ReadFile("templates/sourcevault-logo.png")
		if err != nil {
			slog.Error("failed to read logo image", "error", err)
			http.Error(w, "Image not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(content)
	})

	// Serve the embedded favicon.
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		content, err := sourcevault.Templates.ReadFile("templates/favicon.ico")
		if err != nil {
			slog.Error("failed to read favicon", "error", err)
			http.Error(w, "Favicon not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(content)
	})

	// Status endpoint for health checks.
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Status: OK")
	})

	// Example path parameter handler for name greetings.
	// Uses Go 1.22+ mux path value extraction.
	mux.HandleFunc("GET /hello/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		fmt.Fprintf(w, "Hello, %s!\n", name)
	})

	// Wrap the entire mux with the logging middleware.
	return loggingMiddleware(mux)
}

func Run(ctx context.Context, cfg *config.Config) error {
	slog.Info("Starting web server", "host", cfg.Web.Host, "port", cfg.Web.Port)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Web.Host, cfg.Web.Port),
		Handler: Handler(),
	}

	// Monitor the context for cancellation and trigger server shutdown.
	go func() {
		<-ctx.Done()
		slog.Info("Shutting down web server")
		srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("web server error: %w", err)
	}

	return nil
}
