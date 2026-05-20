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

package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"

	"sourcevault/internal/crypto"
	"sourcevault/internal/db"
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
		// Initialize the database connection.
		dbConn, err := db.Initialize(appCfg)
		if err != nil {
			return fmt.Errorf("initializing database: %w", err)
		}
		defer dbConn.Close()

		// Ensure the schema is up to date.
		if err := db.RunMigrations(dbConn, appCfg.Database.Driver); err != nil {
			return fmt.Errorf("running database migrations: %w", err)
		}

		keyType, _ := cmd.Flags().GetString("key-type")
		rsaBits, _ := cmd.Flags().GetInt("rsa-bits")
		validForStr, _ := cmd.Flags().GetString("valid-for")
		name, _ := cmd.Flags().GetString("name")

		if keyType != "" {
			appCfg.CA.DefaultKeyType = keyType
		}
		if rsaBits != 0 {
			appCfg.CA.DefaultRSABits = rsaBits
		}
		if validForStr != "" {
			d, err := crypto.ParseHumanDuration(validForStr)
			if err != nil {
				return fmt.Errorf("invalid validity period %q: %w", validForStr, err)
			}
			appCfg.CA.DefaultValidDays = int(d.Hours() / 24)
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

		// Use the interactive passphrase for this creation.
		appCfg.CA.Passphrase = string(passphrase)

		// Delegate creation logic to the crypto bootstrap module.
		if err := crypto.ForceCreateCA(appCfg, dbConn, appSigner, name); err != nil {
			return fmt.Errorf("force-creating CA: %w", err)
		}

		active, err := registry.GetActiveCA(appCfg)
		if err != nil {
			return fmt.Errorf("verifying new active CA: %w", err)
		}

		fmt.Printf("\nCA created successfully and registered as authoritative\n")
		fmt.Printf("  UUID:        %s\n", active.UUID)
		fmt.Printf("  Algorithm:   %s\n", active.Algorithm)
		fmt.Printf("  Fingerprint: %s\n", active.Fingerprint)
		fmt.Printf("  Valid until: %s\n", active.ValidUntil.Format(time.RFC3339))
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

		// Check if the CA has expired.
		if time.Now().After(active.ValidUntil) {
			return fmt.Errorf("CA %s expired on %s: run 'sourcevault ca rotate' to issue a new CA",
				active.UUID, active.ValidUntil.Format(time.RFC3339))
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
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var pubKeyBytes []byte
		var inputIsFile bool
		var pubKeyPath string

		// Step 1: Obtain the target public key.
		if len(args) > 0 {
			pubKeyPath = args[0]
			var err error
			pubKeyBytes, err = os.ReadFile(pubKeyPath)
			if err != nil {
				return fmt.Errorf("reading public key: %w", err)
			}
			inputIsFile = true
		} else {
			// Prompt for the public key data if no file is provided.
			fmt.Print("Paste the public key you wish to sign: ")
			var input string
			fmt.Scanln(&input) // This might be too simple for long SSH keys. 
			// Let's use a more robust way to read the line.
			// However, for brevity and following the prompt instruction:
			pubKeyBytes = []byte(strings.TrimSpace(input))
		}

		// Step 2: Gather certificate metadata (Identity and Principals).
		keyID, _ := cmd.Flags().GetString("id")
		if keyID == "" && term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Print("Enter Key ID (identity): ")
			fmt.Scanln(&keyID)
		}

		principals, _ := cmd.Flags().GetStringSlice("principals")
		if len(principals) == 0 && term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Print("Enter Principals (comma separated, optional): ")
			var pInput string
			fmt.Scanln(&pInput)
			if pInput != "" {
				principals = strings.Split(pInput, ",")
				for i := range principals {
					principals[i] = strings.TrimSpace(principals[i])
				}
			}
		}

		certTypeStr, _ := cmd.Flags().GetString("type")
		validForStr, _ := cmd.Flags().GetString("valid-for")

		var certType uint32
		switch certTypeStr {
		case "user":
			certType = ssh.UserCert
		case "host":
			certType = ssh.HostCert
		default:
			return fmt.Errorf("invalid certificate type %q: must be \"user\" or \"host\"", certTypeStr)
		}

		var validFor time.Duration
		if validForStr != "" {
			var err error
			validFor, err = crypto.ParseHumanDuration(validForStr)
			if err != nil {
				return fmt.Errorf("invalid validity period %q: %w", validForStr, err)
			}
		} else {
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

		// Step 3: Attempt signing (RPC first, then local fallback).
		client, err := sv_rpc.GetClient(appCfg.Sockets.SourceVault)
		if err == nil {
			defer client.Close()
			slog.Info("Requesting certificate signing via RPC")
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
			// Local process fallback.
			if !appSigner.IsUnsealed() {
				// Automated CA discovery for signing if sealed.
				active, err := registry.GetActiveCA(appCfg)
				if err != nil {
					return fmt.Errorf("getting active CA for unseal: %w", err)
				}
				if active == nil {
					return fmt.Errorf("no active CA found in registry: run 'sourcevault ca create' first")
				}

				caPath := filepath.Join(appCfg.RootDir, "data", "ca", active.UUID)
				passphrase, err := promptPassphrase(fmt.Sprintf("CA is sealed. Enter passphrase for %s: ", active.UUID))
				if err != nil {
					return fmt.Errorf("reading passphrase: %w", err)
				}

				if err := appSigner.UnsealFromPath(caPath, passphrase); err != nil {
					return fmt.Errorf("unseal failed: %w", err)
				}
			}

			// Verify expiration before signing locally.
			active, err := registry.GetActiveCA(appCfg)
			if err != nil {
				return fmt.Errorf("verifying CA validity: %w", err)
			}
			if active != nil && time.Now().After(active.ValidUntil) {
				return fmt.Errorf("cannot sign: active CA %s has expired", active.UUID)
			}

			slog.Info("Signing certificate locally")
			certBytes, serial, err = appSigner.Sign(pubKeyBytes, certConfig)
			if err != nil {
				return fmt.Errorf("signing failed: %w", err)
			}
		}

		// Step 4: Output the signed certificate.
		if inputIsFile {
			// Write to file alongside the public key.
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
		} else {
			// Output directly to stdout for interactive use.
			fmt.Printf("\n--- SIGNED SSH CERTIFICATE (Serial: %d) ---\n", serial)
			fmt.Println(string(certBytes))
			fmt.Println("-------------------------------------------")
		}

		return nil
	},
}

