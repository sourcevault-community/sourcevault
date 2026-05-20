// SPDX-License-Identifier: AGPL-3.0-or-later
// SPDX-FileCopyrightText: 2026 The sourcevault Authors. All rights reserved.
// ===================================================================================================================================== //
// MP""""""`MM MMP"""""YMM M""MMMMM""M MM"""""""`MM MM'""""'YMM MM""""""""`M M""MMMMM""M MMP"""""""MM M""MMMMM""M M""MMMMMMMM M""""""""M //
// M  mmmmm..M M' .mmm. `M M  MMMMM  M MM  mmmm,  M M' .mmm. `M MM  mmmmmmmM M  MMMMM  M M' .mmmm  MM M  MMMMM  M M  MMMMMMMM Mmmm  mmmM //
// M.      `YM M  MMMMM  M M  MMMMM  M M'        .M M  MMMMMooM M`      MMMM M  MMMMP  M M         `M M  MMMMM  M M  MMMMMMMM MMMM  MMMM //
// MMMMMMM.  M M  MMMMM  M M  MMMMM  M MM  MMMb. "M M  MMMMMMMM MM  MMMMMMMM M  MMMM' .M M  MMMMM  MM M  MMMMM  M M  MMMMMMMM MMMM  MMMM //
// M. .MMM'  M M. `MMM' .M M  `MMM'  M MM  MMMMM  M M. `MMM' .M MM  MMMMMMMM M  MMP' .MM M  MMMMM  MM Mb       dM M         M MMMM  MMMM //
// MMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMMM MMMMMMMMMM //
// ===================================================================================================================================== //

package main

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"

	"sourcevault/internal/crypto"
	"sourcevault/internal/registry"
	sv_rpc "sourcevault/internal/rpc"
)

// caCmd is the root "sourcevault ca" subcommand.
var caCmd = &cobra.Command{
	Use:   "ca",
	Short: "Manage the SourceVault Certificate Authority",
	Long:  "Create, rotate, revoke, unseal, and seal the local SSH Certificate Authority.",
}

// caCreateCmd handles "sourcevault ca create".
var caCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new CA keypair",
	RunE: func(cmd *cobra.Command, args []string) error {
		keyType, _ := cmd.Flags().GetString("key-type")
		rsaBits, _ := cmd.Flags().GetInt("rsa-bits")
		validFor, _ := cmd.Flags().GetDuration("valid-for")
		name, _ := cmd.Flags().GetString("name")

		if keyType == "" {
			keyType = appCfg.CA.DefaultKeyType
		}
		if rsaBits == 0 {
			rsaBits = appCfg.CA.DefaultRSABits
		}
		if validFor == 0 {
			validFor = time.Duration(appCfg.CA.DefaultValidDays) * 24 * time.Hour
		}

		// Prompt for passphrase with confirmation.
		passphrase, err := promptPassphrase("Enter passphrase to encrypt CA private key: ")
		if err != nil {
			return fmt.Errorf("reading passphrase: %w", err)
		}
		confirm, err := promptPassphrase("Confirm passphrase: ")
		if err != nil {
			return fmt.Errorf("reading passphrase confirmation: %w", err)
		}
		if string(passphrase) != string(confirm) {
			return fmt.Errorf("passphrases do not match")
		}

		slog.Info("Generating CA keypair", "key_type", keyType, "valid_for", validFor)

		privPEM, pubKey, err := crypto.GenerateCAKey(keyType, rsaBits, passphrase)
		if err != nil {
			return fmt.Errorf("generating CA key: %w", err)
		}

		// Derive paths and write key files.
		caID := uuid.New().String()
		caDir := filepath.Join(appCfg.RootDir, "data", "ca")
		privPath := filepath.Join(caDir, caID)
		pubPath := privPath + ".pub"

		if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
			return fmt.Errorf("writing private key: %w", err)
		}
		slog.Info("CA private key written", "path", privPath)

		if err := os.WriteFile(pubPath, pubKey, 0o644); err != nil {
			return fmt.Errorf("writing public key: %w", err)
		}
		slog.Info("CA public key written", "path", pubPath)

		// Compute fingerprint from the public key bytes.
		parsedPub, _, _, _, err := ssh.ParseAuthorizedKey(pubKey)
		if err != nil {
			return fmt.Errorf("parsing generated public key: %w", err)
		}
		sum := sha256.Sum256(parsedPub.Marshal())
		fingerprint := ssh.FingerprintSHA256(parsedPub)

		_ = sum // fingerprint string is already computed via ssh package

		// Write public metadata to the registry.
		now := time.Now().UTC()
		meta := registry.CAMetadata{
			UUID:                caID,
			Name:                name,
			Algorithm:           keyType,
			Fingerprint:         fingerprint,
			EncryptedPrivateKey: string(privPEM),
			PublicKey:           string(pubKey),
			ValidFrom:           now,
			ValidUntil:          now.Add(validFor),
			CreatedAt:           now,
			Revoked:             false,
		}
		if err := registry.SaveCAMetadata(appCfg, meta); err != nil {
			return fmt.Errorf("saving CA metadata to registry: %w", err)
		}

		fmt.Printf("\nCA created successfully\n")
		fmt.Printf("  UUID:        %s\n", caID)
		fmt.Printf("  Algorithm:   %s\n", keyType)
		fmt.Printf("  Fingerprint: %s\n", fingerprint)
		fmt.Printf("  Valid until: %s\n", now.Add(validFor).Format(time.RFC3339))
		fmt.Printf("  Private key: %s\n", privPath)
		fmt.Printf("  Public key:  %s\n", pubPath)
		return nil
	},
}

