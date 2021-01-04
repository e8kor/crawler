package commons

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
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

// HandleFailure will respond with failure for client
func HandleFailure(w http.ResponseWriter, err error) {
	log.Println("error when processing request", err)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(fmt.Sprintln("error when processing request", err)))
}

// HandleSuccess will respond with responses for client
func HandleSuccess(w http.ResponseWriter, v interface{}) {
	raw, err := json.Marshal(v)
	if err != nil {
		HandleFailure(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(raw)
}
