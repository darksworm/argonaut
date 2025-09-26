package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

func main() {
	// Create a new CA certificate and key pair for testing
	caCert, caKey, caPEM, err := createTestCA()
	if err != nil {
		fmt.Printf("Failed to create test CA: %v\n", err)
		os.Exit(1)
	}

	// Write CA certificate
	if err := os.WriteFile("ca.pem", caPEM, 0644); err != nil {
		fmt.Printf("Failed to write CA certificate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated ca.pem")

	// Generate server certificate signed by CA
	serverCert, serverKey, err := createServerCert(caCert, caKey, []string{"localhost", "127.0.0.1"})
	if err != nil {
		fmt.Printf("Failed to create server certificate: %v\n", err)
		os.Exit(1)
	}

	// Write server certificate
	if err := os.WriteFile("server.crt", serverCert, 0644); err != nil {
		fmt.Printf("Failed to write server certificate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated server.crt")

	// Write server private key
	if err := os.WriteFile("server.key", serverKey, 0600); err != nil {
		fmt.Printf("Failed to write server key: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated server.key")

	// Generate client certificate and key
	clientCert, clientKey, err := createClientCert(caCert, caKey)
	if err != nil {
		fmt.Printf("Failed to create client certificate: %v\n", err)
		os.Exit(1)
	}

	// Write client certificate
	if err := os.WriteFile("client.crt", clientCert, 0644); err != nil {
		fmt.Printf("Failed to write client certificate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated client.crt")

	// Write client private key
	if err := os.WriteFile("client.key", clientKey, 0600); err != nil {
		fmt.Printf("Failed to write client key: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated client.key")

	fmt.Println("All certificates generated successfully!")
}

func createClientCert(caCert *x509.Certificate, caKey *rsa.PrivateKey) ([]byte, []byte, error) {
	// Generate client private key
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Create client certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Organization:  []string{"Argonaut E2E Test Client"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    "test-client",
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Create certificate signed by CA
	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}

	// Encode certificate as PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Encode private key as PEM
	keyDER, err := x509.MarshalPKCS8PrivateKey(clientKey)
	if err != nil {
		return nil, nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM, nil
}

// createTestCA creates a test CA certificate
func createTestCA() (*x509.Certificate, *rsa.PrivateKey, []byte, error) {
	// Generate private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Argonaut E2E Test CA"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, nil, err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, nil, err
	}

	// Encode as PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	return cert, priv, certPEM, nil
}

// createServerCert creates a server certificate signed by the given CA
func createServerCert(caCert *x509.Certificate, caKey *rsa.PrivateKey, hosts []string) ([]byte, []byte, error) {
	// Generate private key for server
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization:  []string{"Argonaut E2E Test Server"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     hosts,
	}

	// Add IP addresses
	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}

	// Create certificate signed by CA
	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}

	// Encode certificate as PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Encode private key as PEM
	keyDER, err := x509.MarshalPKCS8PrivateKey(serverKey)
	if err != nil {
		return nil, nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM, nil
}