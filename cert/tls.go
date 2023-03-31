package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/pem"
	"github.com/farmerx/gorsa"
	"log"
)

//go:embed server.crt
var rsaCert []byte

//go:embed  server.key
var PrivateKey []byte

var Tlsconfig *tls.Config

func init() {
	//内置证书

	cert, err := tls.X509KeyPair(rsaCert, PrivateKey)
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

//go:embed  public.pem
var publicKey []byte


var privateKey []byte

// 使用公钥进行加密
func RSAEncrypter(msg []byte) []byte {
	block, _ := pem.Decode(publicKey)
	pub, _ := x509.ParsePKIXPublicKey(block.Bytes)
	cipherText, _ := rsa.EncryptPKCS1v15(rand.Reader, pub.(*rsa.PublicKey), msg)
	return cipherText
}
func RSAEncrypterStr(msg string) string {
	block, _ := pem.Decode(publicKey)
	pub, _ := x509.ParsePKIXPublicKey(block.Bytes)
	cipherText, _ := rsa.EncryptPKCS1v15(rand.Reader, pub.(*rsa.PublicKey), []byte(msg))
	return base64.StdEncoding.EncodeToString(cipherText)
}

// 使用私钥进行解密
func RSADecrypter(cipherText []byte) []byte {
	if block, _ := pem.Decode(privateKey); block != nil {
		p, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
		afterDecrypter, _ := rsa.DecryptPKCS1v15(rand.Reader, p, cipherText)
		return afterDecrypter
	}
	return []byte{}
}
func RSADecrypterStr(cipherText string) string {
	if block, _ := pem.Decode(privateKey); block != nil {
		p, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
		b, _ := base64.StdEncoding.DecodeString(cipherText)
		afterDecrypter, _ := rsa.DecryptPKCS1v15(rand.Reader, p, b)
		return string(afterDecrypter)
	}
	return ""
}
func RSAEncrypterByPriv(msg string) string {
	prienctypt, _ := gorsa.RSA.PriKeyENCTYPT([]byte(msg))
	return base64.StdEncoding.EncodeToString(prienctypt)
}
func RSAEncrypterByPrivByte(msg []byte) []byte {
	prienctypt, _ := gorsa.RSA.PriKeyENCTYPT([]byte(msg))
	return prienctypt
}
func RSADecrypterByPub(cipherText string) string{
	b, _ := base64.StdEncoding.DecodeString(cipherText)
	pubdecrypt, _ := gorsa.RSA.PubKeyDECRYPT(b)
	return string(pubdecrypt)
}
func RSADecrypterByPubByte(cipherText []byte) []byte {
	pubdecrypt, _ := gorsa.RSA.PubKeyDECRYPT(cipherText)
	return pubdecrypt
}
func init(){
	gorsa.RSA.SetPublicKey(string(publicKey))
	gorsa.RSA.SetPrivateKey(string(privateKey))
}
