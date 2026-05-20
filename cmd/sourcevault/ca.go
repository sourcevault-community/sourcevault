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

		// Write public metadata to the registry (private key material is never included).
		now := time.Now().UTC()
		meta := registry.CAMetadata{
			UUID:        caID,
			Name:        name,
			Algorithm:   keyType,
			Fingerprint: fingerprint,
			ValidFrom:   now,
			ValidUntil:  now.Add(validFor),
			CreatedAt:   now,
			Revoked:     false,
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
	Short: "Decrypt and load the CA key into memory",
	RunE: func(cmd *cobra.Command, args []string) error {
		caPath, _ := cmd.Flags().GetString("key")
		if caPath == "" {
			return fmt.Errorf("--key is required: specify the path to the CA private key file")
		}

		passphrase, err := promptPassphrase("Enter CA passphrase: ")
		if err != nil {
			return fmt.Errorf("reading passphrase: %w", err)
		}

		if err := appSigner.UnsealFromPath(caPath, passphrase); err != nil {
			return err
		}

		fmt.Println("CA unsealed. Certificate signing is now available.")
		return nil
	},
}

// caSealCmd handles "sourcevault ca seal".
var caSealCmd = &cobra.Command{
	Use:   "seal",
	Short: "Clear the CA key from memory",
	Run: func(cmd *cobra.Command, args []string) {
		appSigner.Seal()
		fmt.Println("CA sealed.")
	},
}

// caStatusCmd handles "sourcevault ca status".
var caStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the CA is currently unsealed",
	Run: func(cmd *cobra.Command, args []string) {
		if appSigner.IsUnsealed() {
			fmt.Printf("Status: unsealed\nLoaded key: %s\n", appSigner.CAPath())
		} else {
			fmt.Println("Status: sealed")
		}
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

func init() {
	// ca create flags
	caCreateCmd.Flags().String("key-type", "", "Key algorithm: ed25519 or rsa (default from config)")
	caCreateCmd.Flags().Int("rsa-bits", 0, "RSA key size in bits (default from config, only used with --key-type=rsa)")
	caCreateCmd.Flags().Duration("valid-for", 0, "Certificate validity period e.g. 8760h (default from config)")
	caCreateCmd.Flags().String("name", "", "Human-readable label for this CA")

	// ca unseal flags
	caUnsealCmd.Flags().String("key", "", "Path to the CA private key file")

	// ca revoke flags
	caRevokeCmd.Flags().String("uuid", "", "UUID of the CA to revoke")

	// ca rotate flags
	caRotateCmd.Flags().String("revoke-uuid", "", "UUID of the CA being replaced")
	caRotateCmd.Flags().String("key-type", "", "Key algorithm for the new CA")
	caRotateCmd.Flags().Int("rsa-bits", 0, "RSA key size for the new CA")
	caRotateCmd.Flags().Duration("valid-for", 0, "Validity period for the new CA")
	caRotateCmd.Flags().String("name", "", "Name for the new CA")

	// Register all subcommands under caCmd.
	caCmd.AddCommand(caCreateCmd, caUnsealCmd, caSealCmd, caStatusCmd, caRevokeCmd, caRotateCmd)
}

// promptPassphrase reads a passphrase from the terminal without echoing it.
func promptPassphrase(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	pass, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	return pass, err
}
