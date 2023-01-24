// Package spiffecert contains cert reload and spiffe items
// Copyright (c) 2016-2021, F5 Networks, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package spiffecert

import (
	"fmt"
	"os"
	"path"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

const (
	bundleFileMode = os.FileMode(0o644)
	certsFileMode  = os.FileMode(0o644)
	keyFileMode    = os.FileMode(0o600)
)

//go:generate counterfeiter -generate

// DiskSVIDConfig contains the configuration for a Writer.
type DiskSVIDConfig struct {
	// CertDir is the directory that holds the certificates and key.
	CertDir,
	// KeyFilename is the name of the private key file.
	KeyFilename,
	// CertFilename is the name of the certificate file
	CertFilename,
	// CABundleFilename is the name of the CA certificate file.
	CABundleFilename string
}

// SVIDWriter knows how extract and write certificates and keys from a SPIFFE X509-SVID.
//
//counterfeiter:generate ./. SVIDWriter
type SVIDWriter interface {
	// Write writes a private key, certificate, and CA certificate from a SPIFFE X509-SVID
	Write(svidResponse *workloadapi.X509Context) error
}

// DiskSVIDWriter implements SVIDWriter interface.
type DiskSVIDWriter struct {
	KeyFile,
	CertFile,
	CaBundleFile string
}

// NewDiskSVIDWriter creates a new instance of Writer.
// Returns an error if the cert directory does not exist.
func NewDiskSVIDWriter(config DiskSVIDConfig) (*DiskSVIDWriter, error) {
	if _, err := os.Stat(config.CertDir); err != nil && os.IsNotExist(err) {
		return nil, err
	}
	writer := &DiskSVIDWriter{
		KeyFile:      path.Join(config.CertDir, config.KeyFilename),
		CertFile:     path.Join(config.CertDir, config.CertFilename),
		CaBundleFile: path.Join(config.CertDir, config.CABundleFilename),
	}

	return writer, nil
}

// Write parses the svidResponse into a private key, certificate, and CA.
// The key, cert, and CA cert are written to disk.
func (d *DiskSVIDWriter) Write(svidResponse *workloadapi.X509Context) error {
	svid := svidResponse.DefaultSVID()
	caBundle, err := ParseCABundle(svidResponse)
	if err != nil {
		return err
	}

	// Convert to PEM format
	pemBundle, err := caBundle.Marshal()
	if err != nil {
		return fmt.Errorf("unable to marshal X.509 SVID Bundle: %w", err)
	}
	pemCerts, pemKey, err := svid.Marshal()
	if err != nil {
		return fmt.Errorf("unable to marshal X.509 SVID: %w", err)
	}

	// Write out the files
	if err := os.WriteFile(d.CertFile, pemCerts, certsFileMode); err != nil {
		return fmt.Errorf("error writing certificate: %w", err)
	}
	if err := os.WriteFile(d.CaBundleFile, pemBundle, bundleFileMode); err != nil {
		return fmt.Errorf("error writing CA certificate: %w", err)
	}
	if err := os.WriteFile(d.KeyFile, pemKey, keyFileMode); err != nil {
		return fmt.Errorf("error writing private key: %w", err)
	}

	return nil
}

// ParseCABundle converts an X509Context into a native go bundle.
func ParseCABundle(svidResponse *workloadapi.X509Context) (*x509bundle.Bundle, error) {
	trustDomain := svidResponse.DefaultSVID().ID.TrustDomain()
	bundle, err := svidResponse.Bundles.GetX509BundleForTrustDomain(trustDomain)
	if err != nil {
		return nil, fmt.Errorf("error parsing CA bundle from svid response: %w", err)
	}

	return bundle, nil
}