// caUnsealCmd handles "sourcevault ca unseal".
var caUnsealCmd = &cobra.Command{
	Use:   "unseal",
	Short: "Decrypt and load the active CA key into memory",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Find the active CA metadata from the registry.
		active, err := registry.GetActiveCA(appCfg)
		if err != nil {
			return fmt.Errorf("getting active CA from registry: %w", err)
		}
		if active == nil {
			return fmt.Errorf("no active CA found: run 'sourcevault ca create' first")
		}

		caPath := filepath.Join(appCfg.RootDir, "data", "ca", active.UUID)

		passphrase, err := promptPassphrase("Enter CA passphrase: ")
		if err != nil {
			return fmt.Errorf("reading passphrase: %w", err)
		}

		// Attempt to talk to the running server via RPC
		client, err := sv_rpc.GetClient(appCfg.Sockets.SourceVault)
		if err == nil {
			defer client.Close()
			var reply sv_rpc.UnsealReply
			if err := client.Call("CAService.Unseal", &sv_rpc.UnsealArgs{
				CAPath:     caPath,
				Passphrase: passphrase,
			}, &reply); err != nil {
				return fmt.Errorf("RPC error: %w", err)
			}
			if reply.Success {
				fmt.Println("CA unsealed successfully (Server)")
			}
			return nil
		}

		if err := appSigner.UnsealFromPath(caPath, passphrase); err != nil {
			return err
		}

		fmt.Println("CA unsealed. Certificate signing is now available (Local Process).")
		return nil
	},
}

// caSealCmd handles "sourcevault ca seal".
var caSealCmd = &cobra.Command{
	Use:   "seal",
	Short: "Clear the CA key from memory",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Attempt to talk to the running server via RPC
		client, err := sv_rpc.GetClient(appCfg.Sockets.SourceVault)
		if err == nil {
			defer client.Close()
			var reply sv_rpc.SealReply
			if err := client.Call("CAService.Seal", &sv_rpc.SealArgs{}, &reply); err != nil {
				return fmt.Errorf("RPC error: %w", err)
			}
			if reply.Success {
				fmt.Println("CA sealed successfully (Server)")
			}
			return nil
		}

		appSigner.Seal()
		fmt.Println("CA sealed (Local Process).")
		return nil
	},
}

