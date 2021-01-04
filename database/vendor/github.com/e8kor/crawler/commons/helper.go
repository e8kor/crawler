package commons

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
)

// StreamToByte convert stream of bytes to byte array
func StreamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(stream)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// GetAPISecret is helper to read secrets in openfaas
func GetAPISecret(secretName string) (secret string) {
	secretBytes, err := ioutil.ReadFile("/var/openfaas/secrets/" + secretName)
	if err != nil {
		panic(err)
	}
	secret = string(secretBytes)
	return secret
}

//RandomFilename is shortcut to generate filename string
func RandomFilename() (s string, err error) {
	b := make([]byte, 8)
	_, err = rand.Read(b)
	if err != nil {
		return
	}
	s = fmt.Sprintf("%x", b)
	return
}
