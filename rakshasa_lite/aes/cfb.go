package aes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
)

func AesCfbNewEncrypSteam() cipher.Stream {
	block, _ := aes.NewCipher(Key)
	iv := bytes.Repeat([]byte("1"), block.BlockSize())

	return cipher.NewCFBEncrypter(block, iv)
}
func AesCfbNewDecrypSteam() cipher.Stream {
	block, _ := aes.NewCipher(Key)
	iv := bytes.Repeat([]byte("1"), block.BlockSize())

	return cipher.NewCFBDecrypter(block, iv)
}
