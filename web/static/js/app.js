class GPSSimulatorApp {
    constructor() {
        this.ws = null;
        this.map = null;
        this.serial = null;
        this.isSimulatorRunning = false;
        this.nmeaVisible = true;
        this.nmeaPaused = false;
        this.maxNmeaLines = 100; // Limit NMEA lines to prevent memory issues
        
        this.init();
    }

    init() {
        // Initialize components
        this.initWebSocket();
        this.initMap();
        this.initSerial();
        this.initEventListeners();
        this.updateUI();
    }

    initWebSocket() {
        this.ws = new WebSocketManager();
        
        this.ws.on('connect', () => {
            this.updateConnectionStatus(true);
        });

        this.ws.on('disconnect', () => {
            this.updateConnectionStatus(false);
        });

        this.ws.on('status', (status) => {
            this.handleStatusUpdate(status);
        });

        this.ws.on('nmea_data', (data) => {
            this.handleNMEAData(data);
        });
    }

    initMap() {
        this.map = new MapManager('map');
        
        // Handle window resize for map
        window.addEventListener('resize', () => {
            setTimeout(() => this.map.invalidateSize(), 100);
        });
    }

    initSerial() {
        this.serial = new SerialManager();
        
        // Update serial UI based on support
        if (!this.serial.isWebSerialSupported()) {
            const serialGroup = document.getElementById('serial-group');
            const serialStatus = document.getElementById('serial-status');
            serialStatus.textContent = 'Web Serial API not supported in this browser';
            serialStatus.className = 'error';
            
            // Disable serial button
            document.getElementById('serial-btn').disabled = true;
            document.getElementById('serial-btn').textContent = 'Serial Not Supported';
        }
    }

    initEventListeners() {
        // Simulator controls
        document.getElementById('start-btn').addEventListener('click', () => {
            this.startSimulator();
        });

        document.getElementById('stop-btn').addEventListener('click', () => {
            this.stopSimulator();
        });

        // Configuration
        document.getElementById('update-config-btn').addEventListener('click', () => {
            this.updateConfiguration();
        });

        // Jitter slider
        const jitterSlider = document.getElementById('config-jitter');
        const jitterValue = document.getElementById('jitter-value');
        jitterSlider.addEventListener('input', (e) => {
            jitterValue.textContent = `${Math.round(e.target.value * 100)}%`;
        });

        // Map controls
        document.getElementById('center-map-btn').addEventListener('click', () => {
            this.map.centerOnGPS();
        });

        document.getElementById('auto-center-btn').addEventListener('click', () => {
            const isEnabled = this.map.toggleAutoCenter();
            const button = document.getElementById('auto-center-btn');
            button.textContent = `Auto-Center: ${isEnabled ? 'ON' : 'OFF'}`;
            button.className = `btn ${isEnabled ? 'btn-primary' : 'btn-secondary'}`;
        });

        document.getElementById('clear-track-btn').addEventListener('click', () => {
            this.map.clearTrack();
        });

        // NMEA stream controls
        document.getElementById('toggle-nmea-btn').addEventListener('click', () => {
            this.toggleNmeaStream();
        });

        document.getElementById('pause-nmea-btn').addEventListener('click', () => {
            this.toggleNmeaPause();
        });

        document.getElementById('clear-nmea-btn').addEventListener('click', () => {
            this.clearNmeaStream();
        });

        // Serial controls
        document.getElementById('serial-btn').addEventListener('click', () => {
            this.toggleSerialPanel();
        });

        document.getElementById('serial-connect-btn').addEventListener('click', () => {
            this.connectSerial();
        });

        document.getElementById('serial-disconnect-btn').addEventListener('click', () => {
            this.disconnectSerial();
        });
    }

    updateConnectionStatus(connected) {
        const statusDot = document.getElementById('status-dot');
        const statusText = document.getElementById('status-text');
        
        if (connected) {
            statusDot.className = 'status-dot connected';
            statusText.textContent = 'Connected';
        } else {
            statusDot.className = 'status-dot';
            statusText.textContent = 'Disconnected';
        }
    }

    handleStatusUpdate(status) {
        console.log('Received status update:', status);
        this.isSimulatorRunning = status.running;
        console.log('Set isSimulatorRunning to:', this.isSimulatorRunning);
        this.updateUI();
        
        if (status.position) {
            this.updateGPSDisplay(status.position);
            this.map.updatePosition(status.position);
        }
    }

    handleNMEAData(data) {
        if (data.position) {
            this.updateGPSDisplay(data.position);
            this.map.updatePosition(data.position);
        }

        // Display NMEA sentences if stream is visible and not paused
        if (this.nmeaVisible && !this.nmeaPaused && data.sentences) {
            this.displayNmeaSentences(data.sentences);
        }

        // Send to serial if connected
        if (this.serial && this.serial.isConnected) {
            this.serial.sendNMEAData(data).catch(error => {
                console.error('Error sending NMEA data to serial:', error);
            });
        }
    }

    updateGPSDisplay(position) {
        document.getElementById('lock-status').textContent = 
            position.is_locked ? 'GPS Lock' : 'No Fix';
        document.getElementById('latitude').textContent = 
            position.latitude.toFixed(6) + '°';
        document.getElementById('longitude').textContent = 
            position.longitude.toFixed(6) + '°';
        document.getElementById('altitude').textContent = 
            position.altitude.toFixed(1) + ' m';
        document.getElementById('speed').textContent = 
            position.speed.toFixed(1) + ' kts';
        document.getElementById('course').textContent = 
            position.course.toFixed(1) + '°';
        document.getElementById('satellites').textContent = 
            position.satellites ? position.satellites.length : '0';
    }

    updateUI() {
        const startBtn = document.getElementById('start-btn');
        const stopBtn = document.getElementById('stop-btn');
        const statusDot = document.getElementById('status-dot');
        
        console.log('Updating UI - isSimulatorRunning:', this.isSimulatorRunning);
        
        if (this.isSimulatorRunning) {
            console.log('Enabling stop button, disabling start button');
            startBtn.disabled = true;
            stopBtn.disabled = false;
            statusDot.classList.add('running');
        } else {
            console.log('Enabling start button, disabling stop button');
            startBtn.disabled = false;
            stopBtn.disabled = true;
            statusDot.classList.remove('running');
        }
    }

    getConfiguration() {
        return {
            latitude: parseFloat(document.getElementById('config-lat').value),
            longitude: parseFloat(document.getElementById('config-lon').value),
            speed: parseFloat(document.getElementById('config-speed').value),
            course: parseFloat(document.getElementById('config-course').value),
            jitter: parseFloat(document.getElementById('config-jitter').value),
            radius: parseFloat(document.getElementById('config-radius').value),
            altitude: 45.0,
            altitude_jitter: 0.0,
            satellites: 8,
            time_to_lock: "2s",
            output_rate: "1s",
            baud_rate: 9600,
            quiet: false,
            gpx_enabled: false,
            duration: "0s",
            replay_speed: 1.0,
            replay_loop: false
        };
    }

    async startSimulator() {
        console.log('Start button clicked - calling /api/start');
        try {
            const config = this.getConfiguration();
            const response = await fetch('/api/start', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(config)
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            console.log('Simulator started successfully:', result);
            // Manually set the running state since we know it started
            this.isSimulatorRunning = true;
            this.updateUI();
        } catch (error) {
            console.error('Error starting simulator:', error);
            alert('Error starting simulator: ' + error.message);
        }
    }

    async stopSimulator() {
        console.log('Stop button clicked - calling /api/stop');
        try {
            const response = await fetch('/api/stop', {
                method: 'POST'
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            console.log('Simulator stopped successfully:', result);
            // Manually set the running state since we know it stopped
            this.isSimulatorRunning = false;
            this.updateUI();
        } catch (error) {
            console.error('Error stopping simulator:', error);
            alert('Error stopping simulator: ' + error.message);
        }
    }

    async updateConfiguration() {
        console.log('Update Configuration button clicked - calling /api/config');
        try {
            const config = this.getConfiguration();
            console.log('Sending config:', config);
            const response = await fetch('/api/config', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(config)
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            console.log('Configuration updated successfully:', result);
            
            // Center map on new position
            this.map.setView(config.latitude, config.longitude);
        } catch (error) {
            console.error('Error updating configuration:', error);
            // Don't fall back to starting - this was the problem!
            alert('Error updating configuration: ' + error.message);
        }
    }

    toggleSerialPanel() {
        const serialGroup = document.getElementById('serial-group');
        if (serialGroup.style.display === 'none') {
            serialGroup.style.display = 'block';
        } else {
            serialGroup.style.display = 'none';
        }
    }

    async connectSerial() {
        try {
            await this.serial.requestPort();
            await this.serial.connect(9600);
            
            const serialStatus = document.getElementById('serial-status');
            serialStatus.textContent = 'Connected to Serial Port';
            serialStatus.className = 'connected';
            
            document.getElementById('serial-connect-btn').disabled = true;
            document.getElementById('serial-disconnect-btn').disabled = false;
        } catch (error) {
            console.error('Error connecting to serial port:', error);
            alert('Error connecting to serial port: ' + error.message);
        }
    }

    async disconnectSerial() {
        try {
            await this.serial.disconnect();
            
            const serialStatus = document.getElementById('serial-status');
            serialStatus.textContent = 'Not Connected';
            serialStatus.className = '';
            
            document.getElementById('serial-connect-btn').disabled = false;
            document.getElementById('serial-disconnect-btn').disabled = true;
        } catch (error) {
            console.error('Error disconnecting from serial port:', error);
            alert('Error disconnecting from serial port: ' + error.message);
        }
    }

    toggleNmeaStream() {
        this.nmeaVisible = !this.nmeaVisible;
        const streamDiv = document.getElementById('nmea-stream');
        const toggleBtn = document.getElementById('toggle-nmea-btn');
        
        console.log('Toggling NMEA stream visibility to:', this.nmeaVisible);
        
        if (this.nmeaVisible) {
            streamDiv.style.display = 'block';
            toggleBtn.textContent = 'Hide NMEA';
            toggleBtn.className = 'btn btn-primary';
        } else {
            streamDiv.style.display = 'none';
            toggleBtn.textContent = 'Show NMEA';
            toggleBtn.className = 'btn btn-secondary';
        }
    }

    displayNmeaSentences(sentences) {
        const nmeaContent = document.getElementById('nmea-content');
        const timestamp = new Date().toLocaleTimeString();
        
        sentences.forEach(sentence => {
            // Remove existing \r\n and clean up the sentence
            const cleanSentence = sentence.trim();
            if (cleanSentence) {
                const line = document.createElement('div');
                line.className = 'nmea-sentence';
                line.innerHTML = `<span class="nmea-timestamp">${timestamp}</span>${cleanSentence}`;
                nmeaContent.appendChild(line);
            }
        });

        // Limit the number of lines to prevent memory issues
        const lines = nmeaContent.children;
        while (lines.length > this.maxNmeaLines) {
            nmeaContent.removeChild(lines[0]);
        }

        // Auto-scroll to bottom
        const streamDiv = document.getElementById('nmea-stream');
        streamDiv.scrollTop = streamDiv.scrollHeight;

        // Fade older sentences
        Array.from(lines).forEach((line, index) => {
            if (index < lines.length - 10) {
                line.classList.add('fade');
            } else {
                line.classList.remove('fade');
            }
        });
    }

    clearNmeaStream() {
        console.log('Clearing NMEA stream');
        const nmeaContent = document.getElementById('nmea-content');
        if (nmeaContent) {
            // Clear all content
            nmeaContent.innerHTML = '';
            
            // Add a clear indicator message that will fade away
            const clearMsg = document.createElement('div');
            clearMsg.className = 'nmea-sentence';
            clearMsg.style.color = '#ffaa00';
            clearMsg.style.fontStyle = 'italic';
            clearMsg.innerHTML = `<span class="nmea-timestamp">${new Date().toLocaleTimeString()}</span>--- NMEA Stream Cleared ---`;
            nmeaContent.appendChild(clearMsg);
            
            // Remove the clear message after 2 seconds
            setTimeout(() => {
                if (clearMsg.parentNode) {
                    clearMsg.parentNode.removeChild(clearMsg);
                }
            }, 2000);
            
            console.log('NMEA stream cleared successfully');
        } else {
            console.error('Could not find nmea-content element');
        }
    }

    toggleNmeaPause() {
        this.nmeaPaused = !this.nmeaPaused;
        const pauseBtn = document.getElementById('pause-nmea-btn');
        
        console.log('NMEA stream paused:', this.nmeaPaused);
        
        if (this.nmeaPaused) {
            pauseBtn.textContent = 'Resume';
            pauseBtn.className = 'btn btn-warning';
            
            // Add pause indicator
            const nmeaContent = document.getElementById('nmea-content');
            const pauseMsg = document.createElement('div');
            pauseMsg.id = 'pause-indicator';
            pauseMsg.className = 'nmea-sentence';
            pauseMsg.style.color = '#ffaa00';
            pauseMsg.style.fontStyle = 'italic';
            pauseMsg.innerHTML = `<span class="nmea-timestamp">${new Date().toLocaleTimeString()}</span>--- NMEA Stream Paused ---`;
            nmeaContent.appendChild(pauseMsg);
        } else {
            pauseBtn.textContent = 'Pause';
            pauseBtn.className = 'btn btn-secondary';
            
            // Remove pause indicator
            const pauseIndicator = document.getElementById('pause-indicator');
            if (pauseIndicator && pauseIndicator.parentNode) {
                pauseIndicator.parentNode.removeChild(pauseIndicator);
            }
            
            // Add resume indicator
            const nmeaContent = document.getElementById('nmea-content');
            const resumeMsg = document.createElement('div');
            resumeMsg.className = 'nmea-sentence';
            resumeMsg.style.color = '#00aa00';
            resumeMsg.style.fontStyle = 'italic';
            resumeMsg.innerHTML = `<span class="nmea-timestamp">${new Date().toLocaleTimeString()}</span>--- NMEA Stream Resumed ---`;
            nmeaContent.appendChild(resumeMsg);
            
            // Remove resume message after 2 seconds
            setTimeout(() => {
                if (resumeMsg.parentNode) {
                    resumeMsg.parentNode.removeChild(resumeMsg);
                }
            }, 2000);
        }
    }
}

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new GPSSimulatorApp();
});
