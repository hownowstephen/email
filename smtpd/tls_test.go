package smtpd_test

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/tls"
    "crypto/x509"
    "crypto/x509/pkix"
    "math/big"
    "net"
    "sync"
)

var tlsGen sync.Once
var tlsConfig *tls.Config

func TestingTLSConfig() *tls.Config {

    tlsGen.Do(func() {

        priv, err := rsa.GenerateKey(rand.Reader, 2048)
        if err != nil {
            panic(err)
        }
        serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
        serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
        if err != nil {
            panic(err)
        }
        xc := x509.Certificate{
            SerialNumber: serialNumber,
            Subject: pkix.Name{
                Organization: []string{"Acme Co"},
            },
            IsCA:                  true,
            KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
            ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
            BasicConstraintsValid: true,
            IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
        }

        b, err := x509.CreateCertificate(rand.Reader, &xc, &xc, &priv.PublicKey, priv)
        if err != nil {
            panic(err)
        }

        tlsConfig = &tls.Config{
            Certificates: []tls.Certificate{
                tls.Certificate{
                    Certificate: [][]byte{b},
                    PrivateKey:  priv,
                    Leaf:        &xc,
                },
            },
            ClientAuth: tls.VerifyClientCertIfGiven,
            Rand:       rand.Reader,
        }
    })

    return tlsConfig
}
