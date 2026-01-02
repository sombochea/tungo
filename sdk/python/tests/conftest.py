"""Pytest configuration and fixtures."""

import pytest


@pytest.fixture
def server_url():
    """Default test server URL."""
    return "ws://localhost:5555/ws"


@pytest.fixture
def local_port():
    """Default local port for testing."""
    return 8000


@pytest.fixture
def retry_config():
    """Default retry configuration for tests."""
    return {
        "max_retries": 3,
        "retry_interval": 2.0,
    }
