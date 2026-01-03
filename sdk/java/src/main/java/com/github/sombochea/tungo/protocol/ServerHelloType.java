package com.github.sombochea.tungo.protocol;

/**
 * Server hello response types
 */
public enum ServerHelloType {
    SUCCESS("success"),
    SUBDOMAIN_IN_USE("sub_domain_in_use"),
    INVALID_SUBDOMAIN("invalid_sub_domain"),
    AUTH_FAILED("auth_failed"),
    ERROR("error");

    private final String value;

    ServerHelloType(String value) {
        this.value = value;
    }

    public String getValue() {
        return value;
    }

    public static ServerHelloType fromValue(String value) {
        for (ServerHelloType type : ServerHelloType.values()) {
            if (type.value.equals(value)) {
                return type;
            }
        }
        throw new IllegalArgumentException("Unknown server hello type: " + value);
    }
}
