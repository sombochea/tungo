package com.github.sombochea.tungo;

/**
 * Tunnel Information returned after connection
 */
public class TunnelInfo {
    private final String url;
    private final String subdomain;
    private final String clientId;

    public TunnelInfo(String url, String subdomain, String clientId) {
        this.url = url;
        this.subdomain = subdomain;
        this.clientId = clientId;
    }

    public String getUrl() {
        return url;
    }

    public String getSubdomain() {
        return subdomain;
    }

    public String getClientId() {
        return clientId;
    }

    @Override
    public String toString() {
        return "TunnelInfo{" +
                "url='" + url + '\'' +
                ", subdomain='" + subdomain + '\'' +
                ", clientId='" + clientId + '\'' +
                '}';
    }
}
