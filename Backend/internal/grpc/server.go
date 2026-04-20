package grpc

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	pb "mangahub/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MangaServer implements the proto.MangaServiceServer interface.
type MangaServer struct {
	pb.UnimplementedMangaServiceServer
	db *sql.DB
}

func NewMangaServer(db *sql.DB) *MangaServer {
	return &MangaServer{db: db}
}

// ─── GetManga ────────────────────────────────────────────────────────────────

func (s *MangaServer) GetManga(ctx context.Context, req *pb.GetMangaRequest) (*pb.MangaResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "manga id is required")
	}

	item, err := s.fetchMangaByID(req.GetId())
	if err == sql.ErrNoRows {
		return &pb.MangaResponse{Found: false, Message: "manga not found"}, nil
	}
	if err != nil {
		log.Printf("[gRPC] GetManga error: %v", err)
		return nil, status.Error(codes.Internal, "failed to fetch manga")
	}

	return &pb.MangaResponse{Manga: item, Found: true, Message: "ok"}, nil
}

// ─── SearchManga ─────────────────────────────────────────────────────────────

func (s *MangaServer) SearchManga(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	query := "SELECT id, title, author, genres, status, total_chapters, description FROM manga WHERE 1=1"
	args := []interface{}{}

	if req.GetQuery() != "" {
		query += " AND (title LIKE ? OR author LIKE ?)"
		like := "%" + req.GetQuery() + "%"
		args = append(args, like, like)
	}
	if req.GetStatus() != "" {
		query += " AND status = ?"
		args = append(args, req.GetStatus())
	}
	if req.GetGenre() != "" {
		query += " AND genres LIKE ?"
		args = append(args, "%"+req.GetGenre()+"%")
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("[gRPC] SearchManga query error: %v", err)
		return nil, status.Error(codes.Internal, "database query failed")
	}
	defer rows.Close()

	var results []*pb.MangaItem
	for rows.Next() {
		item, err := scanMangaRow(rows)
		if err != nil {
			log.Printf("[gRPC] SearchManga scan error: %v", err)
			continue
		}
		results = append(results, item)
	}

	if results == nil {
		results = []*pb.MangaItem{}
	}

	return &pb.SearchResponse{Manga: results, Count: int32(len(results))}, nil
}

// ─── UpdateProgress ──────────────────────────────────────────────────────────

func (s *MangaServer) UpdateProgress(ctx context.Context, req *pb.ProgressRequest) (*pb.ProgressResponse, error) {
	if req.GetUserId() == "" || req.GetMangaId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and manga_id are required")
	}

	validStatuses := map[string]bool{
		"reading": true, "completed": true, "plan_to_read": true, "dropped": true,
	}
	if req.GetStatus() != "" && !validStatuses[req.GetStatus()] {
		return nil, status.Error(codes.InvalidArgument, "invalid status value")
	}

	var exists string
	err := s.db.QueryRowContext(ctx, "SELECT id FROM manga WHERE id = ?", req.GetMangaId()).Scan(&exists)
	if err == sql.ErrNoRows {
		return &pb.ProgressResponse{Success: false, Message: "manga not found"}, nil
	}
	if err != nil {
		log.Printf("[gRPC] UpdateProgress manga check error: %v", err)
		return nil, status.Error(codes.Internal, "failed to verify manga")
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO user_progress (user_id, manga_id, current_chapter, status, updated_at)
         VALUES (?, ?, ?, ?, ?)
         ON CONFLICT(user_id, manga_id) DO UPDATE SET
             current_chapter = excluded.current_chapter,
             status          = excluded.status,
             updated_at      = excluded.updated_at`,
		req.GetUserId(), req.GetMangaId(), req.GetCurrentChapter(), req.GetStatus(), time.Now().UTC(),
	)
	if err != nil {
		log.Printf("[gRPC] UpdateProgress exec error: %v", err)
		return nil, status.Error(codes.Internal, "failed to update progress")
	}

	log.Printf("[gRPC] Progress updated — user=%s manga=%s chapter=%d",
		req.GetUserId(), req.GetMangaId(), req.GetCurrentChapter())

	return &pb.ProgressResponse{Success: true, Message: "progress updated successfully"}, nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (s *MangaServer) fetchMangaByID(id string) (*pb.MangaItem, error) {
	row := s.db.QueryRow(
		"SELECT id, title, author, genres, status, total_chapters, description FROM manga WHERE id = ?", id,
	)
	return scanMangaRow(row)
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanMangaRow(s scanner) (*pb.MangaItem, error) {
	var item pb.MangaItem
	var genresJSON string

	err := s.Scan(
		&item.Id,
		&item.Title,
		&item.Author,
		&genresJSON,
		&item.Status,
		&item.TotalChapters,
		&item.Description,
	)
	if err != nil {
		return nil, err
	}

	// Sanitize all string fields — MangaDex data may contain invalid UTF-8
	// which causes protobuf marshaling to fail with "string field contains invalid UTF-8"
	item.Id = sanitizeUTF8(item.Id)
	item.Title = sanitizeUTF8(item.Title)
	item.Author = sanitizeUTF8(item.Author)
	item.Status = sanitizeUTF8(item.Status)
	item.Description = sanitizeUTF8(item.Description)

	var genres []string
	if jsonErr := json.Unmarshal([]byte(genresJSON), &genres); jsonErr == nil {
		for i, g := range genres {
			genres[i] = sanitizeUTF8(g)
		}
		item.Genres = genres
	}

	return &item, nil
}

// sanitizeUTF8 replaces any invalid UTF-8 byte sequences with the Unicode
// replacement character so protobuf can marshal the string without errors.
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, "")
}
