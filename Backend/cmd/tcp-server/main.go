package main

import (
	"log"
	"mangahub/internal/tcp"
)

func main() {
	log.Println("🚀 Starting TCP Progress Sync Server...")
	server := tcp.NewTCPServer("9090")
	server.Start()
}
