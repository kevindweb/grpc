# SOCKET CLIENT
import socket
import sys
import dataclasses
from dataclasses import dataclass
import simplejson as json
import time

STOP_CHARACTER = "\r\n\r\n"


class EnhancedJSONEncoder(json.JSONEncoder):
    def default(self, o):
        if dataclasses.is_dataclass(o):
            return dataclasses.asdict(o)
        return super().default(o)


@dataclass
class Request:
    Data:   str
    Dist:   int
    Systems: bool


@dataclass
class Response:
    Data: str = ""
    Updated: int = 0


def connect_socket(ip, port):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    try:
        s.connect((ip, port))
        return s, None
    except socket.error:
        print("Caught exception socket.error: ", socket.error)
        return s, socket.error


def send_message(fn, req, res, socket):
    # encode into bytes & json
    msg = fn.encode('utf-8') + STOP_CHARACTER.encode('utf-8') + json.dumps(req, cls=EnhancedJSONEncoder).encode(
        'utf-8') + STOP_CHARACTER.encode('utf-8') + json.dumps(res, cls=EnhancedJSONEncoder).encode('utf-8')
    socket.send(msg)


def check_code(data, code):
    return data[0] == code


def recv_message(res, socket):
    recv_data = socket.recv(1024)

    # error codes
    if check_code(recv_data, 6):
        print("too many failures executing function")
    elif check_code(recv_data, 5):
        print("failed to connect to server")
    elif check_code(recv_data, 4):
        print("no servers are currently avaiable")
    elif check_code(recv_data, 0):
        recv_data_clean = recv_data[1:]
        data = recv_data_clean.decode("utf-8").strip()
        obj_conts = json.loads(data)
        res.Updated = Response(**obj_conts).Updated
        res.Data = Response(**obj_conts).Data
        return 0

    return -1


# main
if __name__ == "__main__":
    s, err = connect_socket("localhost", 4000)
    start = time.time()
    if(err != None):
        exit(1)

    req = Request("world", 65, True)
    res = Response()

    if len(sys.argv) > 1:
        # user wanted to test the failure response
        print("Running %s" % sys.argv[1])

        # run cli function
        send_message(sys.argv[1], req, res, s)
    else:
        send_message("modifyArg", req, res, s)

    if recv_message(res, s) == -1:
        sys.exit(1)
    print(res)
    end = time.time()
    print("Time elapsed: ", end - start)
