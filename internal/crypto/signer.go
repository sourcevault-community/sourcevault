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

// Package crypto provides the in-memory CA signer (the "unseal" mechanism).
package crypto

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// CASigner holds the unsealed CA state in memory.
// Instantiated once in start.go and injected wherever signing is needed.
// The zero value is a valid, sealed signer.
type CASigner struct {
	mu           sync.RWMutex
	activeSigner ssh.Signer // nil when sealed
	caPath       string
}

// UnsealFromPath reads the encrypted CA key from disk and loads it into memory.
func (s *CASigner) UnsealFromPath(caPath string, passphrase []byte) error {
	slog.Info("Attempting to unseal CA", "path", caPath)

	keyPEM, err := os.ReadFile(caPath)
	if err != nil {
		return fmt.Errorf("reading CA key from %s: %w", caPath, err)
	}

	caPrivateKey, err := ssh.ParseRawPrivateKeyWithPassphrase(keyPEM, passphrase)
	if err != nil {
		slog.Error("CA unseal failed", "path", caPath, "error", err)
		return fmt.Errorf("parsing CA key (invalid passphrase?): %w", err)
	}

	signer, err := ssh.NewSignerFromKey(caPrivateKey)
	if err != nil {
		return fmt.Errorf("creating signer: %w", err)
	}

	s.mu.Lock()
	s.activeSigner = signer
	s.caPath = caPath
	s.mu.Unlock()

	slog.Info("CA unsealed successfully", "path", caPath)
	return nil
}

// Seal clears the in-memory signer. Sign() calls will be rejected until Unseal is called again.
func (s *CASigner) Seal() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.activeSigner = nil
	s.caPath = ""
	slog.Info("CA sealed — signing key cleared from memory")
}

// IsUnsealed reports whether a decrypted CA key is currently held in memory.
func (s *CASigner) IsUnsealed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeSigner != nil
}

// CAPath returns the path of the currently loaded CA key, or empty if sealed.
func (s *CASigner) CAPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.caPath
}

// Sign issues a signed SSH certificate for the given public key.
// Returns the marshaled certificate and its serial number.
// Ported from the reference daemon.go Sign() implementation.
func (s *CASigner) Sign(pubKeyBytes []byte, cfg CertConfig) (certBytes []byte, serial uint64, err error) {
	s.mu.RLock()
	signer := s.activeSigner
	s.mu.RUnlock()

	if signer == nil {
		return nil, 0, fmt.Errorf("CA is sealed: run 'sourcevault ca unseal' first")
	}

	userPubKey, _, _, _, err := ssh.ParseAuthorizedKey(pubKeyBytes)
	if err != nil {
		return nil, 0, fmt.Errorf("parsing target public key: %w", err)
	}

	// Generate a cryptographically secure random serial number.
	serialBytes := make([]byte, 8)
	if _, err := rand.Read(serialBytes); err != nil {
		return nil, 0, fmt.Errorf("generating serial number: %w", err)
	}
	serial = binary.BigEndian.Uint64(serialBytes) & 0x7FFFFFFFFFFFFFFF
	if serial == 0 {
		serial = 1
	}

	now := time.Now()

	cert := &ssh.Certificate{
		Key:             userPubKey,
		CertType:        cfg.CertType,
		KeyId:           cfg.KeyID,
		Serial:          serial,
		ValidPrincipals: cfg.Principals,
		ValidAfter:      uint64(now.Unix()),
		ValidBefore:     uint64(now.Add(cfg.ValidFor).Unix()),
		Permissions: ssh.Permissions{
			Extensions: map[string]string{
				"permit-pty":              "",
				"permit-port-forwarding":  "",
				"permit-agent-forwarding": "",
				"permit-X11-forwarding":   "",
				"permit-user-rc":          "",
			},
		},
	}

	if err := cert.SignCert(rand.Reader, signer); err != nil {
		return nil, 0, fmt.Errorf("signing certificate: %w", err)
	}

	slog.Info("Certificate signed",
		"key_id", cfg.KeyID,
		"serial", serial,
		"valid_for", cfg.ValidFor,
		"cert_type", cfg.CertType,
	)

	return ssh.MarshalAuthorizedKey(cert), cert.Serial, nil
}
