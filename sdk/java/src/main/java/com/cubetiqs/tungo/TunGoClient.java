package com.cubetiqs.tungo;

import com.cubetiqs.tungo.protocol.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.websocket.*;
import java.io.IOException;
import java.net.URI;
import java.util.Map;
import java.util.concurrent.*;

/**
 * TunGo Client - Expose your local server to the internet
 */
@ClientEndpoint
public class TunGoClient {
    private static final Logger logger = LoggerFactory.getLogger(TunGoClient.class);

    private final TunGoOptions options;
    private final TunGoEventListener eventListener;
    private Session session;
    private TunnelInfo tunnelInfo;
    private int reconnectAttempts = 0;
    private final Map<String, StreamHandler> streams = new ConcurrentHashMap<>();
    private ScheduledExecutorService scheduler;
    private ScheduledFuture<?> pingTask;
    private final CountDownLatch connectLatch = new CountDownLatch(1);
    private Exception connectError;

    public TunGoClient(TunGoOptions options) {
        this(options, null);
    }

    public TunGoClient(TunGoOptions options, TunGoEventListener eventListener) {
        this.options = options;
        this.eventListener = eventListener != null ? eventListener : new TunGoEventListener() {};
        this.scheduler = Executors.newScheduledThreadPool(2);
    }

    /**
     * Start the tunnel
     */
    public TunnelInfo start() throws Exception {
        if (session != null && session.isOpen()) {
            throw new IllegalStateException("Tunnel is already running");
        }

        // Build WebSocket URL
        String wsUrl;
        if (options.getServerUrl() != null && !options.getServerUrl().isEmpty()) {
            wsUrl = options.getServerUrl();
            if (!wsUrl.startsWith("ws://") && !wsUrl.startsWith("wss://")) {
                wsUrl = "ws://" + wsUrl;
            }
            if (!wsUrl.endsWith("/ws")) {
                wsUrl = wsUrl.endsWith("/") ? wsUrl + "ws" : wsUrl + "/ws";
            }
        } else {
            wsUrl = String.format("ws://%s:%d/ws", 
                options.getServerHost(), 
                options.getControlPort());
        }

        logger.info("Connecting to TunGo server: {}", wsUrl);

        final WebSocketContainer container = ContainerProvider.getWebSocketContainer();
        final String finalWsUrl = wsUrl;
        
        try {
            // Connect with timeout
            ExecutorService executor = Executors.newSingleThreadExecutor();
            Future<Session> future = executor.submit(() -> 
                container.connectToServer(this, URI.create(finalWsUrl)));
            
            session = future.get(options.getConnectTimeout(), TimeUnit.MILLISECONDS);
            executor.shutdown();

            // Wait for server hello with timeout
            boolean connected = connectLatch.await(options.getConnectTimeout(), TimeUnit.MILLISECONDS);
            
            if (!connected) {
                stop();
                throw new TimeoutException("Connection timeout waiting for server hello");
            }

            if (connectError != null) {
                stop();
                throw connectError;
            }

            if (tunnelInfo == null) {
                stop();
                throw new Exception("Failed to establish tunnel");
            }

            // Start ping task
            startPingTask();

            return tunnelInfo;

        } catch (TimeoutException e) {
            stop();
            throw new TimeoutException("Connection timeout");
        } catch (Exception e) {
            stop();
            throw new Exception("Failed to connect: " + e.getMessage(), e);
        }
    }

    /**
     * Stop the tunnel
     */
    public void stop() {
        logger.info("Stopping tunnel...");

        if (pingTask != null) {
            pingTask.cancel(false);
            pingTask = null;
        }

        // Close all streams
        for (Map.Entry<String, StreamHandler> entry : streams.entrySet()) {
            try {
                entry.getValue().close();
            } catch (Exception e) {
                logger.debug("Error closing stream: {}", e.getMessage());
            }
        }
        streams.clear();

        if (session != null && session.isOpen()) {
            try {
                session.close();
            } catch (IOException e) {
                logger.debug("Error closing session: {}", e.getMessage());
            }
            session = null;
        }

        tunnelInfo = null;
        reconnectAttempts = 0;

        if (eventListener != null) {
            eventListener.onStatus("stopped");
        }
    }

    /**
     * Shutdown the client and cleanup resources
     */
    public void shutdown() {
        stop();
        if (scheduler != null && !scheduler.isShutdown()) {
            scheduler.shutdown();
            try {
                if (!scheduler.awaitTermination(5, TimeUnit.SECONDS)) {
                    scheduler.shutdownNow();
                }
            } catch (InterruptedException e) {
                scheduler.shutdownNow();
                Thread.currentThread().interrupt();
            }
        }
    }

    @OnOpen
    public void onOpen(Session session) {
        logger.debug("WebSocket connected");
        this.session = session;
        sendClientHello();
    }

    @OnMessage
    public void onMessage(String message) {
        try {
            // Check if this is ServerHello (first message after connection)
            if (tunnelInfo == null) {
                Protocol.ServerHello hello = Protocol.decodeServerHello(message);
                handleServerHello(hello);
            } else {
                // Handle regular protocol messages
                Protocol.Message msg = Protocol.decodeMessage(message);
                handleMessage(msg);
            }
        } catch (Exception e) {
            logger.error("Failed to decode message: {}", e.getMessage());
        }
    }

    @OnError
    public void onError(Session session, Throwable error) {
        logger.error("WebSocket error: {}", error.getMessage());
        if (tunnelInfo == null) {
            connectError = new Exception(error);
            connectLatch.countDown();
        }
        if (eventListener != null) {
            eventListener.onError(error);
        }
    }

