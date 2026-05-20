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

package rpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/rpc"
	"os"
	"path/filepath"

	"sourcevault/internal/crypto"
)

// CAService provides RPC methods for managing the Certificate Authority.
type CAService struct {
	Signer *crypto.CASigner
}

// StatusArgs is the input for the Status method.
type StatusArgs struct{}

// StatusReply is the output for the Status method.
type StatusReply struct {
	IsUnsealed bool
	CAPath     string
}

// Status returns the current unseal status and path of the CA.
func (s *CAService) Status(args *StatusArgs, reply *StatusReply) error {
	reply.IsUnsealed = s.Signer.IsUnsealed()
	reply.CAPath = s.Signer.CAPath()
	return nil
}

// UnsealArgs is the input for the Unseal method.
type UnsealArgs struct {
	CAPath     string
	Passphrase []byte
}

// UnsealReply is the output for the Unseal method.
type UnsealReply struct {
	Success bool
}

// Unseal attempts to unseal the CA with the provided path and passphrase.
func (s *CAService) Unseal(args *UnsealArgs, reply *UnsealReply) error {
	if err := s.Signer.UnsealFromPath(args.CAPath, args.Passphrase); err != nil {
		return err
	}
	reply.Success = true
	return nil
}

// SealArgs is the input for the Seal method.
type SealArgs struct{}

// SealReply is the output for the Seal method.
type SealReply struct {
	Success bool
}

// Seal clears the in-memory private key material.
func (s *CAService) Seal(args *SealArgs, reply *SealReply) error {
	s.Signer.Seal()
	reply.Success = true
	return nil
}

// SignArgs is the input for the Sign method.
type SignArgs struct {
	PublicKey []byte
	Config    crypto.CertConfig
}

// SignReply is the output for the Sign method.
type SignReply struct {
	SignedCert []byte
	Serial     uint64
}

// Sign issues a signed SSH certificate.
func (s *CAService) Sign(args *SignArgs, reply *SignReply) error {
	cert, serial, err := s.Signer.Sign(args.PublicKey, args.Config)
	if err != nil {
		return err
	}
	reply.SignedCert = cert
	reply.Serial = serial
	return nil
}

// StartServer starts the RPC server on a Unix Domain Socket.
func StartServer(ctx context.Context, socketPath string, signer *crypto.CASigner) error {
	// Ensure the directory for the socket exists
	if err := os.MkdirAll(filepath.Dir(socketPath), 0700); err != nil {
		return fmt.Errorf("creating socket directory: %w", err)
	}

	// Remove existing socket if it exists
	_ = os.Remove(socketPath)

	service := &CAService{Signer: signer}
	if err := rpc.Register(service); err != nil {
		return fmt.Errorf("registering RPC service: %w", err)
	}

	l, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listening on unix socket %s: %w", socketPath, err)
	}

	slog.Info("RPC server started", "socket", socketPath)

	go func() {
		<-ctx.Done()
		slog.Info("Shutting down RPC server")
		l.Close()
		_ = os.Remove(socketPath)
	}()

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					slog.Debug("RPC accept error", "error", err)
					continue
				}
			}
			go rpc.ServeConn(conn)
		}
	}()

	return nil
}

// GetClient connects to the SourceVault RPC server.
func GetClient(socketPath string) (*rpc.Client, error) {
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("sourcevault server is not running (socket %s not found)", socketPath)
	}
	return rpc.Dial("unix", socketPath)
}
