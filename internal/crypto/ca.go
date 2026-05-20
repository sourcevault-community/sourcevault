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

// Package crypto provides cryptographic primitives for the SourceVault CA system.
// This file handles SSH CA keypair generation.
package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/ssh"
)

// GenerateCAKey creates a new SSH CA private/public keypair.
// keyType must be "ed25519" (recommended) or "rsa". rsaBits is used only for RSA keys.
// The private key is encrypted using the provided passphrase in the modern OpenSSH format.
// Both the key type and RSA bit size are validated against the active crypto policy
// before generation proceeds.
func GenerateCAKey(keyType string, rsaBits int, passphrase []byte) (privPEM []byte, pubAuthorizedKey []byte, err error) {
	// Validate against the active crypto policy (currently a no-op stub; see SV-010).
	if err := ValidateKeyType(keyType); err != nil {
		return nil, nil, fmt.Errorf("crypto policy violation: %w", err)
	}

	var privateKey interface{}
	var publicKey ssh.PublicKey

	switch keyType {
	case "ed25519":
		slog.Debug("Generating Ed25519 CA keypair")
		pub, priv, err2 := ed25519.GenerateKey(rand.Reader)
		if err2 != nil {
			return nil, nil, fmt.Errorf("generating Ed25519 key: %w", err2)
		}
		privateKey = priv
		publicKey, err = ssh.NewPublicKey(pub)

	case "rsa":
		// Validate RSA bit size against policy before proceeding.
		if err := ValidateBits(rsaBits); err != nil {
			return nil, nil, fmt.Errorf("crypto policy violation: %w", err)
		}
		slog.Debug("Generating RSA CA keypair", "bits", rsaBits)
		priv, err2 := rsa.GenerateKey(rand.Reader, rsaBits)
		if err2 != nil {
			return nil, nil, fmt.Errorf("generating RSA-%d key: %w", rsaBits, err2)
		}
		privateKey = priv
		publicKey, err = ssh.NewPublicKey(&priv.PublicKey)

	default:
		return nil, nil, fmt.Errorf("unsupported key type %q: must be \"ed25519\" or \"rsa\"", keyType)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("creating SSH public key: %w", err)
	}

	// Encrypt the private key with the passphrase using the modern OpenSSH KDF format.
	privBlock, err := ssh.MarshalPrivateKeyWithPassphrase(privateKey, "SourceVault CA", passphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("encrypting CA private key: %w", err)
	}

	privPEM = pem.EncodeToMemory(privBlock)
	pubAuthorizedKey = ssh.MarshalAuthorizedKey(publicKey)
	return privPEM, pubAuthorizedKey, nil
}
