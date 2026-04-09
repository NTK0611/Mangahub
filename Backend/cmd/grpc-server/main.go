package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpcserver "mangahub/internal/grpc"
	"mangahub/pkg/database"
	pb "mangahub/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	defaultPort   = "50051"
	defaultDBPath = "./data/mangahub.db"
)

func main() {
	// Allow overriding defaults via environment variables
	port := getEnv("GRPC_PORT", defaultPort)
	dbPath := getEnv("DB_PATH", defaultDBPath)

	// ── Database ──────────────────────────────────────────────────────────────
	db := database.InitDB(dbPath)
	defer db.Close()

	database.CreateTables(db)

	// ── gRPC server ───────────────────────────────────────────────────────────
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("❌ Failed to listen on port %s: %v", port, err)
	}

	grpcSrv := grpc.NewServer(
		// Add interceptors here in the future (logging, auth, etc.)
		grpc.UnaryInterceptor(loggingInterceptor),
	)

	// Register our MangaService implementation
	mangaServer := grpcserver.NewMangaServer(db)
	pb.RegisterMangaServiceServer(grpcSrv, mangaServer)

	// Register reflection service so tools like grpcurl work out of the box
	reflection.Register(grpcSrv)

	log.Printf("🚀 gRPC server listening on port %s", port)
	log.Printf("💡 Use grpcurl -plaintext localhost:%s list  to explore services", port)

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("gRPC serve error: %v", err)
		}
	}()

	<-quit
	log.Println("🛑 Shutting down gRPC server...")
	grpcSrv.GracefulStop()
	log.Println("✅ gRPC server stopped")
}

// loggingInterceptor logs every incoming unary RPC call.
func loggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	log.Printf("[gRPC] ▶  %s", info.FullMethod)
	resp, err := handler(ctx, req)
	if err != nil {
		log.Printf("[gRPC] ✗  %s → %v", info.FullMethod, err)
	} else {
		log.Printf("[gRPC] ✓  %s", info.FullMethod)
	}
	return resp, err
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
