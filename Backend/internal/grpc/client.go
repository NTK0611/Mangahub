package grpc

import (
	"context"
	"log"
	"time"

	pb "mangahub/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MangaClient wraps the generated gRPC client with helper methods
// so the HTTP API server can call gRPC internally without boilerplate.
type MangaClient struct {
	conn   *grpc.ClientConn
	client pb.MangaServiceClient
}

// NewMangaClient dials the gRPC server and returns a ready-to-use client.
//
// Example:
//
//	c, err := grpc.NewMangaClient("localhost:50051")
//	defer c.Close()
func NewMangaClient(address string) (*MangaClient, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	log.Printf("✅ gRPC client connected to %s", address)
	return &MangaClient{
		conn:   conn,
		client: pb.NewMangaServiceClient(conn),
	}, nil
}

// Close releases the underlying gRPC connection.
func (c *MangaClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// ─── Wrapper methods ──────────────────────────────────────────────────────────

// GetManga fetches a single manga by ID via gRPC.
func (c *MangaClient) GetManga(id string) (*pb.MangaItem, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.GetManga(ctx, &pb.GetMangaRequest{Id: id})
	if err != nil {
		return nil, false, err
	}
	return resp.GetManga(), resp.GetFound(), nil
}

// SearchManga searches manga with optional filters via gRPC.
func (c *MangaClient) SearchManga(query, status, genre string) ([]*pb.MangaItem, int32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.SearchManga(ctx, &pb.SearchRequest{
		Query:  query,
		Status: status,
		Genre:  genre,
	})
	if err != nil {
		return nil, 0, err
	}
	return resp.GetManga(), resp.GetCount(), nil
}

// UpdateProgress updates a user's reading progress via gRPC.
func (c *MangaClient) UpdateProgress(userID, mangaID, status string, chapter int32) (bool, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.UpdateProgress(ctx, &pb.ProgressRequest{
		UserId:         userID,
		MangaId:        mangaID,
		CurrentChapter: chapter,
		Status:         status,
	})
	if err != nil {
		return false, "", err
	}
	return resp.GetSuccess(), resp.GetMessage(), nil
}
