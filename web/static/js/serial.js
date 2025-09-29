class SerialManager {
    constructor() {
        this.port = null;
        this.writer = null;
        this.isConnected = false;
        this.isSupported = 'serial' in navigator;
        
        // Check if Web Serial API is supported
        if (!this.isSupported) {
            console.warn('Web Serial API is not supported in this browser');
        }
    }

    isWebSerialSupported() {
        return this.isSupported;
    }

    async requestPort() {
        if (!this.isSupported) {
            throw new Error('Web Serial API is not supported');
        }

        try {
            // Request a port and open a connection
            this.port = await navigator.serial.requestPort();
            return true;
        } catch (error) {
            console.error('Error requesting serial port:', error);
            throw error;
        }
    }

    async connect(baudRate = 9600) {
        if (!this.port) {
            throw new Error('No serial port selected');
        }

        try {
            // Open the serial port
            await this.port.open({ 
                baudRate: baudRate,
                dataBits: 8,
                stopBits: 1,
                parity: 'none'
            });

            // Get the writer for sending data
            this.writer = this.port.writable.getWriter();
            this.isConnected = true;

            console.log('Serial port connected successfully');
            return true;
        } catch (error) {
            console.error('Error connecting to serial port:', error);
            throw error;
        }
    }

    async disconnect() {
        try {
            if (this.writer) {
                await this.writer.close();
                this.writer = null;
            }

            if (this.port) {
                await this.port.close();
                this.port = null;
            }

            this.isConnected = false;
            console.log('Serial port disconnected');
            return true;
        } catch (error) {
            console.error('Error disconnecting from serial port:', error);
            throw error;
        }
    }

    async sendNMEAData(nmeaData) {
        if (!this.isConnected || !this.writer) {
            throw new Error('Serial port is not connected');
        }

        try {
            // Convert NMEA sentences to Uint8Array
            const sentences = nmeaData.sentences || [];
            const data = sentences.join('');
            const encoder = new TextEncoder();
            const uint8Array = encoder.encode(data);

            // Send data to serial port
            await this.writer.write(uint8Array);
            return true;
        } catch (error) {
            console.error('Error sending NMEA data to serial port:', error);
            throw error;
        }
    }

    getConnectionStatus() {
        return {
            isConnected: this.isConnected,
            isSupported: this.isSupported,
            portInfo: this.port ? {
                productId: this.port.getInfo().usbProductId,
                vendorId: this.port.getInfo().usbVendorId
            } : null
        };
    }

    // Get available ports (if any were previously authorized)
    async getAvailablePorts() {
        if (!this.isSupported) {
            return [];
        }

        try {
            const ports = await navigator.serial.getPorts();
            return ports;
        } catch (error) {
            console.error('Error getting available ports:', error);
            return [];
        }
    }

    // Listen for port connection/disconnection events
    setupPortEventListeners(onConnect, onDisconnect) {
        if (!this.isSupported) {
            return;
        }

        navigator.serial.addEventListener('connect', (event) => {
            console.log('Serial port connected:', event.target);
            if (onConnect) onConnect(event.target);
        });

        navigator.serial.addEventListener('disconnect', (event) => {
            console.log('Serial port disconnected:', event.target);
            if (onDisconnect) onDisconnect(event.target);
            
            // Clean up if this was our active port
            if (this.port === event.target) {
                this.port = null;
                this.writer = null;
                this.isConnected = false;
            }
        });
    }
}

// Export for use in other modules
window.SerialManager = SerialManager;
