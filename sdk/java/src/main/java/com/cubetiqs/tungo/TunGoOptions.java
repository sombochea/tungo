package com.cubetiqs.tungo;

/**
 * TunGo Client Options
 */
public class TunGoOptions {
    private final String serverUrl;
    private final String serverHost;
    private final int controlPort;
    private final String localHost;
    private final int localPort;
    private final String subdomain;
    private final String secretKey;
    private final int connectTimeout;
    private final int maxRetries;
    private final int retryInterval;
    private final String logLevel;

    private TunGoOptions(Builder builder) {
        this.serverUrl = builder.serverUrl;
        this.serverHost = builder.serverHost;
        this.controlPort = builder.controlPort;
        this.localHost = builder.localHost;
        this.localPort = builder.localPort;
        this.subdomain = builder.subdomain;
        this.secretKey = builder.secretKey;
        this.connectTimeout = builder.connectTimeout;
        this.maxRetries = builder.maxRetries;
        this.retryInterval = builder.retryInterval;
        this.logLevel = builder.logLevel;
    }

    public String getServerUrl() { return serverUrl; }
    public String getServerHost() { return serverHost; }
    public int getControlPort() { return controlPort; }
    public String getLocalHost() { return localHost; }
    public int getLocalPort() { return localPort; }
    public String getSubdomain() { return subdomain; }
    public String getSecretKey() { return secretKey; }
    public int getConnectTimeout() { return connectTimeout; }
    public int getMaxRetries() { return maxRetries; }
    public int getRetryInterval() { return retryInterval; }
    public String getLogLevel() { return logLevel; }

    public static Builder builder(int localPort) {
        return new Builder(localPort);
    }

    public static class Builder {
        private String serverUrl = "";
        private String serverHost = "localhost";
        private int controlPort = 5555;
        private String localHost = "localhost";
        private final int localPort;
        private String subdomain = "";
        private String secretKey = "";
        private int connectTimeout = 10000;
        private int maxRetries = 5;
        private int retryInterval = 5000;
        private String logLevel = "INFO";

        public Builder(int localPort) {
            this.localPort = localPort;
        }

        public Builder serverUrl(String serverUrl) {
            this.serverUrl = serverUrl;
            return this;
        }

        public Builder serverHost(String serverHost) {
            this.serverHost = serverHost;
            return this;
        }

        public Builder controlPort(int controlPort) {
            this.controlPort = controlPort;
            return this;
        }

        public Builder localHost(String localHost) {
            this.localHost = localHost;
            return this;
        }

        public Builder subdomain(String subdomain) {
            this.subdomain = subdomain;
            return this;
        }

        public Builder secretKey(String secretKey) {
            this.secretKey = secretKey;
            return this;
        }

        public Builder connectTimeout(int connectTimeout) {
            this.connectTimeout = connectTimeout;
            return this;
        }

        public Builder maxRetries(int maxRetries) {
            this.maxRetries = maxRetries;
            return this;
        }

        public Builder retryInterval(int retryInterval) {
            this.retryInterval = retryInterval;
            return this;
        }

        public Builder logLevel(String logLevel) {
            this.logLevel = logLevel;
            return this;
        }

        public TunGoOptions build() {
            return new TunGoOptions(this);
        }
    }
}