// caStatusCmd handles "sourcevault ca status".
var caStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the CA is currently unsealed",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Attempt to talk to the running server via RPC
		client, err := sv_rpc.GetClient(appCfg.Sockets.SourceVault)
		if err == nil {
			defer client.Close()
			var reply sv_rpc.StatusReply
			if err := client.Call("CAService.Status", &sv_rpc.StatusArgs{}, &reply); err != nil {
				return fmt.Errorf("RPC error: %w", err)
			}
			fmt.Printf("CA Status (Server):\n")
			fmt.Printf("  Unsealed:  %t\n", reply.IsUnsealed)
			if reply.CAPath != "" {
				fmt.Printf("  CA Path:   %s\n", reply.CAPath)
			}
			return nil
		}

		// Fallback to local process memory
		if appSigner.IsUnsealed() {
			fmt.Printf("CA Status (Local Process):\n")
			fmt.Printf("  Unsealed:  true\nLoaded key: %s\n", appSigner.CAPath())
		} else {
			fmt.Printf("CA Status (Local Process):\n")
			fmt.Printf("  Unsealed:  false\n")
		}
		return nil
	},
}

// caRevokeCmd handles "sourcevault ca revoke".
var caRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Mark a CA as revoked in the registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		caUUID, _ := cmd.Flags().GetString("uuid")
		if caUUID == "" {
			return fmt.Errorf("--uuid is required")
		}

		if err := registry.RevokeCAMetadata(appCfg, caUUID); err != nil {
			return fmt.Errorf("revoking CA: %w", err)
		}

		fmt.Printf("CA %s has been revoked in the registry.\n", caUUID)
		return nil
	},
}

// caRotateCmd handles "sourcevault ca rotate" — creates a new CA and revokes the old one.
var caRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Create a new CA and revoke the existing one",
	RunE: func(cmd *cobra.Command, args []string) error {
		oldUUID, _ := cmd.Flags().GetString("revoke-uuid")
		if oldUUID == "" {
			return fmt.Errorf("--revoke-uuid is required: provide the UUID of the CA to replace")
		}

		// Create new CA first so the node is never left without a valid CA.
		slog.Info("Rotating CA — creating new keypair")
		if err := caCreateCmd.RunE(caCreateCmd, args); err != nil {
			return fmt.Errorf("creating new CA during rotate: %w", err)
		}

		// Revoke the old CA after the new one is in place.
		slog.Info("Revoking old CA", "uuid", oldUUID)
		if err := registry.RevokeCAMetadata(appCfg, oldUUID); err != nil {
			return fmt.Errorf("revoking old CA during rotate: %w", err)
		}

		fmt.Printf("CA rotation complete. Old CA %s has been revoked.\n", oldUUID)
		return nil
	},
}

