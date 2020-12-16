import java.net.*;
import java.io.*;
import com.google.gson.Gson;

public class Server {
    private ServerSocket serverSocket;
    private Socket clientSocket;
    private PrintWriter out;
    private BufferedReader in;
    public class Request{
        public String data;
        public int dist;
        public boolean systems;
    }
    public class Response{
        public String data;
        public int updated;
    }

    public static void main(String[] args) {
        Server server = new Server();
        String recv = server.start(4000);
        if(recv != ""){
            Gson gson = new Gson(); 
            Server.Request r = server.new Request();
            Server.Response resp = server.new Response();
            r = gson.fromJson(recv, Request.class);
            server.modifyArgs(r,resp);
            String resp_correct = gson.toJson(resp);
            System.out.println(resp_correct);
        }
        server.stop();
    }

    public void modifyArgs(Request req, Response resp){
        resp.data = req.data + " world";
	    resp.updated = req.dist;

	    if (req.systems) {
		    resp.updated++;
	    }
    }

    public String start(int port) {
        try {
            serverSocket = new ServerSocket(port);
            clientSocket = serverSocket.accept();
            out = new PrintWriter(clientSocket.getOutputStream(), true);
            in = new BufferedReader(new InputStreamReader(clientSocket.getInputStream()));
            String greeting = in.readLine();
            System.out.println(greeting);
            return greeting;
        } catch (IOException i) {
            System.out.println("IO Exception: " + i);
        }
        return "";
    }

    public void stop() {
        try {
            in.close();
        } catch (IOException i) {
            System.out.println("IO Exception: " + i);
        }
        out.close();
        try {
            clientSocket.close();
        } catch (IOException i) {
            System.out.println("IO Exception: " + i);
        }
        try {
            serverSocket.close();
        } catch (IOException i) {
            System.out.println("IO Exception: " + i);
        }
    }
}