package main

import (
	"bufio"
	"container/list"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

// SyncQueue is a thread-safe queue
type SyncQueue struct {
	IpAddr    string
	RegPort   string
	NextPort  string
	Lock      *sync.Mutex
	Queue     *list.List
	Addresses map[string]bool
}

const (
	// StopCharacter is used by both server and client for communication
	StopCharacter = "\r\n\r\n"
	// ConnectionTimeout is how long to wait before killing connection
	ConnectionTimeout = 5 * time.Second
	// HealthInterval is the interval to check servers
	HealthInterval = 500 * time.Millisecond
	// HealthTimeout is how long we wait for a server response
	HealthTimeout = time.Second
	// MaxRetry is how many times we retry the client request on server failure
	MaxRetry = 3

	// instantiate error code constants

	// NoServers is when all servers are dead
	NoServers = 4
	// DialErr means we found server but failed to connect
	DialErr = 5
	// MaxFailures sent when we retried function too many times
	MaxFailures = 6
)

// SocketServer starts the load balancer
func SocketServer(port int) error {
	listen, err := net.Listen("tcp4", ":"+strconv.Itoa(port))

	if err != nil {
		log.Fatalf("Socket listen port %d failed,%s", port, err)
		return err
	}

	defer listen.Close()

	log.Printf("Begin listen port: %d", port)

	serverQueue := createQueue()
	go pollServers(serverQueue)

	// check server health on interval
	go setHealthInterval(serverQueue, HealthInterval, HealthTimeout)

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalln(err)
			continue
		}
		go handler(conn, serverQueue)
	}
}

func handler(conn net.Conn, queue *SyncQueue) {
	defer conn.Close()
	var (
		buf = make([]byte, 1024)
		r   = bufio.NewReader(conn)
		w   = bufio.NewWriter(conn)
	)

	n, err := r.Read(buf)

	if err != nil {
		log.Printf("Failed with err: %s", err)
		respondError(w, DialErr)
		return
	}

	passRequest(queue, buf[:n], w, 0)
}

func passRequest(queue *SyncQueue, requestBuff []byte, w *bufio.Writer, retry int) {
	// get a new server from the queue
	server := getServer(queue)
	if server == "" {
		// tell client there's no servers available
		log.Println("There are no available servers in the queue")
		respondError(w, NoServers)
		return
	}

	conn, err := net.Dial("tcp", server)
	if err != nil {
		// create error for user instead of response data
		respondError(w, DialErr)
		return
	}

	// send the server the client's encoded request
	conn.Write(requestBuff)

	clientResponse := make([]byte, 1024)
	// read response from server
	n, _ := conn.Read(clientResponse)
	if n == 0 {
		// server failed

		if retry > MaxRetry {
			// already retried too many times
			respondError(w, MaxFailures)
		} else {
			passRequest(queue, requestBuff, w, retry+1)
		}

		return
	}

	// write the response data back to the client
	w.Write(clientResponse[:n])
	w.Flush()
}

func respondError(w *bufio.Writer, errorCode int) {
	w.Write([]byte{byte(errorCode)})
	w.Flush()
}

// getServer takes in the list of servers and returns the optimal
// server as a string of "ip:port"
func getServer(queue *SyncQueue) string {
	queue.Lock.Lock()
	defer queue.Lock.Unlock()

	if queue.Queue.Len() == 0 {
		return ""
	}

	// use round robin for now
	server := queue.Queue.Front()

	queue.Queue.Remove(server)

	// round robin put to end of queue
	queue.Queue.PushBack(server.Value.(string))

	return server.Value.(string)
}

func createQueue() *SyncQueue {
	return &SyncQueue{
		IpAddr:    "localhost",
		RegPort:   "3333",
		NextPort:  "4001",
		Lock:      &sync.Mutex{},
		Queue:     list.New(),
		Addresses: make(map[string]bool),
	}
}

func pollServers(q *SyncQueue) error {
	//listen on the dedicated registration port
	listen, err := net.Listen("tcp4", ":"+q.RegPort)

	if err != nil {
		log.Fatalf("Socket listen port %s failed, %s", q.RegPort, err)
		return err
	}

	defer listen.Close()

	log.Printf("Begin listen port: %s", q.RegPort)

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalln(err)
			continue
		}
		go registerServer(q, conn)
	}
}

func registerServer(q *SyncQueue, conn net.Conn) {
	// send registration to service
	defer conn.Close()

	fn := "register-server"
	port := q.NextPort
	w := bufio.NewWriter(conn)

	// send the function name
	w.Write([]byte(fn))
	w.Write([]byte(StopCharacter))
	// write assigned port
	w.Write([]byte(port))
	w.Write([]byte(StopCharacter))
	err := w.Flush()

	if err != nil {
		log.Printf("Failed with err: %s", err)
	}

	q.Lock.Lock()
	defer q.Lock.Unlock()

	// update service container
	portNum, _ := strconv.Atoi(port)
	q.NextPort = strconv.Itoa(portNum + 1)
	address := q.IpAddr + ":" + port
	q.Queue.PushBack(address)
	printAvailableServers(q.Queue)
	q.Addresses[address] = true
}

func setHealthInterval(queue *SyncQueue, interval, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	log.Printf("Setting Health Interval")
	// set up the timer to check health over the interval
	go func() {
		for {
			select {
			case <-ticker.C:
				checkHealth(queue, timeout)
			}
		}
	}()
}

// checkHealth continuously iterates through the server queue
// and removes servers that are dead
func checkHealth(queue *SyncQueue, timeout time.Duration) {
	// make thread safe
	queue.Lock.Lock()
	defer queue.Lock.Unlock()

	failed := false
	for e := queue.Queue.Front(); e != nil; e = e.Next() {
		// test that we can hit the server
		addr := e.Value.(string)
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			// server was dead, handle error
			queue.Queue.Remove(e)
			failed = true
			log.Printf("Health check of %s failed", addr)
			continue
		}

		if conn != nil {
			// connection was live
			conn.Close()
		}
	}

	if failed {
		printAvailableServers(queue.Queue)
	}
}

func printAvailableServers(q *list.List) {
	// listen for services attempting to connect
	if q.Len() > 0 {
		log.Printf("Listing available servers: %d", q.Len())
	}

	for e := q.Front(); e != nil; e = e.Next() {
		log.Printf("Server port: %s", e.Value.(string))
	}
}

func main() {
	port := 4000

	// start listening on load balancer port
	SocketServer(port)
}
