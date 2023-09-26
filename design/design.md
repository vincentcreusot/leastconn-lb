# Design document

As requested, the project is broken into 2 components
- the TCP load balancer library
- a server that exposes the load balancer functionality and provides secure communication

The load balancer is written in Go.

Here is a simplified high level diagram (the diagram is generated and written in [d2lang](https://d2lang.com/) in this folder): <img src="layers.svg" width="300"/>

## Load balancer library
The load balancer library will contain the core logic for distributing incoming requests across backend servers. It will:
- provide methods to forward a TCP connection to the upstream servers of the load balancer
- use a least connection algorithm which means we will keep track of the usage of the upstream servers
- rate limit the connections based on the client usage

### Connection forwarder
The forwarder maintains a list of connections opened towards the upstream servers. It will forward incoming TCP connections to the least used upstream server based on the least connection algorithm.

The map of upstreams containing the connections count uses a mutex to make it thread safe.

After finding the least used upstream server, it increments the count and opens a new connection towards that server. It copies the data from the client connection to the upstream connection and vice versa using `io.Copy`.

Copying the data in a goroutine ensures non blocking behavior.

### Rate limiter
The rate limiter discards connection if the client makes too many connections for an amount of time.
I will use the `token bucket` algorithm implemented in package `x/time/rate`. It permits to define a burst of connections and will have a max connections per second refill of the "bucket". 

To demonstrate that feature easily I will use a bucket size of `3` and a rate of `1`.

## Server
The server uses the forwarder library and appends security.
### Secured communication
#### Authentication
The server uses mutual TLS for authentication. Clients will need to provide a valid client certificate during the TLS handshake.

In Mutual TLS, the server requests the client to provide a digital certificate which contains the client's public key and identity information. The server verifies the client's certificate by checking its authenticity and ensuring it is signed by a trusted Certificate Authority (CA) that the server recognizes. This confirms the client's identity.
##### Certificate configuration
To generate the certificates we must use the same Certificate Authority.
```
# Generate CA cert
openssl genrsa -aes256 -out ca/ca.key 4096 
openssl req -new -x509 -sha256 -days 20 -key ca/ca.key -out ca/ca.crt

# Generate CSR for server
openssl genrsa -out server/localhost.key 2048
openssl req -new -key server/localhost.key -sha256 -out server/localhost.csr
# Validate CSR
openssl x509 -req -days 365 -sha256 -in server/localhost.csr -CA ca/ca.crt -CAkey ca/ca.key -set_serial 1 -out server/localhost.crt

# Create client CRT
openssl genrsa -out client/client1.key 2048
openssl req -new -key client/client1.key -out client/client1.csr
# Validate CSR
openssl x509 -req -days 365 -sha256 -in client/client1.csr -CA ca/ca.crt -CAkey ca/ca.key -set_serial 2 -out client/client1.crt
```

#### TLS versions
We will support TLS 1.2 and 1.3 for the communication between clients and server. It removes a lot of unsecure or deprecated TLS features and provides better security. I found out that an estimate of 20% of corporate internet traffic is using TLS 1.3 in 2023 and adding TLS 1.2 will cover the most secure enough clients possible. 

#### Cipher suites
A cipher suite consists of a key exchange algorithm, an authentication algorithm, a bulk encryption algorithm, and a message authentication algorithm.

The following cipher suites can be used from must secure to less secure: 
1. ECDHE-ECDSA-AES256-GCM-SHA384
2. ECDHE-RSA-AES256-GCM-SHA384
3. ECDHE-ECDSA-CHACHA20-POLY1305
4. ECDHE-RSA-CHACHA20-POLY1305
5. ECDHE-ECDSA-AES128-GCM-SHA256
6. ECDHE-RSA-AES128-GCM-SHA256


#### Configuration
We can use [Mozilla SSL Config Generator](https://ssl-config.mozilla.org/#server=go&version=1.21&config=intermediate&guideline=5.7) with the intermediate configuration and extract the Go TLS configuration as follows
```
cfg := &tls.Config{
    MinVersion: tls.VersionTLS12,
    CipherSuites: []uint16{
        tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
        tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
    },
}
```
### Authorization scheme
#### Principle
A map is created in the server with the upstreams associated to each client. This map is passed to the forward method who will verify if the upstream selected is in the list passed for the client.

The way to determine the client is the subject.CommonName field of the certificate. The certificate needs to have that information correctly filled so that we can identify it and authorize it.

### User experience
We can simulate the user experience by using `curl` in the background and use the different certificates alternatively.
```sh
curl --cacert file.crt &
```
I will maintain a shell script that launches them parallely.

The server side can be simulated with several `nginx` servers running in `docker`. Reading the access logs of the servers will give us an idea of the load balancing accuracy.

## Trade Offs
Here is a list of trade offs/assumptions used for the design of the solution: 

- for the simplicity of the exercise, I will not manage upstream servers failures, the server will be available and the least connections algorithm will not include the health of the upstream.
- to have a better security, I will discard SSL and TLS before 1.2
- for the sake of the exercise, the list of upstreams is hard coded
- for the sake of the exercise, the rate limit is hard coded
- the number of clients is considered small enough to not have to clean the clients not using the load balancer for a long period 