module github.com/Bucknalla/go-gps-simulator/web

go 1.21

require (
	github.com/Bucknalla/go-gps-simulator/gps v0.0.0-00010101000000-000000000000
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.1
)

require golang.org/x/net v0.17.0 // indirect

replace github.com/Bucknalla/go-gps-simulator/gps => ../gps
