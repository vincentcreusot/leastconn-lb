# Least Connection Load Balancer

## Features
Ths is a simple TCP load balancer that distributes incoming requests across multiple servers.
- it uses least connection algorithm to route requests evenly accross upstream servers
- it limits the rate of connections each client can do
- it provides mTLS communication
- it has an authorization scheme to authenticate clients

## Design
The design is available [here](design/design.md)

## Balancer library
### API
The balancer library exposes the following API:
```go
// NewLoadBalancer initializes a balancer with the given upstreams
func NewLoadBalancer(upstreams []string) *Balancer
// Balance distributes a connection to an upstream server
func (b *Balancer) Balance(conn *net.TCPConn, clientId string, allowedUpstreams []string, errorsChan chan []error) error
```
The `Balance` function takes these parameters:
- `conn` is the incoming TCP connection from the client
- `clientId` is a unique ID for the client (should be the CN in the certificate)
- `allowedUpstreams` limits which upstreams can be selected (used by auth scheme)
- `errorsChan` is used to report any errors during forwarding

### Implementation
The balancer sequentiallly applies the following:
- calls the rate limiter and if the client has exceeded the rate limit, return an error and close the connection
- pick a least used upstream server from the allowed upstreams
- forward the connection to the selected upstream using a connection forwarder

### Testing
The `test` folder contains a `docker-compose.yml` file to spin up dummy servers for testing.

A bash script is provided to call parallelly a big number of requests.