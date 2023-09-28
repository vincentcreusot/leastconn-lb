#!/bin/sh
mkdir -p ca server client
# Generate CA cert
openssl genrsa -aes256 -out ca/ca.key 4096 
openssl req -new -x509 -sha256 -days 20 -key ca/ca.key -out ca/ca.crt 

# Generate CSR for server
openssl genrsa -aes256 -out server/localhost.key 4096
openssl req -new -key server/localhost.key -sha256 -out server/localhost.csr
# Validate CSR
openssl x509 -req -days 20 -sha256 -in server/localhost.csr -CA ca/ca.crt -CAkey ca/ca.key -set_serial 1 -out server/localhost.crt

# Create client CRT
openssl genrsa -aes256 -out client/client1.key 4096
openssl req -new -key client/client1.key -out client/client1.csr
# Validate CSR
openssl x509 -req -days 20 -sha256 -in client/client1.csr -CA ca/ca.crt -CAkey ca/ca.key -set_serial 2 -out client/client1.crt