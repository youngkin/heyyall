// Copyright (c) 2020 Richard Youngkin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

// This is a test https server. It can be used to test heyyall's client authentication capability.
//
// This code was influenced by an example project in GitHub - https://github.com/jcbsmpsn/golang-https-example

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	help := flag.Bool("help", false, "Optional, prints usage info")
	host := flag.String("host", "", "Required flag, must be the hostname that is resolvable via DNS")
	port := flag.String("port", "443", "The https port, defaults to 443")
	serverPEM := flag.String("srvpem", "", "Required, the name of the server's PEM file")
	clientPEM := flag.String("clientpem", "", "Required, the name of the to be authenticated client's PEM file")
	privKey := flag.String("key", "", "Required, the file name of the server's private key file")
	flag.Parse()

	usage := `usage:
	
testserver -host <hostname> -srvpem <serverPEMFile> -clientpem <clientPEMFile> -key <serverPrivateKeyFile> [-port <port> -help]
	
Options:
  -help       Prints this message
  -host       Required, a DNS resolvable host name
  -srvpem     Required, the name the server's PEM file
  -clientpem  Required, the name the to be authenticated client's PEM file
  -key        Required, the name the server's key PEM file
  -port       Optional, the https port for the server to listen on`

	if *help == true {
		fmt.Println(usage)
		return
	}
	if *host == "" || *serverPEM == "" || *clientPEM == "" || *privKey == "" {
		fmt.Printf("One or more required fields missing:\n%s", usage)
		os.Exit(1)
	}

	server := &http.Server{
		Addr: ":" + *port,
		// 5 min to allow for delays when 'curl' on OSx prompts for username/password
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 10 * time.Second,
		TLSConfig:    tlsConfig(*host, *clientPEM),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s request for host %s from IP address %s and X-FORWARDED-FOR %s",
			r.Method, r.Host, r.RemoteAddr, r.Header.Get("X-FORWARDED-FOR"))
		w.Write([]byte(fmt.Sprintf("Protocol: %s", r.Proto)))
		log.Printf("Sent response %s", r.Proto)
	})

	log.Printf("Starting TLS server on host %s and port %s", *host, *port)
	if err := server.ListenAndServeTLS(*serverPEM, *privKey); err != nil {
		log.Fatal(err)
	}
}

// func tlsConfig(serverPEMFile, serverKeyFile, host, clientPEMFile string) *tls.Config {
func tlsConfig(host, clientPEMFile string) *tls.Config {
	clientPEM, err := ioutil.ReadFile(clientPEMFile)
	if err != nil {
		log.Fatal("Error opening cert file", clientPEMFile, ", error ", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(clientPEM)

	return &tls.Config{
		// Certificates: []tls.Certificate{cert},
		ServerName: host,
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  caCertPool,
	}
}
