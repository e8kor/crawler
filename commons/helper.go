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
	"net/url"
	"os"
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

//CallFunction calls underlyting function
func CallFunction(functionName string, params url.Values, body interface{}, data interface{}) (err error) {
	var (
		gatewayPrefix = os.Getenv("GATEWAY_URL")
	)
	raw, err := json.Marshal(body)
	if err != nil {
		log.Println("error while marshalling data for", err)
		return
	}
	response, err := http.Post(gatewayPrefix+functionName+"?"+params.Encode(), "application/json", bytes.NewBuffer(raw))
	if err != nil {
		log.Println("error when sending", functionName, "request", err)
		return
	}
	err = json.NewDecoder(response.Body).Decode(&data)
	return nil
}

//FireFunction calls underlyting function
func FireFunction(functionName string, params url.Values, body interface{}) (err error) {
	var (
		gatewayPrefix = os.Getenv("GATEWAY_URL")
	)
	raw, err := json.Marshal(body)
	if err != nil {
		log.Println("error while marshalling data for", err)
		return
	}
	response, err := http.Post(gatewayPrefix+functionName+"?"+params.Encode(), "application/json", bytes.NewBuffer(raw))
	if err != nil {
		log.Println("error when sending", functionName, "function request", err)
		return
	}
	log.Println("response from", functionName, "function", response)
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
