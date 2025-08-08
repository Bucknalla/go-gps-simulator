#!/bin/bash

# GPS Demo Script - Complete Setup
# This script sets up virtual serial ports and starts both the viewer and simulator

echo "=== GPS Simulator Demo Setup ==="
echo ""

# Check if socat is available
if ! command -v socat &> /dev/null; then
    echo "Error: socat is required but not installed."
    echo "Install socat:"
    echo "  macOS: brew install socat"
    echo "  Linux: apt-get install socat"
    exit 1
fi

# Check if GPS simulator exists
if [[ ! -f "./gps-simulator" ]]; then
    echo "Error: GPS simulator not found. Please build it first:"
    echo "  go build -o gps-simulator"
    exit 1
fi

# Clean up any existing processes and ports
echo "Cleaning up existing processes..."
pkill -f "socat.*gps_out.*gps_in" 2>/dev/null
pkill -f "gps-simulator.*serial" 2>/dev/null
rm -f /tmp/gps_out /tmp/gps_in 2>/dev/null

# Setup virtual serial ports
echo "Setting up virtual serial ports..."
socat -d -d pty,raw,echo=0,link=/tmp/gps_out pty,raw,echo=0,link=/tmp/gps_in &
SOCAT_PID=$!

# Wait for ports to be ready
sleep 2

if [[ ! -e "/tmp/gps_out" || ! -e "/tmp/gps_in" ]]; then
    echo "Error: Failed to create virtual serial ports"
    kill $SOCAT_PID 2>/dev/null
    exit 1
fi

echo "✓ Virtual serial ports ready"
echo ""

# Parse command line arguments for GPS simulator
GPS_ARGS=""
while [[ $# -gt 0 ]]; do
    case $1 in
        -lat|--latitude)
            GPS_ARGS="$GPS_ARGS -lat $2"
            shift 2
            ;;
        -lon|--longitude)
            GPS_ARGS="$GPS_ARGS -lon $2"
            shift 2
            ;;
        -radius|--radius)
            GPS_ARGS="$GPS_ARGS -radius $2"
            shift 2
            ;;
        -jitter|--jitter)
            GPS_ARGS="$GPS_ARGS -jitter $2"
            shift 2
            ;;
        -satellites|--satellites)
            GPS_ARGS="$GPS_ARGS -satellites $2"
            shift 2
            ;;
        -lock-time|--lock-time)
            GPS_ARGS="$GPS_ARGS -lock-time $2"
            shift 2
            ;;
        -rate|--rate)
            GPS_ARGS="$GPS_ARGS -rate $2"
            shift 2
            ;;
        -baud|--baud)
            GPS_ARGS="$GPS_ARGS -baud $2"
            shift 2
            ;;
        -h|--help)
            echo "GPS Demo Script Usage:"
            echo "  $0 [GPS simulator options]"
            echo ""
            echo "Examples:"
            echo "  $0                                    # Default settings"
            echo "  $0 -lat 40.7128 -lon -74.0060       # New York City"
            echo "  $0 -jitter 0.8 -rate 500ms          # High jitter, fast updates"
            echo "  $0 -satellites 12 -lock-time 5s     # Many satellites, slow lock"
            echo ""
            echo "This script will:"
            echo "  1. Set up virtual serial ports"
            echo "  2. Start GPS simulator in background"
            echo "  3. Show live NMEA stream with colors and timestamps"
            echo "  4. Clean up everything when you press Ctrl+C"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h for help"
            exit 1
            ;;
    esac
done

# Default GPS settings if none provided
if [[ -z "$GPS_ARGS" ]]; then
    GPS_ARGS="-lat 37.7749 -lon -122.4194 -radius 100 -jitter 0.3 -satellites 8 -lock-time 3s -rate 1s -baud 9600"
fi

echo "Starting GPS simulator with: $GPS_ARGS"
echo "GPS output port: /tmp/gps_out"
echo "GPS input port: /tmp/gps_in"
echo ""

# Start GPS simulator in background
./gps-simulator -serial /tmp/gps_out $GPS_ARGS 2>/dev/null &
GPS_PID=$!

# Give simulator a moment to start
sleep 1

echo "=== Live GPS Stream (Press Ctrl+C to stop) ==="
echo ""

# Setup cleanup function
cleanup() {
    echo ""
    echo "Shutting down GPS demo..."
    kill $GPS_PID 2>/dev/null
    kill $SOCAT_PID 2>/dev/null
    rm -f /tmp/gps_out /tmp/gps_in 2>/dev/null
    echo "Cleanup complete."
    exit 0
}

# Setup signal handlers
trap cleanup INT TERM

# Function to format NMEA output with colors and timestamps
format_nmea() {
    while IFS= read -r line; do
        timestamp=$(date '+%H:%M:%S')

        # Color code different NMEA sentence types
        if [[ $line == \$GPGGA* ]]; then
            echo -e "\033[32m[$timestamp] $line\033[0m"  # Green for GGA (position)
        elif [[ $line == \$GPRMC* ]]; then
            echo -e "\033[34m[$timestamp] $line\033[0m"  # Blue for RMC (recommended minimum)
        elif [[ $line == \$GPGSA* ]]; then
            echo -e "\033[33m[$timestamp] $line\033[0m"  # Yellow for GSA (satellite status)
        elif [[ $line == \$GPGSV* ]]; then
            echo -e "\033[36m[$timestamp] $line\033[0m"  # Cyan for GSV (satellites in view)
        else
            echo "[$timestamp] $line"
        fi
    done
}

# Start reading from the GPS stream
if [[ -e "/tmp/gps_in" ]]; then
    cat /tmp/gps_in | format_nmea
else
    echo "Error: GPS input port not available"
    cleanup
fi