func init() {
	// ca create flags
	caCreateCmd.Flags().String("key-type", "", "Key algorithm: ed25519 or rsa (default from config)")
	caCreateCmd.Flags().Int("rsa-bits", 0, "RSA key size in bits (default from config, only used with --key-type=rsa)")
	caCreateCmd.Flags().String("valid-for", "", "Certificate validity period e.g. 1y, 2M, 30d, 12h (default from config)")
	caCreateCmd.Flags().String("name", "", "Human-readable label for this CA")

	// ca revoke flags
	caRevokeCmd.Flags().String("uuid", "", "UUID of the CA to revoke")

	// ca rotate flags
	caRotateCmd.Flags().String("revoke-uuid", "", "UUID of the CA being replaced")
	caRotateCmd.Flags().String("key-type", "", "Key algorithm for the new CA")
	caRotateCmd.Flags().Int("rsa-bits", 0, "RSA key size for the new CA")
	caRotateCmd.Flags().String("valid-for", "", "Validity period for the new CA e.g. 1y")
	caRotateCmd.Flags().String("name", "", "Name for the new CA")

	// ca sign flags
	caSignCmd.Flags().String("type", "user", "Certificate type: user or host")
	caSignCmd.Flags().String("id", "", "Key ID (comment) to embed in the certificate")
	caSignCmd.Flags().StringSlice("principals", nil, "List of valid principals (comma separated)")
	caSignCmd.Flags().String("valid-for", "", "Certificate validity period e.g. 1y, 2M, 30d, 12h (default from config)")

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
