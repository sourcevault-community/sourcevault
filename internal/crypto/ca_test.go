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
	"bytes"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateCAKey_Ed25519(t *testing.T) {
	passphrase := []byte("test-passphrase")
	privPEM, pubAuth, err := GenerateCAKey("ed25519", 0, passphrase)
	if err != nil {
		t.Fatalf("GenerateCAKey failed: %v", err)
	}

	if len(privPEM) == 0 {
		t.Error("private key PEM is empty")
	}

	if !bytes.HasPrefix(privPEM, []byte("-----BEGIN OPENSSH PRIVATE KEY-----")) {
		t.Error("private key does not have expected OpenSSH prefix")
	}

	_, err = ssh.ParseRawPrivateKeyWithPassphrase(privPEM, passphrase)
	if err != nil {
		t.Errorf("failed to parse generated private key with passphrase: %v", err)
	}

	_, _, _, _, err = ssh.ParseAuthorizedKey(pubAuth)
	if err != nil {
		t.Errorf("failed to parse generated public key: %v", err)
	}
}

func TestGenerateCAKey_RSA(t *testing.T) {
	passphrase := []byte("test-passphrase")
	privPEM, pubAuth, err := GenerateCAKey("rsa", 2048, passphrase)
	if err != nil {
		t.Fatalf("GenerateCAKey failed: %v", err)
	}

	if len(privPEM) == 0 {
		t.Error("private key PEM is empty")
	}

	_, err = ssh.ParseRawPrivateKeyWithPassphrase(privPEM, passphrase)
	if err != nil {
		t.Errorf("failed to parse generated private key with passphrase: %v", err)
	}

	_, _, _, _, err = ssh.ParseAuthorizedKey(pubAuth)
	if err != nil {
		t.Errorf("failed to parse generated public key: %v", err)
	}
}

func TestGenerateCAKey_Unsupported(t *testing.T) {
	_, _, err := GenerateCAKey("unsupported", 0, []byte("pass"))
	if err == nil {
		t.Error("expected error for unsupported key type, got nil")
	}
}
