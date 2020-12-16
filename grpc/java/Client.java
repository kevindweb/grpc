import java.net.*;
import java.io.*;
import com.google.gson.Gson;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;

public class Client {
    private static final String STOP_CHARACTER = "\r\n\r\n";
    private Socket clientSocket;
    private InputStream in;
    private OutputStream os;

    public class Request {
        public String data;
        public int dist;
        public boolean systems;
    }

    // capital var names to match json sent from go
    public class Response {
        public String Data;
        public int Updated;
    }

    public class Returned {
        public int errorCode;
        public String response;
    }

    public static void main(String[] args) {
        Client client = new Client();
        client.startConnection("127.0.0.1", 4000);
        long startTime = System.nanoTime();    
        Client.Request r = client.new Request();
        r.data = "world";
        r.dist = 5;
        r.systems = true;

        Client.Response res = client.new Response();
        Client.Returned ret = client.new Returned();

        Gson gson = new Gson();
        String json_req = gson.toJson(r);
        String json_res = gson.toJson(res);
        client.sendMessage("modifyArg", json_req, json_res);
        ret = client.recvMessage(ret);
        
        // error codes
        if (ret.errorCode == 0) {
            res = gson.fromJson(ret.response, Client.Response.class);
            System.out.println(ret.response);
        } else if (ret.errorCode == 6) {
            System.out.println("too many failures executing function");
        } else if (ret.errorCode == 5) {
            System.out.println("failed to connect to server");
        } else if (ret.errorCode == 4) {
            System.out.println("no servers are currently avaiable");
        }
        client.stopConnection();
        long endTime = System.nanoTime() - startTime;
        System.out.println("Time elapsed: "+ endTime);
    }

    public void startConnection(String ip, int port) {
        try {
            clientSocket = new Socket(ip, port);
        } catch (IOException i) {
            System.out.println("IO Exception: " + i);
            System.exit(1);
        }
        try {
            os = clientSocket.getOutputStream();
        } catch (IOException i) {
            System.out.println("IO Exception: " + i);
            System.exit(1);
        }
        try {
            in = clientSocket.getInputStream();
        } catch (IOException i) {
            System.out.println("IO Exception: " + i);
            System.exit(1);
        }
    }

    public void sendMessage(String fn, String msg_req, String msg_res) {
        StringBuilder str = new StringBuilder(fn);
        str.append(STOP_CHARACTER);
        str.append(msg_req);
        str.append(STOP_CHARACTER);
        str.append(msg_res);

        byte[] total_b = str.toString().getBytes(StandardCharsets.UTF_8);

        try {
            os.write(total_b);
            os.flush();
        } catch (IOException i) {
            System.out.println("IO Exception: " + i);
            System.exit(1);
        }
    }

    public Returned recvMessage(Returned ret) {
        byte[] resp = null;
        try {
            resp = in.readAllBytes();
            ret.errorCode = resp[0];
            ret.response = new String(Arrays.copyOfRange(resp, 1, resp.length), StandardCharsets.UTF_8);

        } catch (IOException i) {
            System.out.println("IO Exception: " + i);
            System.exit(1);
        }
        return ret;
    }

    public void stopConnection() {
        try {
            in.close();
        } catch (IOException i) {
            System.out.println("IO Exception: " + i);
            System.exit(1);
        }
        try {
            os.close();
        } catch (IOException i) {
            System.out.println("IO Exception: " + i);
            System.exit(1);
        }
        try {
            clientSocket.close();
        } catch (IOException i) {
            System.out.println("IO Exception: " + i);
            System.exit(1);
        }
    }
}
