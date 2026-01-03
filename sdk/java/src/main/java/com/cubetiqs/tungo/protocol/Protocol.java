package com.cubetiqs.tungo.protocol;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

/**
 * Protocol utilities for encoding/decoding messages
 */
public class Protocol {
    private static final Gson GSON = new GsonBuilder().create();

    /**
     * Client type enumeration
     */
    public enum ClientType {
        AUTH("auth"),
        ANONYMOUS("anonymous");

        private final String value;

        ClientType(String value) {
            this.value = value;
        }

        public String getValue() {
            return value;
        }
    }

    /**
     * Create a client hello message
     */
    public static String createClientHello(String subdomain, String secretKey) {
        Map<String, Object> hello = new HashMap<>();
        hello.put("id", UUID.randomUUID().toString());
        hello.put("client_type", secretKey != null ? ClientType.AUTH.getValue() : ClientType.ANONYMOUS.getValue());

        if (subdomain != null && !subdomain.isEmpty()) {
            hello.put("sub_domain", subdomain);
        }

        if (secretKey != null && !secretKey.isEmpty()) {
            Map<String, String> keyMap = new HashMap<>();
            keyMap.put("key", secretKey);
            hello.put("secret_key", keyMap);
        }

        return GSON.toJson(hello);
    }

    /**
     * Create a protocol message
     */
    public static String createMessage(MessageType type, String streamId, Object data) {
        Map<String, Object> message = new HashMap<>();
        message.put("type", type.getValue());

        if (streamId != null) {
            message.put("stream_id", streamId);
        }

        if (data != null) {
            message.put("data", data);
        }

        return GSON.toJson(message);
    }

    /**
     * Decode a message from JSON
     */
    public static Message decodeMessage(String json) {
        return GSON.fromJson(json, Message.class);
    }

    /**
     * Decode server hello from JSON
     */
    public static ServerHello decodeServerHello(String json) {
        return GSON.fromJson(json, ServerHello.class);
    }

    /**
     * Generate a random stream ID
     */
    public static String generateStreamId() {
        return UUID.randomUUID().toString();
    }

    /**
     * Message class
     */
    public static class Message {
        private String type;
        private String stream_id;
        private Object data;

        public String getType() {
            return type;
        }

        public String getStreamId() {
            return stream_id;
        }

        public Object getData() {
            return data;
        }

        public MessageType getMessageType() {
            return MessageType.fromValue(type);
        }
    }

    /**
     * Server Hello message class
     */
    public static class ServerHello {
        private String type;
        private String sub_domain;
        private String hostname;
        private String public_url;
        private String client_id;
        private Map<String, String> reconnect_token;
        private String error;

        public String getType() {
            return type;
        }

        public String getSubDomain() {
            return sub_domain;
        }

        public String getHostname() {
            return hostname;
        }

        public String getPublicUrl() {
            return public_url;
        }

        public String getClientId() {
            return client_id;
        }

        public Map<String, String> getReconnectToken() {
            return reconnect_token;
        }

        public String getError() {
            return error;
        }

        public ServerHelloType getServerHelloType() {
            return ServerHelloType.fromValue(type);
        }
    }

    /**
     * Init Stream message data
     */
    public static class InitStreamData {
        private String stream_id;
        private String protocol;

        public InitStreamData(String streamId, String protocol) {
            this.stream_id = streamId;
            this.protocol = protocol;
        }

        public String getStreamId() {
            return stream_id;
        }

        public String getProtocol() {
            return protocol;
        }
    }
}
