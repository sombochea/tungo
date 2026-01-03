package com.github.sombochea.tungo;

/**
 * Event listener interface for TunGo Client events
 */
public interface TunGoEventListener {
    
    /**
     * Called when connection is established and tunnel is ready
     */
    default void onConnect(TunnelInfo tunnelInfo) {}
    
    /**
     * Called when connection is closed
     */
    default void onDisconnect(String reason) {}
    
    /**
     * Called when an error occurs
     */
    default void onError(Throwable error) {}
    
    /**
     * Called when attempting to reconnect
     */
    default void onReconnect(int attempt, int maxRetries) {}
    
    /**
     * Called when status message is received
     */
    default void onStatus(String message) {}
}
