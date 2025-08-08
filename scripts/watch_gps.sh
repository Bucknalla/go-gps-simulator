#!/bin/bash

# GPS Live Viewer Script with Virtual Serial Port Setup
# Usage: ./watch_gps.sh [serial_device]
#
# If no serial device is specified, creates virtual serial ports:
# - /tmp/gps_out (for simulator to write to)
# - /tmp/gps_in (for this script to read from)

SERIAL_DEVICE=${1:-/tmp/gps_in}
USE_VIRTUAL_PORTS=false

# Check if we should set up virtual serial ports
if [[ "$SERIAL_DEVICE" == "/tmp/gps_in" && ! -e "/tmp/gps_in" ]]; then
    USE_VIRTUAL_PORTS=true
fi

echo "=== GPS NMEA Live Viewer ==="

# Setup virtual serial ports if needed
if [[ "$USE_VIRTUAL_PORTS" == "true" ]]; then
    echo "Setting up virtual serial ports..."

    # Check if socat is available
    if ! command -v socat &> /dev/null; then
        echo "Error: socat is required but not installed."
        echo "Install socat: brew install socat (macOS) or apt-get install socat (Linux)"
        exit 1
    fi

    # Kill any existing socat processes for these ports
    pkill -f "socat.*gps_out.*gps_in" 2>/dev/null

    # Clean up any existing symlinks
    rm -f /tmp/gps_out /tmp/gps_in 2>/dev/null

    # Start socat in background
    echo "Creating virtual serial port pair..."
    socat -d -d pty,raw,echo=0,link=/tmp/gps_out pty,raw,echo=0,link=/tmp/gps_in &
    SOCAT_PID=$!

    # Wait for socat to create the devices
    sleep 2

    if [[ ! -e "/tmp/gps_out" || ! -e "/tmp/gps_in" ]]; then
        echo "Error: Failed to create virtual serial ports"
        kill $SOCAT_PID 2>/dev/null
        exit 1
    fi

    echo "✓ Virtual serial ports created:"
    echo "  GPS Simulator output: /tmp/gps_out"
    echo "  GPS Viewer input: /tmp/gps_in"
    echo ""
    echo "To start GPS simulator in another terminal:"
    echo "  ./gps-simulator -serial /tmp/gps_out -baud 9600 -rate 1s"
    echo ""

    # Setup cleanup on exit
    trap 'echo ""; echo "Cleaning up virtual serial ports..."; kill $SOCAT_PID 2>/dev/null; rm -f /tmp/gps_out /tmp/gps_in 2>/dev/null; exit' INT TERM
fi

echo "Reading from: $SERIAL_DEVICE"
echo "Press Ctrl+C to stop"
echo "=========================="
echo ""

# Function to add timestamps and color coding
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

# Start reading from the serial device
if [ -e "$SERIAL_DEVICE" ]; then
    cat "$SERIAL_DEVICE" | format_nmea
else
    echo "Error: Serial device $SERIAL_DEVICE not found!"
    echo "Make sure to run the GPS simulator first or check the device path."
    exit 1
fi
