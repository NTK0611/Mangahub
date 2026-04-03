package main

import (
	"log"
	"mangahub/internal/udp"
)

func main() {
	log.Println("🚀 Starting UDP Notification Server...")
	server := udp.NewUDPServer("9091")
	server.Start()
}