    @OnClose
    public void onClose(Session session, CloseReason reason) {
        logger.info("WebSocket closed: {}", reason);

        if (pingTask != null) {
            pingTask.cancel(false);
            pingTask = null;
        }

        this.session = null;

        if (tunnelInfo != null) {
            if (eventListener != null) {
                eventListener.onDisconnect(reason.getReasonPhrase());
            }
            handleReconnect();
        } else if (connectError == null) {
            connectError = new Exception("Connection closed before tunnel established");
            connectLatch.countDown();
        }
    }

    private void sendClientHello() {
        String hello = Protocol.createClientHello(
            options.getSubdomain() != null && !options.getSubdomain().isEmpty() ? options.getSubdomain() : null,
            options.getSecretKey() != null && !options.getSecretKey().isEmpty() ? options.getSecretKey() : null
        );

        if (session != null && session.isOpen()) {
            try {
                session.getBasicRemote().sendText(hello);
                logger.debug("Sent client hello");
            } catch (IOException e) {
                logger.error("Failed to send client hello: {}", e.getMessage());
            }
        }
    }

    private void handleServerHello(Protocol.ServerHello hello) {
        if (hello.getServerHelloType() != ServerHelloType.SUCCESS) {
            String error = hello.getError() != null ? hello.getError() : 
                "Server hello failed: " + hello.getType();
            connectError = new Exception(error);
            connectLatch.countDown();
            return;
        }

        String publicUrl = hello.getPublicUrl() != null ? hello.getPublicUrl() :
            "http://" + hello.getHostname();

        tunnelInfo = new TunnelInfo(
            publicUrl,
            hello.getSubDomain(),
            hello.getClientId()
        );

        // Store subdomain back to options for reconnection
        if (hello.getSubDomain() != null) {
            options.getClass(); // Keep the original options immutable
        }

        logger.info("Tunnel established: {}", publicUrl);
        connectLatch.countDown();

        if (eventListener != null) {
            eventListener.onConnect(tunnelInfo);
        }
    }

    private void handleMessage(Protocol.Message message) {
        MessageType type = message.getMessageType();

        switch (type) {
            case INIT:
                handleInitStream(message.getStreamId(), message.getData());
                break;
            case DATA:
                handleStreamData(message.getStreamId(), message.getData());
                break;
            case END:
                handleStreamEnd(message.getStreamId());
                break;
            case PING:
                handlePing();
                break;
            default:
                logger.debug("Unknown message type: {}", message.getType());
        }
    }

    private void handleInitStream(String streamId, Object data) {
        logger.debug("Initiating stream: {}", streamId);

        StreamHandler handler = new StreamHandler(
            streamId,
            options.getLocalHost(),
            options.getLocalPort(),
            this
        );

        streams.put(streamId, handler);
        handler.start();
    }

    private void handleStreamData(String streamId, Object data) {
        StreamHandler handler = streams.get(streamId);
        if (handler != null) {
            handler.handleData(data);
        }
    }

    private void handleStreamEnd(String streamId) {
        logger.debug("Ending stream: {}", streamId);
        StreamHandler handler = streams.remove(streamId);
        if (handler != null) {
            handler.close();
        }
    }

    private void handlePing() {
        sendMessage(MessageType.PONG, null, null);
    }

    private void handleReconnect() {
        if (reconnectAttempts >= options.getMaxRetries()) {
            logger.warn("Max reconnection attempts reached ({}), resetting counter with extended delay", 
                options.getMaxRetries());
            
            reconnectAttempts = 0;
            
            // Extended delay (6x normal interval, capped at 30 seconds)
            int extendedDelay = Math.min(options.getRetryInterval() * 6, 30000);
            
            if (eventListener != null) {
                eventListener.onReconnect(reconnectAttempts + 1, options.getMaxRetries());
            }
            
            scheduleReconnect(extendedDelay);
            return;
        }

        reconnectAttempts++;
        
        // Calculate delay with exponential backoff
        int delay = Math.min(
            options.getRetryInterval() * (int) Math.pow(2, reconnectAttempts - 1),
            30000 // Cap at 30 seconds
        );

        logger.info("Reconnecting in {}ms (attempt {}/{})", 
            delay, reconnectAttempts, options.getMaxRetries());

        if (eventListener != null) {
            eventListener.onReconnect(reconnectAttempts, options.getMaxRetries());
        }

        scheduleReconnect(delay);
    }

    private void scheduleReconnect(int delayMs) {
        scheduler.schedule(() -> {
            try {
                start();
                reconnectAttempts = 0; // Reset on successful connection
            } catch (Exception e) {
                logger.error("Reconnection failed: {}", e.getMessage());
                handleReconnect();
            }
        }, delayMs, TimeUnit.MILLISECONDS);
    }

    private void startPingTask() {
        pingTask = scheduler.scheduleAtFixedRate(() -> {
            if (session != null && session.isOpen()) {
                sendMessage(MessageType.PING, null, null);
            }
        }, 30, 30, TimeUnit.SECONDS);
    }

    void sendMessage(MessageType type, String streamId, Object data) {
        if (session != null && session.isOpen()) {
            try {
                String message = Protocol.createMessage(type, streamId, data);
                session.getBasicRemote().sendText(message);
            } catch (IOException e) {
                logger.error("Failed to send message: {}", e.getMessage());
            }
        }
    }

    public TunnelInfo getTunnelInfo() {
        return tunnelInfo;
    }

    public boolean isConnected() {
        return session != null && session.isOpen() && tunnelInfo != null;
    }
}
