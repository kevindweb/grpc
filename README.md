# grpc
Language-agnostic RPC load balancer and API to handle distributed communication challenges


This is an implementation of the [gRPC Library by Google](https://github.com/grpc/grpc). gRPC is an extension of the RPC protocol, that enables environment-agnostic control over a communication system. It achieves this by allowing backend communication between multiple programming languages, which is especially helpful for distributed systems such as microservices, where multiple different environments need to be able to effectively communicate. 

RPC is a protocol that executes processes synchronously on a specific server. This is widely used in distributed systems as it can provide abstract access to functionality that only master servers have control over for example. A big problem is that with microservice architectures, individual workers should not be limited by programming language. gRPC comes in and fixes this by providing libraries in many different languages that all can call different servers regardless of the client environment. gRPC also provides performance benefits for other kinds of distributed networks, like IoT devices. [This](https://hal.archives-ouvertes.fr/hal-02495252/document) paper discusses the benefits of offloading computation of embedded IoT devices by using the gRPC framework. 

Our implementation would focus on the Go and Python implementations for testing cross-language functionality and cover a multitude of challenges such as pluggable support for load balancing, tracing, health checking and authentication. 

---
# Running the protocol
The program requires at least 3 terminal sessions open:
1. Load Balancer
2. Server
3. Client

In that order will instatiate a RPC connection. A `tmux` session is the easist way to spin up new terminals. To run the go version of everything, `cd` into the directories (load_balancer for example) and run `go build && ./grpc`. The golang client needs to be built from the `grpc` folder with the `main.go` code. 

The python implementation for example can be run with `python server.py` and `python client.py`. 

## Memory-intensive functions
To see the true benefit of gRPC, run with `go build && ./grpc perfTest`. This will run an extremely IO-bound function to test the functionality of the golang server.

# Challenges 
#### Health Checks & Fault Tolerance
When a client sends a request through our system, it expects to get a timely response from any arbitrary server. Like many systems, the gRPC library handles client deadlines with timeouts. If for any reason a server cannot handle the request in a specific amount of time, we will do one of two things (depending on the system parameters specified by the client).
1. Retry with a secondary server that can hopefully successfully complete the request. This piece assumes that we both have multiple servers registered, and they are healthy enough to take requests. If there are no more servers, default to option 2 below.
2. Do not retry, and return with a failed status code that the client program can use.

The gRPC protocol outlines their use of [deadlines](https://grpc.io/docs/what-is-grpc/core-concepts/#deadlines) to acknowledge the issue of fault tolerance during a program. 

Similarly, research has been done on ways to handle errors in gRPC, like [this](https://www.usenix.org/conference/srecon19asia/presentation/sheerin) talk given at a conference about the challenges of safely handling errors in gRPC.

#### Authentication:
Our implementation will allow for both encrypted and unencrypted traffic, as requested by the user. Each of these can be set with a flag upon the startup of the server on a specific port. Where gRPC has many more advanced authentication features, such as Google credentials, we will limit the parameters to handling a cert file supplied to the program from the filesystem. The traffic will use TLS/SSL encryption, which are verified cryptographic protocols designed to provide communication security over large networks. Metadata through the communication layer is a good way of implementing security as well, and can let the user handle their own security if they want. We will supply an API to send key-value pairs, in addition to the request, letting users hook up their own protocols as they please. In distributed systems, you cannot assume that all your nodes are friendly, as security professionals try to break systems everyday. With encryption, you can be sure that there will be no man-in-the-middle attacks or listeners when transporting important data.

#### Tracing Support
Distributed tracing is a reliable way to find critical errors and bottlenecks quickly in a large system. The [opentracing API](https://github.com/opentracing) is a very well known open source library that provides support for Golang and Python. Since these are our target environments, we will be enhancing our gRPC protocol with support for tracing. Users will be able to supply parameters that tell the program when to set checkpoints. By building tracing into the platform, users will be confident in their ability to use the implementation in their system and develop applications quickly.

#### Load Balancing (two potential solutions):
There are two potential implementations of load balancing in gRPC.
[Proxy Load Balancing](https://grpc.io/blog/grpc-load-balancing/#proxy-load-balancer-options), which is done via a load balancer at the L3/4 (transport level) or the L7 (application level). The advantages of proxy load balancing are that the client implementation can be simple as it has no awareness of the backend servers and that the client can be untrusted. The downside is that there’s a performance bottleneck, the load balancer, which can lead to higher latency. 

[Client Load Balancing](https://grpc.io/blog/grpc-load-balancing/#client-side-lb-options) is the other implementation. This implementation is more complex than the simplest implementation of proxy load balancing, but there’s no performance bottleneck as there’s no load balancer. However, this implementation is based on the assumption that the client can be trusted and requires a thicker client to keep track of server load and implement a load balancing algorithm.

---
## Papers to guide work
1. [Google Cloud's gRPC presentation](https://platformlab.stanford.edu/Seminar%20Talks/gRPC.pdf)
2. [Load Balancing background](https://grpc.io/blog/grpc-load-balancing/#client-side-lb-options)
3. [Error Handling](https://www.usenix.org/conference/srecon19asia/presentation/sheerin)
4. [gRPC website background](https://grpc.io/blog/grpc-load-balancing/)
5. [IoT performance benefit](https://hal.archives-ouvertes.fr/hal-02495252/document)

## Useful extras
- [Golang gRPC package](https://github.com/grpc/grpc-go)
- [Article about an example gRPC use case](https://thenewstack.io/grpc-lean-mean-communication-protocol-microservices/)
- [Article about how to design gRPC services](https://www.bugsnag.com/blog/using-grpc-in-production)
