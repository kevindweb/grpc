# SOCKET SERVER
import socket
import sys
import json
import signal
from random import randint, getrandbits
from time import sleep


class EnhancedJSONEncoder(json.JSONEncoder):
    def default(self, o):
        if dataclasses.is_dataclass(o):
            return dataclasses.asdict(o)
        return super().default(o)


# Class to allow dot notation accesss to dictionaries
class Map(dict):
    """
    Example:
    m = Map({'first_name': 'Eduardo'}, last_name='Pool', age=24, sports=['Soccer'])
    """

    def __init__(self, *args, **kwargs):
        super(Map, self).__init__(*args, **kwargs)
        for arg in args:
            if isinstance(arg, dict):
                for k, v in arg.items():
                    self[k] = v

        if kwargs:
            for k, v in kwargs.items():
                self[k] = v

    def __getattr__(self, attr):
        return self.get(attr)

    def __setattr__(self, key, value):
        self.__setitem__(key, value)

    def __setitem__(self, key, value):
        super(Map, self).__setitem__(key, value)
        self.__dict__.update({key: value})

    def __delattr__(self, item):
        self.__delitem__(item)

    def __delitem__(self, key):
        super(Map, self).__delitem__(key)
        del self.__dict__[key]


def modifyArg(req, res):
    res.Data = req.Data + " world"
    res.Updated = req.Dist
    if(req.Systems):
        res.Updated += 1

    return False


def differentFunc(req, res):
    res.NotData = req.AnotherOne
    res.NotData[1] = "databases"

    return False


# 100% chance to fail after 2 seconds
def fullFail(req, res):
    sleeptime = randint(1, 3)
    sleep(sleeptime)

    # always say we failed
    print("Failed after sleeping for %d seconds" % sleeptime)
    return True


# 50% chance to fail and provide no response data
def randomFail(req, res):
    if bool(getrandbits(1)):
        print("X Unrecoverable failure X")
        # return that we had a failure
        return True

    res.Data = req.Data + " not failed"
    res.Updated = req.Dist * 2

    return False


HOST = "localhost"
LB_PORT = 3333
STOP_CHARACTER = "\r\n\r\n"
LB_RETRY = 5


def signal_handler(sig, frame):
    print('Exiting safely')
    sys.exit(0)


def registrationServer(port):
    # hit the lb and get a port to wait for responses
    count = 0
    while count < LB_RETRY:
        print("Dialing LB on port:", port)

        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.connect((HOST, port))
            data = s.recv(1024)
            messages = data.split(STOP_CHARACTER.encode('utf-8'))

            fn_name = messages[0].decode('utf-8')
            if fn_name == "register-server":
                return int(messages[1].decode('utf-8'))

        count += 1

    # failed
    return -1


def startServer(host, port, function_map):
    print("Starting Python grpc at:", host, port)
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        s.bind((host, port))
        s.listen()

        # wait forever for requests
        while True:
            conn, addr = s.accept()
            with conn:
                handleConnection(conn)


def handleConnection(conn):
    data = conn.recv(1024)
    if len(data) == 0:
        # probably health check
        return

    # get individual messages
    arr = data.split(STOP_CHARACTER.encode('utf-8'))

    fn_name = arr[0].decode('utf-8')
    print("Function %s received" % fn_name)

    obj_conts = json.loads((arr[1].decode('utf-8')))
    req = Map(obj_conts)

    if arr[2].decode('utf') == "":
        # need to handle case when this data is null initially
        obj_conts_2 = {}
    else:
        obj_conts_2 = json.loads((arr[2].decode('utf-8')))

    res = Map(obj_conts_2)

    # call function from a "switch" statement
    failed = function_map[fn_name](req, res)
    if failed:
        return

    # send the error code 0 (no-error) and the response
    conn.sendall(b'\x00'+json.dumps(res,
                                    cls=EnhancedJSONEncoder).encode("utf-8"))


if __name__ == "__main__":
    # define map of function string to functions
    function_map = {
        "modifyArg": modifyArg,
        "differentFunc": differentFunc,
        "randomFail": randomFail,
        "fullFail": fullFail,
    }

    port = registrationServer(LB_PORT)
    if port == -1:
        print("Failed to register a server port")
        sys.exit(1)

    # define signal handler to gracefully exit on C-c
    signal.signal(signal.SIGINT, signal_handler)

    startServer(HOST, port, function_map)
