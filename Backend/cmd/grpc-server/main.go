package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpcserver "mangahub/internal/grpc"
	"mangahub/internal/seeder"
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
	port := getEnv("GRPC_PORT", defaultPort)
	dbPath := getEnv("DB_PATH", defaultDBPath)

	// ── Database ──────────────────────────────────────────────────────────────
	db := database.InitDB(dbPath)
	defer db.Close()
	database.CreateTables(db)
	go seeder.AutoSeed(db)

	// ── gRPC server ───────────────────────────────────────────────────────────
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("❌ Failed to listen on port %s: %v", port, err)
	}

	grpcSrv := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor),
	)

	mangaServer := grpcserver.NewMangaServer(db)
	pb.RegisterMangaServiceServer(grpcSrv, mangaServer)
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
