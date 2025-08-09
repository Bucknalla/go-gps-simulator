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
- **Multiple NMEA Sentence Types**: Supports GGA, RMC, GLL, VTG, GSA, GSV, and ZDA sentences
- **Speed & Course Simulation**: Configurable static speed and course values in NMEA output
- **Realistic Signal Simulation**: Dynamic satellite positions and signal strength
- **GPX Track Generation**: Export GPS tracks to GPX files for analysis and visualization
- **GPX Track Replay**: Replay existing GPX files with configurable speed multipliers
- **Duration Control**: Automatic simulation termination after specified time periods

## Installation

### Install with Homebrew

```bash
brew tap Bucknalla/tools
brew install gps-simulator
```

### Download the latest release

Download the appropriate binary for your platform from the [Releases](https://github.com/Bucknalla/go-gps-simulator/releases) page

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
| `-altitude-jitter` | float    | 0.0       | Altitude jitter factor (0.0=stable, 1.0=high variation)  |
| `-speed`           | float    | 0.0       | Static speed in knots                                    |
| `-course`          | float    | 0.0       | Static course in degrees (0-359)                        |
| `-satellites`      | int      | 8         | Number of satellites to simulate (4-12)                  |
| `-lock-time`       | duration | 2s        | Time to GPS lock simulation                              |
| `-rate`            | duration | 1s        | NMEA output rate                                         |
| `-serial`          | string   | ""        | Serial port for NMEA output (e.g., /dev/ttyUSB0, COM1)   |
| `-baud`            | int      | 9600      | Serial port baud rate                                    |
| `-quiet`           | bool     | false     | Suppress informational messages (only output NMEA data)  |
| `-gpx`             | bool     | false     | Generate GPX track file with timestamp-based filename    |
| `-duration`        | duration | 0         | How long to run the simulation (e.g., 30s, 5m, 1h)      |
| `-replay`          | string   | ""        | GPX file to replay instead of simulating (e.g., track.gpx) |
| `-replay-speed`    | float    | 1.0       | Replay speed multiplier (1.0=real-time, 2.0=2x speed, 0.5=half speed) |
| `-replay-loop`     | bool     | false     | Loop the GPX replay continuously (default: stop after one pass) |

**Note**: When using `-gpx`, the `-duration` flag is required.

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
gps-simulator -lat 51.5074 -lon -0.1278 -radius 25 -satellites 10 -rate 2s -speed 5.0 -course 180.0
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

#### Speed and Course Examples

Simulate a vessel moving east at 10 knots

```bash
gps-simulator -speed 10.0 -course 90.0
```

Aircraft simulation at 250 knots heading northwest

```bash
gps-simulator -speed 250.0 -course 315.0 -altitude 10000
```

Slow pedestrian movement northward

```bash
gps-simulator -speed 3.0 -course 0.0 -radius 25
```

Stationary GPS receiver

```bash
gps-simulator -speed 0.0 -course 0.0 -radius 5
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

#### GPX Track Generation Examples

Generate GPX track with automatic timestamp filename

```bash
gps-simulator -gpx -duration 5m -lock-time 10s
```

Boat journey simulation with GPX export

```bash
gps-simulator -gpx -lat 37.8080 -lon -122.4177 -radius 5000 -speed 8.5 -course 45 -jitter 0.3 -altitude 2.0 -duration 60s
```

Aircraft flight path with GPX

```bash
gps-simulator -gpx -lat 40.7128 -lon -74.0060 -altitude 10000 -speed 250 -course 90 -duration 30m -rate 5s
```

Hiking trail simulation

```bash
gps-simulator -gpx -lat 46.8182 -lon 8.2275 -altitude 2500 -speed 3.0 -radius 100 -duration 2h -rate 30s
```

#### Duration Control Examples

Short test run (30 seconds)

```bash
gps-simulator -duration 30s -rate 1s
```

Extended simulation (2 hours)

```bash
gps-simulator -duration 2h -rate 10s
```

Quick GPS lock test (5 minutes)

```bash
gps-simulator -duration 5m -lock-time 5s -rate 500ms
```

Automated testing scenario

```bash
gps-simulator -gpx -duration 10m -quiet -lat 51.5074 -lon -0.1278 -speed 5.0
```

#### GPX Replay Examples

Replay a GPX track once at real-time speed (default behavior)

```bash
gps-simulator -replay my_track.gpx
```

Replay continuously in a loop

```bash
gps-simulator -replay my_track.gpx -replay-loop
```

Replay at 2x speed for faster testing

```bash
gps-simulator -replay my_track.gpx -replay-speed 2.0
```

Slow motion replay at half speed

```bash
gps-simulator -replay my_track.gpx -replay-speed 0.5
```

Continuous loop at high speed for stress testing

```bash
gps-simulator -replay track.gpx -replay-loop -replay-speed 10.0
```

Replay with custom output settings

```bash
gps-simulator -replay journey.gpx -rate 500ms -serial /dev/ttyUSB0 -baud 4800
```

Quiet replay for piping to applications

```bash
gps-simulator -replay track.gpx -quiet -replay-speed 5.0 | nmea_parser
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
- **GLL**: Geographic Position - Latitude/Longitude (no fix)
- **VTG**: Track Made Good and Ground Speed (no fix)

### After GPS Lock

- **GGA**: Global Positioning System Fix Data (with position)
- **RMC**: Recommended Minimum (with position and time)
- **GLL**: Geographic Position - Latitude/Longitude (with position)
- **VTG**: Track Made Good and Ground Speed (with speed/course)
- **GSA**: GPS DOP and Active Satellites
- **GSV**: GPS Satellites in View (multiple sentences for all satellites)
- **ZDA**: UTC Date and Time (with precise time and date)

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

### Speed and Course Simulation

- **Static Speed Configuration**: Set constant speed in knots for consistent simulation
- **Static Course Configuration**: Set heading in degrees (0-359) for directional simulation
- **NMEA Integration**: Speed and course values are properly formatted in RMC sentences
- **Realistic Values**: Supports speeds from 0 (stationary) to high-speed scenarios (aircraft, vessels)
- **Course Precision**: Full 360-degree range with decimal precision for accurate heading simulation

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

### GPX Track Generation

- **Standard GPX 1.1 Format**: Industry-standard XML format compatible with most GPS applications
- **Automatic Filename Generation**: Creates timestamped files (YYYYMMDD_HHMMSS.gpx) when using `-gpx` flag
- **Complete Track Data**: Includes latitude, longitude, elevation, and UTC timestamps for each point
- **Real-time Writing**: Track points are written during simulation for data safety
- **Post-GPS Lock**: Only records track points after GPS lock is achieved for accurate data
- **Duration Required**: The `-duration` flag must be specified when using `-gpx` to ensure controlled file size

### GPX Track Replay

- **Standard GPX 1.1 Support**: Reads industry-standard GPX files from any GPS application or device
- **Automatic Speed/Course Calculation**: Calculates realistic speed and course values from track point timestamps and positions
- **Configurable Replay Speed**: Speed multipliers from 0.1x (slow motion) to 10x+ (fast forward) for testing scenarios
- **Seamless NMEA Integration**: Replayed positions generate the same NMEA sentences as simulated data
- **Single Pass Default**: By default, stops after completing one pass through the track points
- **Optional Loop Functionality**: Use `-replay-loop` flag to continuously restart from the beginning when reaching the end
- **Time-Based Progression**: Respects original GPX timestamps for accurate replay timing
- **Automatic Completion**: Shows "GPX replay completed" message when finishing a single pass

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

- [x] **Speed & Course Simulation** - Static speed and course configuration
  - Configurable speed in knots via `-speed` flag
  - Configurable course in degrees (0-359) via `-course` flag
  - Properly integrated into RMC NMEA sentences
- [x] **GPX Track Generation** - Export GPS simulation data to GPX files
  - Standard GPX 1.1 XML format with complete track data
  - Automatic timestamp-based filename generation
  - Real-time track point recording during simulation
  - Compatible with GPS analysis and mapping applications
- [x] **Duration Control** - Automatic simulation termination
  - Configurable simulation duration via `-duration` flag
  - Supports various time formats (30s, 5m, 1h, etc.)
  - Clean shutdown with proper GPX file closure
- [ ] **Additional NMEA Sentences** - Expand sentence type support
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
- [x] **GPX Track Replay** - Replay existing GPX files with NMEA output
  - Standard GPX 1.1 file format support
  - Configurable replay speed multipliers (0.1x to 10x+)
  - Automatic speed and course calculation from track data
  - Seamless integration with all existing NMEA output features
- [ ] **Predefined Routes** - Follow realistic movement patterns
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
