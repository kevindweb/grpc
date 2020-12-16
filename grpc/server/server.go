package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"time"
)

const StopCharacter = "\r\n\r\n"
const ConnectionTimeout = 5 * time.Second

func RegistrationServer(port int) (int, error) {

	assigned_port := -1

	for {
		conn, err := net.Dial("tcp4", ":"+strconv.Itoa(port))
		if err != nil {
			log.Fatalf("Error here: %s", err)
			continue
		} else {
			assigned_port, _ = RegistrationHandler(conn)
			if assigned_port != -1 {
				break
			}
		}
	}

	return assigned_port, nil
}

func RegistrationHandler(conn net.Conn) (int, error) {

	assigned_port := -1

	defer conn.Close()
	var (
		buf = make([]byte, 1024)
		r   = bufio.NewReader(conn)
		w   = bufio.NewWriter(conn)
	)

	n, err := r.Read(buf)
	if err != nil {
		log.Println("Error reading registration handler")
		return -1, err
	}

	assigned_port = handleRegistration(buf[:n], w)
	return assigned_port, nil
}

func handleRegistration(buf []byte, w *bufio.Writer) int {
	// req separates function name from data with a stop character
	messages := bytes.Split(buf, []byte(StopCharacter))

	functionName := string(messages[0])
	if functionName == "health-check" || functionName == "" {
		// load balancer is checking our heartbeat
		return -1
	}

	if functionName == "register-server" {
		assigned_port, err := strconv.Atoi(string(messages[1]))
		if err != nil {
			log.Fatalf("Registration failed,%s", err)
			return -1
		}
		return assigned_port
	}

	return -1
}

func SocketServer(port int) error {
	listen, err := net.Listen("tcp4", ":"+strconv.Itoa(port))

	if err != nil {
		log.Fatalf("Socket listen port %d failed,%s", port, err)
		return err
	}

	defer listen.Close()

	log.Printf("Begin listen port: %d", port)

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalf("Error here: %s", err)
			continue
		}
		go handler(conn)
	}
}

func handler(conn net.Conn) {
	defer conn.Close()
	var (
		buf = make([]byte, 1024)
		r   = bufio.NewReader(conn)
		w   = bufio.NewWriter(conn)
	)

	n, err := r.Read(buf)
	if err != nil {
		return
	}

	if n == 0 {
		// probably a health check
		return
	}

	// handle the request data
	handleRequest(buf[:n], w)
}

func handleRequest(buf []byte, w *bufio.Writer) {
	// req separates function name from data with a stop character
	messages := bytes.Split(buf, []byte(StopCharacter))

	functionName := string(messages[0])
	if functionName == "health-check" || functionName == "" {
		// load balancer is checking our heartbeat
		return
	}
	fmt.Println("Function name", functionName)

	funcMap := functionList()
	if functionName == "function-registration" {
		//load balancer is trying to register the service on the wrong port
		log.Fatalf("Load Balancer attempting to re-register service")
		return

	} else {
		// get the parameters from the functionmap and function name
		function, reqv, replyv := Arguments(funcMap, functionName)
		// log.Printf("Function %s received\n", functionName)

		if err := json.Unmarshal(messages[1], reqv.Interface()); err != nil {
			fmt.Println(err)
			return
		}

		in := make([]reflect.Value, 2)
		in[0] = reqv
		in[1] = replyv

		// reply is sent back from a pointer in the Call function
		err := Call(function, in)
		if err != nil {
			fmt.Println(err)
			errorCode(w, 6)
			return
		}

		// return data to client
		b, err := json.Marshal(replyv.Interface())
		if err != nil {
			fmt.Println(err)
			return
		}

		// write a 0 error code
		errorCode(w, 0)

		w.Write(b)
		w.Flush()
	}
}

func errorCode(w *bufio.Writer, code int) {
	w.Write([]byte{byte(code)})
}

// Arguments returns the function, arg1, arg2 from a string function name
func Arguments(m map[string]interface{}, name string) (function, reqv, replyv reflect.Value) {
	function = reflect.ValueOf(m[name])
	if !function.IsValid() {
		// something went wrong
		log.Println("Failed to parse arguments for", name)
		return
	}

	// need to get the abstracted request parameter variable types
	reqv = reflect.New(function.Type().In(0).Elem())
	replyv = reflect.New(function.Type().In(1).Elem())

	return
}

// Call takes in arguments and calls a function by string name
func Call(f reflect.Value, in []reflect.Value) (err error) {
	// best way to handle if f.Call panics
	defer func() {
		if r := recover(); r != nil {
			// use parent function throw err
			err = errors.New(r.(string))
		}
	}()

	// call the requested function with arguments
	res := f.Call(in)
	if len(res) > 0 {
		out := res[0].Interface()
		if out != nil {
			return out.(error)
		}
	}
	return nil
}

type Request struct {
	Data    string
	Dist    int
	Systems bool
}

type Response struct {
	Data    string
	Updated int
}

func modifyArg(req *Request, res *Response) {
	res.Data = req.Data + " world"
	res.Updated = req.Dist

	if req.Systems {
		res.Updated++
	}
}

func noModifyArg(req *Request, res *Response) {
	res.Data = req.Data
	res.Updated = req.Dist
}

type DiffRequest struct {
	AnotherOne []string
}

type DiffRes struct {
	NotData []string
}

func differentFunc(req *DiffRequest, res *DiffRes) {
	res.NotData = req.AnotherOne
	res.NotData[1] = "databases"
}

func randomFail(req *Request, res *Response) error {
	if rand.Float32() < 0.5 {
		return errors.New("X Unrecoverable failure X")
	}

	res.Data = req.Data + " not failed"
	res.Updated = req.Dist * 2
	return nil
}

func fullFail(req *Request, res *Response) error {
	sleepTime := rand.Intn(3) + 1
	time.Sleep(time.Duration(sleepTime) * time.Microsecond)

	return fmt.Errorf("Failed after sleeping for %d seconds", sleepTime)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var randFile string

func randomFileSetup() {
	randFile = "randFile" + strconv.Itoa(rand.Intn(1000))
	err := ioutil.WriteFile(randFile, []byte("Random file\n"), 0644)
	check(err)
}

func ioBound(req *Request, res *Response) error {
	_, err := ioutil.ReadFile(randFile)

	cnt := 1.001
	for i := 0; i < 900; i++ {
		cnt *= 1.001
	}

	// time.Sleep(1 * time.Second)
	var slice = make([]int, 100000)

	for i := 0; i < 100000; i++ {
		slice[i] = int(math.Pow(3, float64(rand.Intn(10))))
	}

	return err
}

func functionList() map[string]interface{} {
	return map[string]interface{}{
		"modifyArg":     modifyArg,
		"noModifyArg":   noModifyArg,
		"differentFunc": differentFunc,
		"randomFail":    randomFail,
		"fullFail":      fullFail,
		"ioBound":       ioBound,
	}
}

func main() {
	// set up the file for the IO Bound function
	rand.Seed(time.Now().UTC().UnixNano())
	randomFileSetup()

	registrationPort := 3333
	log.Printf("Listening for Loadbalancer on port:%d", registrationPort)
	loadbalancerPort, _ := RegistrationServer(registrationPort)
	if loadbalancerPort > 0 {
		SocketServer(loadbalancerPort)
	}
}
