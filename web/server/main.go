package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Bucknalla/go-gps-simulator/gps"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type WebServer struct {
	simulator  *gps.Simulator
	lastConfig gps.Config
	upgrader   websocket.Upgrader
	clients    map[*websocket.Conn]bool
	broadcast  chan gps.NMEAData
}

func NewWebServer() *WebServer {
	return &WebServer{
		lastConfig: gps.DefaultConfig(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan gps.NMEAData),
	}
}

func (ws *WebServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Register client
	ws.clients[conn] = true
	log.Printf("Client connected. Total clients: %d", len(ws.clients))

	// Send current status immediately
	if ws.simulator != nil {
		status := ws.simulator.GetStatus()
		if err := conn.WriteJSON(map[string]interface{}{
			"type": "status",
			"data": status,
		}); err != nil {
			log.Printf("Error sending status: %v", err)
		}
	}

	// Listen for messages from client
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		// Handle client messages (could be used for configuration updates)
		log.Printf("Received message: %v", msg)
	}

	// Unregister client
	delete(ws.clients, conn)
	log.Printf("Client disconnected. Total clients: %d", len(ws.clients))
}

func (ws *WebServer) broadcastToClients() {
	for {
		nmeaData := <-ws.broadcast

		message := map[string]interface{}{
			"type": "nmea_data",
			"data": nmeaData,
		}

		// Send to all connected clients
		for client := range ws.clients {
			err := client.WriteJSON(message)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				client.Close()
				delete(ws.clients, client)
			}
		}
	}
}

