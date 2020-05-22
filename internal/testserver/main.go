package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func main() {
	// port := "8443"
	// port := "80"
	port := "443"

	// selfSignedCert := "./cert/public.crt"
	// selfSignedPrivKey := "./cert/private.key"
	// selfSignedHost := "localhost"

	// server := &http.Server{
	// 	Addr:         ":" + port,
	// 	ReadTimeout:  5 * time.Second,
	// 	WriteTimeout: 10 * time.Second,
	// 	TLSConfig:    tlsConfig(selfSignedCert, selfSignedPrivKey, selfSignedHost),
	// }

	// Server cert setup
	caSignedCert := "/Users/rich_youngkin/certs/fullchain.pem"
	caSignedPrivKey := "/Users/rich_youngkin/certs/privkey.pem"
	caSignedHost := "elev5280.com"

	// Accepted clients cert setup
	clientSignedCert := "/Users/rich_youngkin/certs/cert.pem"

	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 10 * time.Second,
		TLSConfig:    tlsConfig(caSignedCert, caSignedPrivKey, clientSignedCert, caSignedHost),
	}

	//// Having this does not change anything but just showing.
	//// go get -u golang.org/x/net/http2
	//if err := http2.ConfigureServer(server, nil); err != nil {
	//	log.Fatal(err)
	//}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s request for host %s from IP address %s and X-FORWARDED-FOR %s",
			r.Method, r.Host, r.RemoteAddr, r.Header.Get("X-FORWARDED-FOR"))
		w.Write([]byte(fmt.Sprintf("Protocol: %s", r.Proto)))
		log.Printf("Sent response %s", r.Proto)
	})

	// log.Printf("Starting TLS server on host %s and port %s", selfSignedHost, port)
	log.Printf("Starting TLS server on host %s and port %s", caSignedHost, port)

	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatal(err)
	}
}

func tlsConfig(certFileName, keyFileName, clientFileName, host string) *tls.Config {
	crt, err := ioutil.ReadFile(certFileName)
	if err != nil {
		log.Fatal(err)
	}

	key, err := ioutil.ReadFile(keyFileName)
	if err != nil {
		log.Fatal(err)
	}

	cert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		log.Fatal(err)
	}

	clientCert, err := ioutil.ReadFile(clientFileName)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(clientCert)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   host,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
	}
}
