package com.cubetiqs.tungo.protocol;

/**
 * Message types in the TunGo protocol
 */
public enum MessageType {
    HELLO("hello"),
    SERVER_HELLO("server_hello"),
    INIT("init"),
    DATA("data"),
    END("end"),
    PING("ping"),
    PONG("pong");

    private final String value;

    MessageType(String value) {
        this.value = value;
    }

    public String getValue() {
        return value;
    }

    public static MessageType fromValue(String value) {
        for (MessageType type : MessageType.values()) {
            if (type.value.equals(value)) {
                return type;
            }
        }
        throw new IllegalArgumentException("Unknown message type: " + value);
    }
}
