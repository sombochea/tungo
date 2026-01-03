package com.cubetiqs.tungo;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class TunGoOptionsTest {

    @Test
    void testBuilderDefaults() {
        TunGoOptions options = TunGoOptions.builder(3000).build();

        assertEquals(3000, options.getLocalPort());
        assertEquals("", options.getServerUrl());
        assertEquals("localhost", options.getServerHost());
        assertEquals(5555, options.getControlPort());
        assertEquals("localhost", options.getLocalHost());
        assertEquals("", options.getSubdomain());
        assertEquals("", options.getSecretKey());
        assertEquals(10000, options.getConnectTimeout());
        assertEquals(5, options.getMaxRetries());
        assertEquals(5000, options.getRetryInterval());
        assertEquals("INFO", options.getLogLevel());
    }

    @Test
    void testBuilderCustomValues() {
        TunGoOptions options = TunGoOptions.builder(8080)
            .serverHost("example.com")
            .controlPort(6000)
            .localHost("127.0.0.1")
            .subdomain("test")
            .secretKey("secret123")
            .connectTimeout(15000)
            .maxRetries(10)
            .retryInterval(3000)
            .logLevel("DEBUG")
            .build();

        assertEquals(8080, options.getLocalPort());
        assertEquals("example.com", options.getServerHost());
        assertEquals(6000, options.getControlPort());
        assertEquals("127.0.0.1", options.getLocalHost());
        assertEquals("test", options.getSubdomain());
        assertEquals("secret123", options.getSecretKey());
        assertEquals(15000, options.getConnectTimeout());
        assertEquals(10, options.getMaxRetries());
        assertEquals(3000, options.getRetryInterval());
        assertEquals("DEBUG", options.getLogLevel());
    }

    @Test
    void testServerUrl() {
        TunGoOptions options = TunGoOptions.builder(3000)
            .serverUrl("wss://tunnel.example.com")
            .build();

        assertEquals("wss://tunnel.example.com", options.getServerUrl());
    }
}
