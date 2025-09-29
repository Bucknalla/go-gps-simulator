class WebSocketManager {
    constructor() {
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        this.callbacks = {
            'status': [],
            'nmea_data': [],
            'connect': [],
            'disconnect': [],
            'error': []
        };
        this.connect();
    }

    connect() {
        try {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/api/ws`;
            
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = (event) => {
                console.log('WebSocket connected');
                this.reconnectAttempts = 0;
                this.trigger('connect', event);
            };

            this.ws.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    this.trigger(message.type, message.data);
                } catch (error) {
                    console.error('Error parsing WebSocket message:', error);
                }
            };

            this.ws.onclose = (event) => {
                console.log('WebSocket disconnected');
                this.trigger('disconnect', event);
                this.attemptReconnect();
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.trigger('error', error);
            };

        } catch (error) {
            console.error('Failed to create WebSocket connection:', error);
            this.attemptReconnect();
        }
    }

    attemptReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
            
            setTimeout(() => {
                this.connect();
            }, this.reconnectDelay * this.reconnectAttempts);
        } else {
            console.error('Max reconnection attempts reached');
        }
    }

    send(data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(data));
        } else {
            console.warn('WebSocket is not connected');
        }
    }

    on(event, callback) {
        if (this.callbacks[event]) {
            this.callbacks[event].push(callback);
        }
    }

    off(event, callback) {
        if (this.callbacks[event]) {
            const index = this.callbacks[event].indexOf(callback);
            if (index > -1) {
                this.callbacks[event].splice(index, 1);
            }
        }
    }

    trigger(event, data) {
        if (this.callbacks[event]) {
            this.callbacks[event].forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error(`Error in ${event} callback:`, error);
                }
            });
        }
    }

    close() {
        if (this.ws) {
            this.ws.close();
        }
    }

    isConnected() {
        return this.ws && this.ws.readyState === WebSocket.OPEN;
    }
}

// Export for use in other modules
window.WebSocketManager = WebSocketManager;
