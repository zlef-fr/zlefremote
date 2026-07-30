package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// LAN mode has to be HTTPS, and here is why: a browser only exposes
// `crypto.subtle` — i.e. the AES-256-GCM this whole protocol is built on — in a
// SECURE CONTEXT. `http://192.168.x.y:9783` is not one (only localhost gets a
// pass), so over plain HTTP the client silently has no crypto and can't pair.
//
// There is no certificate authority for "the laptop on your Wi-Fi", so the
// agent mints its own: a self-signed certificate covering this machine's
// addresses, cached next to the identity file. The browser warns once, the user
// proceeds, and from then on the origin is secure — WebCrypto works and the
// WebSocket upgrades to wss on the same exception.
//
// The certificate is NOT what protects the session: every frame is sealed with
// the key from the URL fragment, which never leaves the two endpoints. TLS here
// only buys the secure context (and stops a passive sniffer from seeing the
// static assets).

const lanCertValidity = 5 * 365 * 24 * time.Hour

func lanCertPaths() (certPath, keyPath string, err error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", "", err
	}
	d := filepath.Join(dir, "zlefremote")
	if err := os.MkdirAll(d, 0o700); err != nil {
		return "", "", err
	}
	return filepath.Join(d, "lan-cert.pem"), filepath.Join(d, "lan-key.pem"), nil
}

// lanCertificate returns the cached certificate, regenerating it when it is
// missing, expiring, or no longer covers the address we are about to serve on
// (laptops move between networks and get a new IP).
func lanCertificate(host string) (tls.Certificate, error) {
	certPath, keyPath, err := lanCertPaths()
	if err != nil {
		return tls.Certificate{}, err
	}
	if c, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil {
		if leaf, perr := x509.ParseCertificate(c.Certificate[0]); perr == nil {
			if time.Now().Before(leaf.NotAfter.Add(-24*time.Hour)) && coversHost(leaf, host) {
				c.Leaf = leaf
				return c, nil
			}
		}
	}
	return newLANCertificate(certPath, keyPath, host)
}

func coversHost(leaf *x509.Certificate, host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		for _, known := range leaf.IPAddresses {
			if known.Equal(ip) {
				return true
			}
		}
		return false
	}
	return leaf.VerifyHostname(host) == nil
}

func newLANCertificate(certPath, keyPath, host string) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}
	name, _ := os.Hostname()
	if name == "" {
		name = "zlefremote"
	}

	tmpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: name + " (ZlefRemote agent)", Organization: []string{"ZlefRemote"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(lanCertValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              dnsNames(name),
		IPAddresses:           localIPs(host),
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	// best effort: a cert we can't cache is still usable for this run
	_ = os.WriteFile(certPath, certPEM, 0o600)
	_ = os.WriteFile(keyPath, keyPEM, 0o600)

	return tls.X509KeyPair(certPEM, keyPEM)
}

func dnsNames(hostname string) []string {
	names := []string{"localhost"}
	if hostname != "" {
		names = append(names, hostname, hostname+".local")
	}
	return names
}

// localIPs lists every address this machine answers on, so the same certificate
// keeps working whether the phone was handed the Wi-Fi address, a second
// interface, or localhost.
func localIPs(host string) []net.IP {
	ips := []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
	if ip := net.ParseIP(host); ip != nil {
		ips = append(ips, ip)
	}
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok && !ipn.IP.IsLoopback() {
			ips = append(ips, ipn.IP)
		}
	}
	return ips
}
