package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Grpc is a wrapper object
type Grpc struct {
	Conn net.Conn
	Addr string
}

const (
	// StopCharacter is used by both server and client for communication
	StopCharacter = "\r\n\r\n"
)

// Dial instantiates a grpc client and ensures load balancer is available
func Dial(protocol string, ip string, port int) (*Grpc, error) {
	addr := strings.Join([]string{ip, strconv.Itoa(port)}, ":")

	return &Grpc{
		Addr: addr,
	}, nil
}

// Call creates a connection to run function with args, res will hold response data
func (c *Grpc) Call(fn string, req interface{}, res interface{}) error {
	/* data is in the format of:
	1. function name
	2. stop character (to denote start of data)
	3. encoded data
	4. stop character (to denote connection data is done)
	*/

	conn, err := net.Dial("tcp", c.Addr)

	if err != nil {
		return err
	}
	defer conn.Close()

	w := bufio.NewWriter(conn)

	// send the function name
	w.Write([]byte(fn))

	// stop character
	w.Write([]byte(StopCharacter))

	// encode request data
	data, err := json.Marshal(&req)
	if err != nil {
		return err
	}

	// write data to connection
	w.Write(data)

	// stop character
	w.Write([]byte(StopCharacter))
	w.Flush()

	buff := make([]byte, 1024)
	// read response from server
	n, _ := conn.Read(buff)

	// first byte of response is the status code
	returnStatus := int(buff[0])

	if returnStatus > 0 {
		return fmt.Errorf("Failure with exit code %d", returnStatus)
	}

	if n <= 0 {
		return fmt.Errorf("No data returned")
	}

	// _, err = xdr.Unmarshal(bytes.NewReader(buff[1:n]), res)
	if err := json.Unmarshal(buff[1:n], res); err != nil {
		return err
	}

	return nil
}
