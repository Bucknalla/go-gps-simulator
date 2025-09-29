class MapManager {
    constructor(containerId) {
        this.containerId = containerId;
        this.map = null;
        this.gpsMarker = null;
        this.trackPolyline = null;
        this.trackPoints = [];
        this.maxTrackPoints = 1000; // Limit track points to prevent performance issues
        this.autoCenter = true; // Automatically center map on GPS position
        this.lastCenterTime = 0; // Track when we last centered to avoid too frequent updates
        this.init();
    }

    init() {
        // Initialize Leaflet map
        this.map = L.map(this.containerId).setView([37.7749, -122.4194], 13);

        // Add OpenStreetMap tiles
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
            maxZoom: 19
        }).addTo(this.map);

        // Add scale control
        L.control.scale().addTo(this.map);

        // Create custom GPS marker icon
        this.createGPSIcon();
    }

    createGPSIcon() {
        // Create a custom GPS marker icon
        this.gpsIcon = L.divIcon({
            className: 'gps-marker',
            html: '<div style="background: #007bff; width: 12px; height: 12px; border-radius: 50%; border: 2px solid white; box-shadow: 0 2px 4px rgba(0,0,0,0.3);"></div>',
            iconSize: [16, 16],
            iconAnchor: [8, 8]
        });

        // Create direction arrow icon (for when GPS is moving)
        this.gpsArrowIcon = L.divIcon({
            className: 'gps-arrow',
            html: '<div style="color: #007bff; font-size: 16px; text-shadow: 1px 1px 2px rgba(255,255,255,0.8);">▲</div>',
            iconSize: [16, 16],
            iconAnchor: [8, 8]
        });
    }

    updatePosition(position) {
        const { latitude, longitude, altitude, speed, course, is_locked, timestamp } = position;

        if (!is_locked) {
            // Remove marker if GPS is not locked
            if (this.gpsMarker) {
                this.map.removeLayer(this.gpsMarker);
                this.gpsMarker = null;
            }
            return;
        }

        const latLng = [latitude, longitude];

        // Create or update GPS marker
        if (this.gpsMarker) {
            this.gpsMarker.setLatLng(latLng);
        } else {
            // Choose icon based on speed
            const icon = speed > 0.1 ? this.gpsArrowIcon : this.gpsIcon;
            
            this.gpsMarker = L.marker(latLng, { icon })
                .addTo(this.map)
                .bindPopup(this.createPopupContent(position));
        }

        // Rotate arrow icon based on course if moving
        if (speed > 0.1 && this.gpsMarker) {
            const element = this.gpsMarker.getElement();
            if (element) {
                const arrow = element.querySelector('div');
                if (arrow) {
                    arrow.style.transform = `rotate(${course}deg)`;
                }
            }
        }

        // Update popup content
        if (this.gpsMarker) {
            this.gpsMarker.setPopupContent(this.createPopupContent(position));
        }

        // Add point to track
        this.addTrackPoint(latLng);

        // Auto-center map on GPS position (with throttling to avoid too frequent updates)
        if (this.autoCenter) {
            const now = Date.now();
            if (now - this.lastCenterTime > 2000) { // Only center every 2 seconds max
                this.map.setView(latLng, this.map.getZoom(), { animate: true, duration: 1 });
                this.lastCenterTime = now;
            }
        }
    }

    createPopupContent(position) {
        const { latitude, longitude, altitude, speed, course, satellites, timestamp } = position;
        
        return `
            <div style="font-family: monospace; font-size: 12px;">
                <strong>GPS Position</strong><br>
                <strong>Lat:</strong> ${latitude.toFixed(6)}°<br>
                <strong>Lon:</strong> ${longitude.toFixed(6)}°<br>
                <strong>Alt:</strong> ${altitude.toFixed(1)} m<br>
                <strong>Speed:</strong> ${speed.toFixed(1)} kts<br>
                <strong>Course:</strong> ${course.toFixed(1)}°<br>
                <strong>Sats:</strong> ${satellites ? satellites.length : 0}<br>
                <strong>Time:</strong> ${new Date(timestamp).toLocaleTimeString()}
            </div>
        `;
    }

    addTrackPoint(latLng) {
        // Add new point to track
        this.trackPoints.push(latLng);

        // Limit track points to prevent performance issues
        if (this.trackPoints.length > this.maxTrackPoints) {
            this.trackPoints.shift();
        }

        // Update or create track polyline
        if (this.trackPolyline) {
            this.trackPolyline.setLatLngs(this.trackPoints);
        } else if (this.trackPoints.length > 1) {
            this.trackPolyline = L.polyline(this.trackPoints, {
                color: '#007bff',
                weight: 3,
                opacity: 0.8
            }).addTo(this.map);
        }
    }

    centerOnGPS() {
        if (this.gpsMarker) {
            const latLng = this.gpsMarker.getLatLng();
            this.map.setView(latLng, this.map.getZoom(), { animate: true, duration: 1 });
            this.lastCenterTime = Date.now(); // Update last center time
        }
    }

    toggleAutoCenter() {
        this.autoCenter = !this.autoCenter;
        console.log('Auto-center:', this.autoCenter ? 'enabled' : 'disabled');
        return this.autoCenter;
    }

    setAutoCenter(enabled) {
        this.autoCenter = enabled;
        console.log('Auto-center:', this.autoCenter ? 'enabled' : 'disabled');
    }

    clearTrack() {
        // Remove track polyline
        if (this.trackPolyline) {
            this.map.removeLayer(this.trackPolyline);
            this.trackPolyline = null;
        }
        
        // Clear track points array
        this.trackPoints = [];
    }

    setView(lat, lon, zoom = 13) {
        this.map.setView([lat, lon], zoom);
    }

    getMap() {
        return this.map;
    }

    // Method to handle window resize
    invalidateSize() {
        if (this.map) {
            this.map.invalidateSize();
        }
    }
}

// Export for use in other modules
window.MapManager = MapManager;
