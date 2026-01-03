package com.cubetiqs.tungo;

import com.cubetiqs.tungo.protocol.MessageType;
import com.google.gson.Gson;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.Base64;
import java.util.Map;

/**
 * Handles HTTP stream forwarding to local server
 */
class StreamHandler {
    private static final Logger logger = LoggerFactory.getLogger(StreamHandler.class);
    private static final Gson GSON = new Gson();

    private final String streamId;
    private final String localHost;
    private final int localPort;
    private final TunGoClient client;
    private HttpURLConnection connection;
    private volatile boolean closed = false;

    StreamHandler(String streamId, String localHost, int localPort, TunGoClient client) {
        this.streamId = streamId;
        this.localHost = localHost;
        this.localPort = localPort;
        this.client = client;
    }

    void start() {
        // Stream handling is initiated when we receive data
        logger.debug("Stream handler ready for: {}", streamId);
    }

    @SuppressWarnings("unchecked")
    void handleData(Object data) {
        if (closed) {
            return;
        }

        try {
            // Parse the data as JSON object containing base64 encoded data
            Map<String, Object> dataMap;
            if (data instanceof Map) {
                dataMap = (Map<String, Object>) data;
            } else if (data instanceof String) {
                dataMap = GSON.fromJson((String) data, Map.class);
            } else {
                logger.error("Invalid data format for stream {}", streamId);
                return;
            }

            // Get base64 encoded HTTP request data
            String base64Data = (String) dataMap.get("data");
            if (base64Data == null || base64Data.isEmpty()) {
                logger.debug("No data in message for stream {}", streamId);
                return;
            }

            // Decode base64 data to get raw HTTP request
            byte[] httpRequestBytes = Base64.getDecoder().decode(base64Data);
            
            // First data packet contains the HTTP request - parse and forward it
            if (connection == null) {
                initializeConnection(httpRequestBytes);
            }

            // Forward response after receiving request
            forwardResponse();
            
        } catch (Exception e) {
            logger.error("Error handling stream data: {}", e.getMessage(), e);
            close();
        }
    }

    private void initializeConnection(byte[] httpRequestBytes) throws IOException {
        // Parse raw HTTP request
        String httpRequest = new String(httpRequestBytes, "UTF-8");
        String[] lines = httpRequest.split("\r\n");
        
        if (lines.length == 0) {
            throw new IOException("Empty HTTP request");
        }
        
        // Parse request line: METHOD /path HTTP/1.1
        String[] requestLine = lines[0].split(" ");
        if (requestLine.length < 2) {
            throw new IOException("Invalid HTTP request line: " + lines[0]);
        }
        
        String method = requestLine[0];
        String path = requestLine[1];
        
        // Parse headers
        Map<String, String> headers = new java.util.HashMap<>();
        int bodyStart = 0;
        for (int i = 1; i < lines.length; i++) {
            if (lines[i].isEmpty()) {
                // Empty line marks end of headers
                bodyStart = httpRequest.indexOf("\r\n\r\n");
                if (bodyStart != -1) {
                    bodyStart += 4; // Skip the \r\n\r\n
                }
                break;
            }
            
            int colonIndex = lines[i].indexOf(':');
            if (colonIndex > 0) {
                String key = lines[i].substring(0, colonIndex).trim();
                String value = lines[i].substring(colonIndex + 1).trim();
                headers.put(key, value);
            }
        }
        
        // Build URL and create connection
        String urlString = String.format("http://%s:%d%s", localHost, localPort, path);
        logger.debug("Forwarding {} {} to local server", method, path);
        
        URL url = new URL(urlString);
        connection = (HttpURLConnection) url.openConnection();
        connection.setRequestMethod(method);
        connection.setDoOutput(true);
        connection.setDoInput(true);
        
        // Set headers
        for (Map.Entry<String, String> entry : headers.entrySet()) {
            // Skip some headers that HttpURLConnection manages
            String key = entry.getKey().toLowerCase();
            if (!key.equals("host") && !key.equals("connection") && 
                !key.equals("content-length") && !key.equals("transfer-encoding")) {
                connection.setRequestProperty(entry.getKey(), entry.getValue());
            }
        }
        
        // Write request body if present
        if (bodyStart > 0 && bodyStart < httpRequestBytes.length) {
            byte[] body = new byte[httpRequestBytes.length - bodyStart];
            System.arraycopy(httpRequestBytes, bodyStart, body, 0, body.length);
            
            if (body.length > 0) {
                try (OutputStream out = connection.getOutputStream()) {
                    out.write(body);
                    out.flush();
                }
            }
        }
    }

    private void forwardResponse() {
        if (connection == null) {
            return;
        }

        try {
            int statusCode = connection.getResponseCode();
            String statusText = connection.getResponseMessage();

            // Read response body
            InputStream inputStream = statusCode >= 400 ? 
                connection.getErrorStream() : connection.getInputStream();
            
            byte[] responseBody = new byte[0];
            if (inputStream != null) {
                responseBody = readAllBytes(inputStream);
                inputStream.close();
            }

            // Send response back through tunnel
            String base64Body = Base64.getEncoder().encodeToString(responseBody);
            
            Map<String, Object> response = new java.util.HashMap<>();
            response.put("status", statusCode);
            response.put("statusText", statusText);
            response.put("headers", connection.getHeaderFields());
            response.put("data", base64Body);

            client.sendMessage(MessageType.DATA, streamId, response);
            
            // End the stream
            client.sendMessage(MessageType.END, streamId, null);

        } catch (IOException e) {
            logger.error("Error forwarding response: {}", e.getMessage());
        } finally {
            close();
        }
    }

    private byte[] readAllBytes(InputStream input) throws IOException {
        java.io.ByteArrayOutputStream buffer = new java.io.ByteArrayOutputStream();
        byte[] data = new byte[8192];
        int nRead;
        while ((nRead = input.read(data, 0, data.length)) != -1) {
            buffer.write(data, 0, nRead);
        }
        buffer.flush();
        return buffer.toByteArray();
    }

    void close() {
        if (closed) {
            return;
        }
        closed = true;

        if (connection != null) {
            connection.disconnect();
            connection = null;
        }

        logger.debug("Stream closed: {}", streamId);
    }
}