func (ws *WebServer) handleStartSimulator(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON config from request
	var jsonConfig map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&jsonConfig); err != nil {
		log.Printf("JSON decode error: %v", err)
		// Use stored config if no valid config provided
		jsonConfig = make(map[string]interface{})
	}

	// Convert JSON config to gps.Config with proper types
	config := ws.parseConfig(jsonConfig)

	// If no specific config was provided, use the last stored config
	if len(jsonConfig) == 0 {
		config = ws.lastConfig
	}

	// Store this config as the last used config
	ws.lastConfig = config

	log.Printf("Starting simulator with config: %+v", config)

	// Stop existing simulator if running
	if ws.simulator != nil {
		if ws.simulator.IsRunning() {
			log.Printf("Stopping existing simulator before starting new one")
			ws.simulator.Stop()
		}
		ws.simulator = nil
	}

	// Create new simulator
	simulator, err := gps.NewSimulator(config)
	if err != nil {
		log.Printf("Failed to create simulator: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create simulator: %v", err), http.StatusBadRequest)
		return
	}

	// Add callback to broadcast NMEA data
	simulator.AddCallback(func(data gps.NMEAData) {
		select {
		case ws.broadcast <- data:
		default:
			// Channel full, skip this update
		}
	})

	// Start simulator
	if err := simulator.Start(); err != nil {
		log.Printf("Failed to start simulator: %v", err)
		http.Error(w, fmt.Sprintf("Failed to start simulator: %v", err), http.StatusInternalServerError)
		return
	}

	ws.simulator = simulator
	log.Printf("Simulator started successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (ws *WebServer) handleStopSimulator(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Stop request received")

	if ws.simulator != nil {
		if ws.simulator.IsRunning() {
			log.Printf("Stopping running simulator")
			if err := ws.simulator.Stop(); err != nil {
				log.Printf("Failed to stop simulator: %v", err)
				http.Error(w, fmt.Sprintf("Failed to stop simulator: %v", err), http.StatusInternalServerError)
				return
			}
		}
		// Clear the simulator reference to allow fresh start
		ws.simulator = nil
		log.Printf("Simulator stopped and cleared")
	} else {
		log.Printf("No simulator to stop")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func (ws *WebServer) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var status interface{}
	if ws.simulator != nil {
		status = ws.simulator.GetStatus()
	} else {
		status = map[string]interface{}{
			"running": false,
			"message": "No simulator instance",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (ws *WebServer) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON config from request
	var jsonConfig map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&jsonConfig); err != nil {
		log.Printf("JSON decode error: %v", err)
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Convert JSON config to gps.Config with proper types
	config := ws.parseConfig(jsonConfig)

	log.Printf("Updating config: %+v", config)

	// Store the configuration for future use
	ws.lastConfig = config

	// If simulator is running, update its configuration
	if ws.simulator != nil && ws.simulator.IsRunning() {
		if err := ws.simulator.UpdateConfig(config); err != nil {
			log.Printf("Failed to update config: %v", err)
			http.Error(w, fmt.Sprintf("Failed to update config: %v", err), http.StatusBadRequest)
			return
		}
		log.Printf("Configuration updated for running simulator")
	} else {
		// If no simulator is running, just store the config for next start
		log.Printf("Configuration stored - will be used when simulator starts")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// parseConfig converts JSON map to gps.Config with proper type conversion
func (ws *WebServer) parseConfig(jsonConfig map[string]interface{}) gps.Config {
	config := gps.DefaultConfig()

	// Helper function to safely convert interface{} to float64
	getFloat := func(key string, defaultValue float64) float64 {
		if val, ok := jsonConfig[key]; ok {
			if f, ok := val.(float64); ok {
				return f
			}
		}
		return defaultValue
	}

	// Helper function to safely convert interface{} to int
	getInt := func(key string, defaultValue int) int {
		if val, ok := jsonConfig[key]; ok {
			if f, ok := val.(float64); ok {
				return int(f)
			}
		}
		return defaultValue
	}

	// Helper function to safely convert interface{} to bool
	getBool := func(key string, defaultValue bool) bool {
		if val, ok := jsonConfig[key]; ok {
			if b, ok := val.(bool); ok {
				return b
			}
		}
		return defaultValue
	}

	// Helper function to safely convert interface{} to time.Duration
	getDuration := func(key string, defaultValue time.Duration) time.Duration {
		if val, ok := jsonConfig[key]; ok {
			if s, ok := val.(string); ok {
				if d, err := time.ParseDuration(s); err == nil {
					return d
				}
			}
		}
		return defaultValue
	}

	// Parse all config fields
	config.Latitude = getFloat("latitude", config.Latitude)
	config.Longitude = getFloat("longitude", config.Longitude)
	config.Radius = getFloat("radius", config.Radius)
	config.Altitude = getFloat("altitude", config.Altitude)
	config.Jitter = getFloat("jitter", config.Jitter)
	config.AltitudeJitter = getFloat("altitude_jitter", config.AltitudeJitter)
	config.Speed = getFloat("speed", config.Speed)
	config.Course = getFloat("course", config.Course)
	config.Satellites = getInt("satellites", config.Satellites)
	config.TimeToLock = getDuration("time_to_lock", config.TimeToLock)
	config.OutputRate = getDuration("output_rate", config.OutputRate)
	config.BaudRate = getInt("baud_rate", config.BaudRate)
	config.Quiet = getBool("quiet", config.Quiet)
	config.GPXEnabled = getBool("gpx_enabled", config.GPXEnabled)
	config.Duration = getDuration("duration", config.Duration)
	config.ReplaySpeed = getFloat("replay_speed", config.ReplaySpeed)
	config.ReplayLoop = getBool("replay_loop", config.ReplayLoop)

	return config
}

func main() {
	webServer := NewWebServer()

	// Start the broadcast goroutine
	go webServer.broadcastToClients()

	// Create router
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/start", webServer.handleStartSimulator).Methods("POST")
	api.HandleFunc("/stop", webServer.handleStopSimulator).Methods("POST")
	api.HandleFunc("/status", webServer.handleGetStatus).Methods("GET")
	api.HandleFunc("/config", webServer.handleUpdateConfig).Methods("POST")
	api.HandleFunc("/ws", webServer.handleWebSocket)

	// Handle favicon.ico requests
	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Serve static files
	staticDir := filepath.Join(".", "static")
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticDir)))

	port := 8080
	log.Printf("Starting GPS Simulator Web Server on port %d", port)
	log.Printf("Open http://localhost:%d in your browser", port)

	server := &http.Server{
		Addr:         ":" + strconv.Itoa(port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}
