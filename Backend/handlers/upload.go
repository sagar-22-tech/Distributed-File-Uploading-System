package handlers

import (
	"encoding/json"
	"file-system/models"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Response struct {
	Message  string `json:"message"`
	FileId   string `json:"file_id"`
	FileName string `json:"file_name"`
	Time     string `json:"time"`
}

const ChunkSize int64 = 10 * 1024 * 1024

var (
	uploadSessions = make(map[string]models.UploadSession)
	mu             sync.RWMutex
)

func scheduleCleanup(fileID string, delay time.Duration) {
	time.AfterFunc(delay, func() {
		targetDir := filepath.Join("./uploads", fileID)

		if err := os.RemoveAll(targetDir); err != nil {
			log.Printf("[Auto-Cleanup Failed] Could not remove directory %s: %v\n", targetDir, err)
		} else {
			log.Printf("[Auto-Cleanup Success] Removed upload directory: %s\n", targetDir)
		}

		mu.Lock()
		delete(uploadSessions, fileID)
		mu.Unlock()
	})
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Health check passed"))
}

func Upload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	currTime := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, metadata, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	log.Printf("Uploaded File: %+v\n", metadata.Filename)

	fileID := uuid.New().String()
	ext := filepath.Ext(metadata.Filename)

	fileDir := filepath.Join("./uploads", fileID)
	err = os.MkdirAll(fileDir, os.ModePerm)
	if err != nil {
		http.Error(w, "Error creating upload directory", http.StatusInternalServerError)
		return
	}

	scheduleCleanup(fileID, 5*time.Minute)

	dstPath := filepath.Join(fileDir, filepath.Base(fileID)+ext)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Error in storing file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Error in saving file", http.StatusInternalServerError)
		return
	}

	finalTime := time.Since(currTime)
	w.WriteHeader(http.StatusOK)
	response := Response{
		Message:  "File uploaded successfully (will auto-delete in 5 minutes)",
		FileId:   fileID,
		FileName: metadata.Filename,
		Time:     finalTime.String(),
	}

	json.NewEncoder(w).Encode(response)
}

