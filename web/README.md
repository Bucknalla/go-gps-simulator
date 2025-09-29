# GPS Simulator Web Frontend

A web-based interface for the GPS simulator that provides real-time visualization and serial device connectivity.

## Project Checklist

### Core Features
- [ ] **Display a fullscreen map using Leaflet.js**
  - [ ] Initialize Leaflet map with OpenStreetMap tiles
  - [ ] Center map on simulator's current position
  - [ ] Add real-time GPS position marker
  - [ ] Show GPS track/trail as simulator moves
  - [ ] Responsive fullscreen layout
  - [ ] Map controls (zoom, pan, layer selection)

- [ ] **Use Web Serial API to connect physical serial devices**
  - [ ] Implement Web Serial API integration
  - [ ] Serial port selection and connection UI
  - [ ] Stream NMEA0183 data to connected serial devices
  - [ ] Handle serial port connection/disconnection
  - [ ] Error handling and user feedback
  - [ ] Serial port configuration (baud rate, etc.)

- [ ] **Support multiple clients simultaneously**
  - [ ] WebSocket server for real-time data streaming
  - [ ] Multiple client connection management
  - [ ] Broadcast GPS data to all connected clients
  - [ ] Client connection status tracking
  - [ ] Graceful handling of client disconnections

### Technical Architecture

#### Backend Components
- [ ] **WebSocket Server**
  - [ ] Go WebSocket server integration
  - [ ] NMEA data broadcasting to web clients
  - [ ] Client connection management
  - [ ] Integration with existing GPS simulator

- [ ] **HTTP Server**
  - [ ] Serve static web assets
  - [ ] API endpoints for configuration
  - [ ] Health check endpoints

#### Frontend Components
- [ ] **Map Interface**
  - [ ] Leaflet.js integration
  - [ ] Real-time position updates
  - [ ] GPS track visualization
  - [ ] Map layer controls

- [ ] **Serial Interface**
  - [ ] Web Serial API implementation
  - [ ] Port selection UI
  - [ ] Connection status display
  - [ ] Data streaming controls

- [ ] **Control Panel**
  - [ ] Simulator configuration controls
  - [ ] Connection status display
  - [ ] Real-time GPS data display (lat/lon/alt/speed/course)

### Development Phases

#### Phase 1: Basic Web Server and Map
- [ ] Set up Go HTTP server
- [ ] Create basic HTML/CSS/JS structure
- [ ] Implement Leaflet.js map
- [ ] Add WebSocket connection for GPS data
- [ ] Display real-time position on map

#### Phase 2: Web Serial Integration
- [ ] Implement Web Serial API
- [ ] Add serial port selection UI
- [ ] Stream NMEA data to serial devices
- [ ] Add error handling and user feedback

#### Phase 3: Multi-client Support
- [ ] Enhance WebSocket server for multiple clients
- [ ] Add client management
- [ ] Implement broadcast functionality
- [ ] Add connection status monitoring

#### Phase 4: Enhanced Features
- [ ] Add GPS data panels (speed, course, altitude, etc.)
- [ ] Implement map controls and settings
- [ ] Add track recording and playback visualization
- [ ] Performance optimizations

### File Structure
```
web/
├── README.md                 # This file
├── server/
│   ├── main.go              # Web server main
│   ├── websocket.go         # WebSocket handler
│   └── handlers.go          # HTTP handlers
├── static/
│   ├── index.html           # Main HTML page
│   ├── css/
│   │   └── styles.css       # Styles
│   ├── js/
│   │   ├── app.js           # Main application logic
│   │   ├── map.js           # Map handling
│   │   ├── serial.js        # Web Serial API
│   │   └── websocket.js     # WebSocket client
│   └── assets/
│       └── icons/           # Map markers and icons
└── package.json             # Frontend dependencies (if using npm)
```

### Browser Requirements
- Chrome/Edge 89+ (Web Serial API support)
- Firefox 90+ (with WebSocket support, Serial API via polyfill)
- Safari 14+ (WebSocket support, Serial API not supported)

### Security Considerations
- Web Serial API requires HTTPS in production
- CORS configuration for cross-origin requests
- Input validation for serial data
- Rate limiting for WebSocket connections

## Getting Started

### Prerequisites
- Go 1.21 or later
- Modern web browser with WebSocket support
- For serial functionality: Chrome/Edge 89+ (Web Serial API support)

### Installation

1. **Install dependencies** (from web directory):
   ```bash
   cd web
   go mod tidy
   ```

2. **Start the web server** (from web directory):
   ```bash
   go run server/main.go
   ```

3. **Open browser** and navigate to `http://localhost:8080`

4. **Configure and start the simulator** using the web interface

5. **Connect serial device** (if needed) using the Web Serial interface

### Architecture Changes

The GPS simulator has been refactored into a library-based architecture:

- **`gps/` package**: Core GPS simulator library
  - `simulator.go`: Main simulator with WebSocket callbacks
  - `config.go`: Configuration management with validation
  - `types.go`: Data structures for Position, Status, NMEAData
  - `nmea.go`: NMEA sentence generation
  - `gpx.go`: GPX file handling
  - `replay.go`: GPX replay functionality
  - `errors.go`: Error definitions

- **`web/server/` package**: Web server implementation
  - HTTP API endpoints for simulator control
  - WebSocket server for real-time data streaming
  - Multi-client support with broadcast functionality
  - RESTful configuration management

### API Endpoints

- `POST /api/start` - Start simulator with configuration
- `POST /api/stop` - Stop simulator
- `GET /api/status` - Get current simulator status
- `POST /api/config` - Update simulator configuration
- `GET /api/ws` - WebSocket connection for real-time data

## Development Notes

- The Web Serial API is experimental and requires HTTPS for production use
- Consider fallback options for browsers without Web Serial support
- WebSocket connections should be resilient to network interruptions
- Map performance may need optimization for long GPS tracks
