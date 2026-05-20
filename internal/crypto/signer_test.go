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
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestCASigner(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sourcevault-crypto-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	passphrase := []byte("secret")
	privPEM, _, err := GenerateCAKey("ed25519", 0, passphrase)
	if err != nil {
		t.Fatalf("GenerateCAKey failed: %v", err)
	}

	caPath := filepath.Join(tmpDir, "test_ca")
	if err := os.WriteFile(caPath, privPEM, 0600); err != nil {
		t.Fatalf("failed to write CA key: %v", err)
	}

	signer := &CASigner{}

	// Test Initial State (Sealed)
	if signer.IsUnsealed() {
		t.Error("signer should be sealed initially")
	}

	// Test Unseal
	if err := signer.UnsealFromPath(caPath, passphrase); err != nil {
		t.Fatalf("UnsealFromPath failed: %v", err)
	}
	if !signer.IsUnsealed() {
		t.Error("signer should be unsealed after UnsealFromPath")
	}
	if signer.CAPath() != caPath {
		t.Errorf("expected CAPath %s, got %s", caPath, signer.CAPath())
	}

	// Test Sign
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	sshPub, _ := ssh.NewPublicKey(pub)
	pubBytes := ssh.MarshalAuthorizedKey(sshPub)

	certConfig := CertConfig{
		CertType:   ssh.UserCert,
		KeyID:      "test-user",
		Principals: []string{"ubuntu"},
		ValidFor:   1 * time.Hour,
	}

	certBytes, serial, err := signer.Sign(pubBytes, certConfig)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if serial == 0 {
		t.Error("serial number should not be zero")
	}

	parsedCert, _, _, _, err := ssh.ParseAuthorizedKey(certBytes)
	if err != nil {
		t.Fatalf("failed to parse signed certificate: %v", err)
	}
	cert, ok := parsedCert.(*ssh.Certificate)
	if !ok {
		t.Fatal("parsed key is not a certificate")
	}

	if cert.KeyId != certConfig.KeyID {
		t.Errorf("expected KeyID %s, got %s", certConfig.KeyID, cert.KeyId)
	}
	if len(cert.ValidPrincipals) != 1 || cert.ValidPrincipals[0] != "ubuntu" {
		t.Errorf("expected principals [ubuntu], got %v", cert.ValidPrincipals)
	}

	// Test Seal
	signer.Seal()
	if signer.IsUnsealed() {
		t.Error("signer should be sealed after Seal()")
	}
	if signer.CAPath() != "" {
		t.Error("CAPath should be empty after Seal()")
	}

	// Test Sign while Sealed
	_, _, err = signer.Sign(pubBytes, certConfig)
	if err == nil {
		t.Error("expected error when signing while sealed, got nil")
	}
}