func UploadInit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var requestData struct {
		FileName string `json:"fileName"`
		FileSize int64  `json:"fileSize"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	if requestData.FileName == "" || requestData.FileSize <= 0 {
		http.Error(w, "Invalid request data", http.StatusBadRequest)
		return
	}

	fileName := requestData.FileName
	fileSize := requestData.FileSize

	totalChunks := int((fileSize + ChunkSize - 1) / ChunkSize)
	fileID := uuid.New().String()

	session := models.UploadSession{
		FileID:      fileID,
		FileName:    fileName,
		TotalSize:   fileSize,
		ChunkSize:   ChunkSize,
		TotalChunks: totalChunks,
	}

	mu.Lock()
	uploadSessions[session.FileID] = session
	mu.Unlock()

	scheduleCleanup(fileID, 5*time.Minute)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(session)
}

func UploadChunk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	fileID := r.PathValue("fileId")
	chunkNoStr := r.URL.Query().Get("chunkNo")

	if fileID == "" || chunkNoStr == "" {
		http.Error(w, "fileId and chunkNo are required", http.StatusBadRequest)
		return
	}

	chunkNo, err := strconv.Atoi(chunkNoStr)
	if err != nil {
		http.Error(w, "Invalid chunk number", http.StatusBadRequest)
		return
	}

	mu.RLock()
	session, exists := uploadSessions[fileID]
	mu.RUnlock()

	if !exists {
		http.Error(w, "Upload session not found or expired", http.StatusNotFound)
		return
	}

	if chunkNo < 0 || chunkNo >= session.TotalChunks {
		http.Error(w, "Invalid chunk number", http.StatusBadRequest)
		return
	}

	chunk, handler, err := r.FormFile("chunk")
	if err != nil {
		http.Error(w, "Chunk is required", http.StatusBadRequest)
		return
	}
	defer chunk.Close()

	if handler.Size <= 0 || handler.Size > session.ChunkSize {
		http.Error(w, "Invalid chunk size", http.StatusBadRequest)
		return
	}

	chunkDir := filepath.Join("./uploads", fileID, "chunks")

	err = os.MkdirAll(chunkDir, os.ModePerm)
	if err != nil {
		http.Error(
			w,
			"Error creating chunk directory",
			http.StatusInternalServerError,
		)
		return
	}

	chunkPath := filepath.Join(
		chunkDir,
		fmt.Sprintf("chunk-%d", chunkNo),
	)

	if _, err := os.Stat(chunkPath); err == nil {
		http.Error(w, "Chunk already uploaded", http.StatusConflict)
		return
	}

	dst, err := os.Create(chunkPath)
	if err != nil {
		http.Error(
			w,
			"Error creating chunk file",
			http.StatusInternalServerError,
		)
		return
	}

	written, err := io.Copy(dst, chunk)
	if err != nil {
		dst.Close()
		os.Remove(chunkPath)

		http.Error(
			w,
			"Error saving chunk file",
			http.StatusInternalServerError,
		)
		return
	}

	err = dst.Close()
	if err != nil {
		os.Remove(chunkPath)

		http.Error(
			w,
			"Error closing chunk file",
			http.StatusInternalServerError,
		)
		return
	}

	if written != handler.Size {
		os.Remove(chunkPath)

		http.Error(
			w,
			"Chunk size mismatch",
			http.StatusBadRequest,
		)
		return
	}

	response := map[string]interface{}{
		"message":     "Chunk uploaded successfully",
		"fileId":      fileID,
		"chunkNumber": chunkNo,
		"size":        written,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func UploadComplete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fileID := r.PathValue("fileId")

	if fileID == "" {
		http.Error(w, "fileId is required", http.StatusBadRequest)
		return
	}

	mu.RLock()
	session, exists := uploadSessions[fileID]
	mu.RUnlock()

	if !exists {
		http.Error(w, "Upload session not found or expired", http.StatusNotFound)
		return
	}

	chunkDir := filepath.Join("./uploads", fileID, "chunks")

	for i := 0; i < session.TotalChunks; i++ {
		chunkPath := filepath.Join(
			chunkDir,
			fmt.Sprintf("chunk-%d", i),
		)

		info, err := os.Stat(chunkPath)

		if err != nil {
			if os.IsNotExist(err) {
				http.Error(
					w,
					fmt.Sprintf("Chunk %d is missing", i),
					http.StatusConflict,
				)
				return
			}

			http.Error(
				w,
				"Error checking chunk",
				http.StatusInternalServerError,
			)
			return
		}

		if info.Size() <= 0 || info.Size() > session.ChunkSize {
			http.Error(
				w,
				fmt.Sprintf("Invalid size for chunk %d", i),
				http.StatusConflict,
			)
			return
		}
	}

	fileName := filepath.Base(session.FileName)

	finalPath := filepath.Join(
		"./uploads",
		fileID,
		fileName,
	)

	if _, err := os.Stat(finalPath); err == nil {
		http.Error(
			w,
			"Final file already exists",
			http.StatusConflict,
		)
		return
	}

	finalFile, err := os.Create(finalPath)

	if err != nil {
		http.Error(
			w,
			"Error creating final file",
			http.StatusInternalServerError,
		)
		return
	}

	var totalWritten int64

	for i := 0; i < session.TotalChunks; i++ {
		chunkPath := filepath.Join(
			chunkDir,
			fmt.Sprintf("chunk-%d", i),
		)

		chunkFile, err := os.Open(chunkPath)

		if err != nil {
			finalFile.Close()
			os.Remove(finalPath)

			http.Error(
				w,
				fmt.Sprintf("Error opening chunk %d", i),
				http.StatusInternalServerError,
			)
			return
		}

		written, err := io.Copy(finalFile, chunkFile)

		chunkFile.Close()

		if err != nil {
			finalFile.Close()
			os.Remove(finalPath)

			http.Error(
				w,
				fmt.Sprintf("Error writing chunk %d", i),
				http.StatusInternalServerError,
			)
			return
		}

		totalWritten += written
	}

	if totalWritten != session.TotalSize {
		finalFile.Close()
		os.Remove(finalPath)

		http.Error(
			w,
			"Final file size mismatch",
			http.StatusInternalServerError,
		)
		return
	}

	if err := finalFile.Close(); err != nil {
		os.Remove(finalPath)

		http.Error(
			w,
			"Error closing final file",
			http.StatusInternalServerError,
		)
		return
	}

	if err := os.RemoveAll(chunkDir); err != nil {
		http.Error(
			w,
			"File created but failed to remove chunks",
			http.StatusInternalServerError,
		)
		return
	}

	response := map[string]interface{}{
		"message":  "File uploaded successfully (will auto-delete in 5 minutes)",
		"fileId":   fileID,
		"fileName": fileName,
		"size":     totalWritten,
		"path":     finalPath,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