// caSignCmd handles "sourcevault ca sign".
var caSignCmd = &cobra.Command{
	Use:   "sign [public-key-file]",
	Short: "Sign a public key with the unsealed CA",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pubKeyPath := args[0]
		pubKeyBytes, err := os.ReadFile(pubKeyPath)
		if err != nil {
			return fmt.Errorf("reading public key: %w", err)
		}

		certTypeStr, _ := cmd.Flags().GetString("type")
		keyID, _ := cmd.Flags().GetString("id")
		principals, _ := cmd.Flags().GetStringSlice("principals")
		validFor, _ := cmd.Flags().GetDuration("valid-for")

		var certType uint32
		switch certTypeStr {
		case "user":
			certType = ssh.UserCert
		case "host":
			certType = ssh.HostCert
		default:
			return fmt.Errorf("invalid certificate type %q: must be \"user\" or \"host\"", certTypeStr)
		}

		if validFor == 0 {
			validFor = time.Duration(appCfg.CA.DefaultValidDays) * 24 * time.Hour
		}

		certConfig := crypto.CertConfig{
			CertType:   certType,
			KeyID:      keyID,
			Principals: principals,
			ValidFor:   validFor,
		}

		var certBytes []byte
		var serial uint64

		// Attempt to talk to the running server via RPC
		client, err := sv_rpc.GetClient(appCfg.Sockets.SourceVault)
		if err == nil {
			defer client.Close()
			slog.Info("Requesting certificate signing via RPC", "key", pubKeyPath)
			var reply sv_rpc.SignReply
			if err := client.Call("CAService.Sign", &sv_rpc.SignArgs{
				PublicKey: pubKeyBytes,
				Config:    certConfig,
			}, &reply); err != nil {
				return fmt.Errorf("RPC error: %w", err)
			}
			certBytes = reply.SignedCert
			serial = reply.Serial
		} else {
			if !appSigner.IsUnsealed() {
				return fmt.Errorf("CA is sealed: run 'sourcevault ca unseal' first or ensure server is running")
			}

			slog.Info("Signing certificate locally", "key", pubKeyPath, "id", keyID, "type", certTypeStr)

			certBytes, serial, err = appSigner.Sign(pubKeyBytes, certConfig)
			if err != nil {
				return fmt.Errorf("signing failed: %w", err)
			}
		}

		// Write the certificate to [key]-cert.pub
		certPath := pubKeyPath
		if filepath.Ext(pubKeyPath) == ".pub" {
			certPath = pubKeyPath[:len(pubKeyPath)-4] + "-cert.pub"
		} else {
			certPath = pubKeyPath + "-cert.pub"
		}

		if err := os.WriteFile(certPath, certBytes, 0o644); err != nil {
			return fmt.Errorf("writing certificate: %w", err)
		}

		fmt.Printf("Certificate signed successfully\n")
		fmt.Printf("  Serial:      %d\n", serial)
		fmt.Printf("  Certificate: %s\n", certPath)
		return nil
	},
}

func init() {
	// ca create flags
	caCreateCmd.Flags().String("key-type", "", "Key algorithm: ed25519 or rsa (default from config)")
	caCreateCmd.Flags().Int("rsa-bits", 0, "RSA key size in bits (default from config, only used with --key-type=rsa)")
	caCreateCmd.Flags().Duration("valid-for", 0, "Certificate validity period e.g. 8760h (default from config)")
	caCreateCmd.Flags().String("name", "", "Human-readable label for this CA")

	// ca revoke flags
	caRevokeCmd.Flags().String("uuid", "", "UUID of the CA to revoke")

	// ca rotate flags
	caRotateCmd.Flags().String("revoke-uuid", "", "UUID of the CA being replaced")
	caRotateCmd.Flags().String("key-type", "", "Key algorithm for the new CA")
	caRotateCmd.Flags().Int("rsa-bits", 0, "RSA key size for the new CA")
	caRotateCmd.Flags().Duration("valid-for", 0, "Validity period for the new CA")
	caRotateCmd.Flags().String("name", "", "Name for the new CA")

	// ca sign flags
	caSignCmd.Flags().String("type", "user", "Certificate type: user or host")
	caSignCmd.Flags().String("id", "", "Key ID (comment) to embed in the certificate")
	caSignCmd.Flags().StringSlice("principals", nil, "List of valid principals (comma separated)")
	caSignCmd.Flags().Duration("valid-for", 0, "Certificate validity period e.g. 24h (default from config)")

	// Register all subcommands under caCmd.
	caCmd.AddCommand(caCreateCmd, caUnsealCmd, caSealCmd, caStatusCmd, caRevokeCmd, caRotateCmd, caSignCmd)
}

// promptPassphrase reads a passphrase from the terminal without echoing it.
func promptPassphrase(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	pass, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	return pass, err
}
