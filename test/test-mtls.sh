#!/bin/sh

SERVER_IP=127.0.0.1
SERVER_NAME=server.lb.com
PORT=8888
DST=$SERVER_NAME:$PORT
CLIENT_CERT=client/client1.crt
CLIENT_KEY=client/client1.key.pem
CA_CERT=ca/ca.crt
CERT_PARAMS="--cert ${CLIENT_CERT} --key ${CLIENT_KEY} --cacert ca/ca.crt"

curl --resolve ${DST}:${SERVER_IP} --tlsv1.3 ${CERT_PARAMS} https://${DST}/ 
