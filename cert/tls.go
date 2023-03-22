package cert

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"log"
)

//go:embed server.crt
var rsaCert []byte

//go:embed  server.key
var PublicKey []byte

var Tlsconfig *tls.Config

func init() {
	//内置证书

	cert, err := tls.X509KeyPair(rsaCert, PublicKey)
	if err != nil {
		log.Panicln(err)
		return

	}
	certPool := x509.NewCertPool()

	if ok := certPool.AppendCertsFromPEM(rsaCert); !ok {
		log.Panicln("PEM err")
		return
	}
	Tlsconfig = &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
		RootCAs:            certPool,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		ClientCAs:          certPool,
		MaxVersion:         tls.VersionTLS12,
		MinVersion:         tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		},
	}

}
