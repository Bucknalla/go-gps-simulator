# GPS NMEA0183 Simulator

A Go-based command-line tool that simulates a GPS receiver outputting NMEA0183 sentences.
Designed for testing GPS-dependent applications, embedded systems development, or educational purposes.

## Features

- **Configurable Initial Position**: Set starting latitude and longitude coordinates
- **Position Wandering**: Simulate GPS drift within a configurable radius
- **GPS Jitter Control**: Adjust how stable or jittery the GPS position readings are
- **Satellite Simulation**: Configure number of visible satellites (4-12)
- **Time-to-Lock Simulation**: Simulate realistic GPS acquisition time
- **Configurable Output Rate**: Control how frequently NMEA sentences are output
- **Serial Port Support**: Output NMEA data directly to serial devices
- **Output Separation**: NMEA data and logging messages are separated (stdout vs stderr)
- **Multiple NMEA Sentence Types**: Supports GGA, RMC, GSA, and GSV sentences
- **Realistic Signal Simulation**: Dynamic satellite positions and signal strength

## Installation

### Download the latest release

Download the appropriate binary for your platform from the [Releases](https://github.com/Bucknalla/go-gps-simulator/releases) page:

- **Linux (x64)**: `go-gps-simulator-linux-amd64`
- **Linux (ARM64)**: `go-gps-simulator-linux-arm64`
- **macOS (Intel)**: `go-gps-simulator-darwin-amd64`
- **macOS (Apple Silicon)**: `go-gps-simulator-darwin-arm64`
- **Windows (x64)**: `go-gps-simulator-windows-amd64.exe`

### Build from Source

#### Prerequisites

- Go 1.23 or later

```bash
git clone https://github.com/Bucknalla/go-gps-simulator.git
cd go-gps-simulator
go build -o gps-simulator
```

## Usage

### Basic Usage

Run the simulator with default settings (San Francisco coordinates):

```bash
gps-simulator
```

### Command Line Options

```bash
gps-simulator [options]
```

| Flag               | Type     | Default   | Description                                              |
| ------------------ | -------- | --------- | -------------------------------------------------------- |
| `-lat`             | float    | 37.7749   | Initial latitude in decimal degrees                      |
| `-lon`             | float    | -122.4194 | Initial longitude in decimal degrees                     |
| `-radius`          | float    | 100.0     | Wandering radius in meters                               |
| `-altitude`        | float    | 45.0      | Starting altitude in meters                              |
| `-jitter`          | float    | 0.5       | GPS position jitter factor (0.0=stable, 1.0=high jitter) |
| `-altitude-jitter` | float    | 0.1       | Altitude jitter factor (0.0=stable, 1.0=high variation)  |
| `-satellites`      | int      | 8         | Number of satellites to simulate (4-12)                  |
| `-lock-time`       | duration | 30s       | Time to GPS lock simulation                              |
| `-rate`            | duration | 1s        | NMEA output rate                                         |
| `-serial`          | string   | ""        | Serial port for NMEA output (e.g., /dev/ttyUSB0, COM1)   |
| `-baud`            | int      | 9600      | Serial port baud rate                                    |
| `-quiet`           | bool     | false     | Suppress informational messages (only output NMEA data)  |

### Examples

#### Simulate GPS in New York City

```bash
gps-simulator -lat 40.7128 -lon -74.0060 -radius 50
```

#### Fast acquisition with many satellites

```bash
gps-simulator -satellites 12 -lock-time 5s -rate 500ms
```

#### Simulate poor GPS conditions

```bash
gps-simulator -satellites 4 -lock-time 2m -radius 200
```

#### Custom location with specific parameters

```bash
gps-simulator -lat 51.5074 -lon -0.1278 -radius 25 -satellites 10 -rate 2s
```

#### GPS Jitter Examples

Stable, smooth positioning (low jitter)

```bash
gps-simulator -jitter 0.1 -radius 50
```

Moderate jitter (default)

```bash
gps-simulator -jitter 0.5
```

High jitter, unstable positioning

```bash
gps-simulator -jitter 0.9 -radius 200
```

#### Altitude Examples

Aircraft altitude simulation

```bash
gps-simulator -altitude 10000 -altitude-jitter 0.2
```

Mountain hiking simulation

```bash
gps-simulator -lat 46.8182 -lon 8.2275 -altitude 2500 -altitude-jitter 0.1
```

Stable sea-level operation

```bash
gps-simulator -altitude 5 -altitude-jitter 0.0
```

#### Serial Port Output Examples

Output to serial port (Linux/macOS)

```bash
gps-simulator -serial /dev/ttyUSB0 -baud 4800
```

Output to serial port (Windows)

```bash
gps-simulator -serial COM3 -baud 9600
```

High-speed serial output

```bash
gps-simulator -serial /dev/ttyUSB0 -baud 115200 -rate 100ms
```

#### Data Separation Examples

Redirect NMEA to file, keep logging on console

```bash
gps-simulator > gps_data.nmea
```

Send NMEA to serial, redirect logging to file

```bash
gps-simulator -serial /dev/ttyUSB0 2> gps_log.txt
```

Pipe NMEA data to another program

```bash
gps-simulator | your_gps_application
```

#### Quiet Mode Examples

Clean NMEA output without informational messages

```bash
gps-simulator -quiet
```

Quiet mode with file output

```bash
gps-simulator -quiet > clean_nmea.txt
```

Quiet mode for piping to applications

```bash
gps-simulator -quiet -rate 100ms | nmea_parser
```

#### Live GPS Stream Viewing

Quick Demo (Everything Automatic)

```bash
scripts/demo_gps.sh                           # Default settings
scripts/demo_gps.sh -jitter 0.8 -rate 500ms  # High jitter, fast updates
scripts/demo_gps.sh -lat 40.7128 -lon -74.0060 # New York City
```

Manual Setup (Two Terminal Windows)

```bash
# Terminal 1: Start GPS viewer (auto-creates virtual ports)
scripts/watch_gps.sh

# Terminal 2: Start GPS simulator
gps-simulator -serial /tmp/gps_out -baud 9600 -rate 1s
```

View Real GPS Device

```bash
scripts/watch_gps.sh /dev/ttyUSB0
```

## NMEA Sentences Generated

The simulator outputs the following NMEA0183 sentence types:

### During GPS Lock Acquisition

- **GGA**: Global Positioning System Fix Data (no fix)
- **RMC**: Recommended Minimum (no fix)

### After GPS Lock

- **GGA**: Global Positioning System Fix Data (with position)
- **RMC**: Recommended Minimum (with position and time)
- **GSA**: GPS DOP and Active Satellites
- **GSV**: GPS Satellites in View (multiple sentences for all satellites)

## Technical Details

### Position Simulation

- Uses Haversine formula for accurate distance calculations
- Jitter-controlled movement patterns:
  - **Low jitter (0.0-0.2)**: Stable, smooth positioning with minimal drift
  - **Medium jitter (0.3-0.7)**: Blend of stable positioning and random movement
  - **High jitter (0.8-1.0)**: Unstable, jittery positioning with large variations
- Automatically keeps positions within the specified radius
- Maintains realistic coordinate precision

### Altitude Simulation

- Configurable starting altitude with realistic variation
- Separate altitude jitter control independent of position jitter:
  - **Low altitude jitter (0.0-0.2)**: Stable altitude with minimal variation
  - **Medium altitude jitter (0.3-0.7)**: Moderate altitude changes simulating aircraft or terrain following
  - **High altitude jitter (0.8-1.0)**: Large altitude variations for testing edge cases
- Automatic bounds checking to prevent unrealistic altitudes
- Dynamic altitude values reflected in NMEA GGA sentences

### Satellite Simulation

- Simulates satellite elevation (5-85 degrees above horizon)
- Generates realistic azimuth values (0-359 degrees)
- Dynamic signal-to-noise ratio (15-55 dB)
- Satellites slowly move over time for realism

### NMEA Compliance

- Proper checksum calculation for all sentences
- Standard NMEA0183 formatting
- Realistic coordinate conversion (DDMM.MMMMM format)
- UTC timestamp generation

## Development

### Helper Scripts

**`demo_gps.sh`** - Complete one-command GPS demo

- Automatically sets up virtual serial ports
- Starts GPS simulator in background
- Shows live colorized NMEA stream
- Cleans up everything on exit
- Supports all GPS simulator options

**`watch_gps.sh`** - Live GPS stream viewer

- Auto-creates virtual serial ports if needed
- Color-coded NMEA sentence types
- Timestamps on all messages
- Works with real serial devices too
- Handles cleanup automatically

## Feature Roadmap

### High Priority Features

- [ ] **Speed & Course Simulation** - Realistic movement speed and direction changes
  - Configurable speed (knots/mph/kmh)
  - Dynamic course calculation based on movement
  - Acceleration/deceleration patterns
- [ ] **Altitude Variation** - Dynamic altitude simulation with terrain following
  - Configurable starting altitude
  - Altitude jitter and gradual changes
  - Support for meters/feet units
- [ ] **Additional NMEA Sentences** - Expand sentence type support
  - VTG (Course and Speed over Ground)
  - GLL (Geographic Position - Latitude/Longitude)
  - ZDA (Time and Date)
- [ ] **Signal Quality Scenarios** - Simulate real-world GPS challenges
  - Urban canyon effects (reduced satellite visibility)
  - Tunnel/building signal loss
  - Gradual signal degradation
  - Weather-based interference

### Medium Priority Features

- [ ] **Configuration Files** - YAML/JSON configuration support
  - Preset scenarios (urban, rural, marine, aviation)
  - Equipment profiles for different GPS receivers
  - Batch testing configurations
- [ ] **Multi-Constellation Support** - Beyond GPS satellites
  - GLONASS, Galileo, BeiDou satellites
  - Constellation-specific NMEA sentences (GLGGA, GNGGA, etc.)
  - Different satellite ID ranges
- [ ] **Predefined Routes** - Follow realistic movement patterns
  - GPX file support for route following
  - City/highway driving patterns
  - Airport runway procedures
  - Maritime shipping lanes
- [ ] **Performance Metrics** - Realistic GPS quality indicators
  - Accurate HDOP/VDOP/PDOP calculations
  - Signal strength variations
  - Fix quality statistics

### Advanced Features

- [ ] **DGPS/RTK Simulation** - High-precision GPS modes
  - Differential GPS corrections
  - RTK fixed/float states
  - Base station simulation
- [ ] **Replay & Recording** - Capture and playback GPS sessions
  - Record real GPS data
  - Replay NMEA logs with speed control
  - Loop recordings for continuous testing
- [ ] **Multiple Receiver Simulation** - Fleet/multi-device scenarios
  - Simulate multiple GPS units simultaneously
  - Independent or synchronized operation
  - Different output ports per receiver
- [ ] **Interactive Control** - Real-time parameter adjustment
  - Keyboard controls for movement
  - Web-based control interface
  - Real-time parameter modification

### Visualization & Analysis

- [ ] **Real-time Visualization** - Map and satellite displays
  - Live position tracking on map
  - Satellite constellation view
  - Signal strength indicators
- [ ] **Fault Injection** - Test edge cases and failures
  - GPS jamming simulation
  - Spoofing scenarios
  - Satellite outages
  - Clock drift simulation

## Contributing

Contributions are welcome! Please feel free to submit issues, feature requests, or pull requests.
