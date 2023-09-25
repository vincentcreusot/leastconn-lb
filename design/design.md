# Design document

As requested, the project is broken into 2 components
- the load balancer library
- a server that exposes the load balancer functionality and provides secure communication

The load balancer is written in Go.

Here is a simplified high level diagram (the diagram is generated and written in [d2lang](https://d2lang.com/) in this folder): !["HighLevel"](layers.svg)

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

## Server

### Secured communication
#### Authentication
The server will use mutual TLS for authentication. Clients will need to provide a valid client certificate during the TLS handshake.
##### Certificate configuration

#### TLS versions
We will support TLS 1.2 and 1.3 for the communication between clients and server. It removes a lot of unsecure or deprecated TLS features and provides better security. I found out that an estimate of 20% of corporate internet traffic is using TLS 1.3 in 2023 and adding TLS 1.2 will cover the most secure enough clients possible. Some adaptations need to be made to stay secure:

#### Cipher suites
The following cipher suites can be used:
- ECDHE-ECDSA-AES128-GCM-SHA256
- ECDHE-ECDSA-CHACHA20-POLY1305
- ECDHE-RSA-AES128-GCM-SHA256
- ECDHE-RSA-CHACHA20-POLY1305
- ECDHE-ECDSA-AES256-GCM-SHA384
- ECDHE-RSA-AES256-GCM-SHA384

#### Configuration

### Authorization scheme
#### Principle


### User experience
