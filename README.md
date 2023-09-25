# Least Connection Load Balancer

## Features
Ths is a simple TCP load balancer that distributes incoming requests across multiple servers.
- it uses least connection algorithm to route requests evenly accross upstream servers
- it limits the rate of connections each client can do
- it provides mTLS communication
- it has an authorization scheme to authenticate clients

## Design
The design is available [here](design/design.md)