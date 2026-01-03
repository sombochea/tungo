package com.cubetiqs.tungo.protocol;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class ProtocolTest {

    @Test
    void testCreateClientHello() {
        String hello = Protocol.createClientHello("test-subdomain", "secret-key");
        
        assertNotNull(hello);
        assertTrue(hello.contains("\"client_type\":\"auth\""));
        assertTrue(hello.contains("\"sub_domain\":\"test-subdomain\""));
        assertTrue(hello.contains("\"secret_key\""));
    }

    @Test
    void testCreateClientHelloAnonymous() {
        String hello = Protocol.createClientHello(null, null);
        
        assertNotNull(hello);
        assertTrue(hello.contains("\"client_type\":\"anonymous\""));
        assertFalse(hello.contains("sub_domain"));
        assertFalse(hello.contains("secret_key"));
    }

    @Test
    void testCreateMessage() {
        String message = Protocol.createMessage(MessageType.PING, null, null);
        
        assertNotNull(message);
        assertTrue(message.contains("\"type\":\"ping\""));
    }

    @Test
    void testCreateMessageWithData() {
        String message = Protocol.createMessage(MessageType.DATA, "stream-123", "test data");
        
        assertNotNull(message);
        assertTrue(message.contains("\"type\":\"data\""));
        assertTrue(message.contains("\"stream_id\":\"stream-123\""));
        assertTrue(message.contains("test data"));
    }

    @Test
    void testDecodeMessage() {
        String json = "{\"type\":\"ping\"}";
        Protocol.Message message = Protocol.decodeMessage(json);
        
        assertNotNull(message);
        assertEquals("ping", message.getType());
        assertEquals(MessageType.PING, message.getMessageType());
    }

    @Test
    void testDecodeServerHello() {
        String json = "{\"type\":\"success\",\"sub_domain\":\"test\",\"hostname\":\"test.example.com\",\"client_id\":\"123\"}";
        Protocol.ServerHello hello = Protocol.decodeServerHello(json);
        
        assertNotNull(hello);
        assertEquals("success", hello.getType());
        assertEquals("test", hello.getSubDomain());
        assertEquals("test.example.com", hello.getHostname());
        assertEquals("123", hello.getClientId());
        assertEquals(ServerHelloType.SUCCESS, hello.getServerHelloType());
    }

    @Test
    void testGenerateStreamId() {
        String id1 = Protocol.generateStreamId();
        String id2 = Protocol.generateStreamId();
        
        assertNotNull(id1);
        assertNotNull(id2);
        assertNotEquals(id1, id2);
        assertTrue(id1.matches("[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"));
    }
}
