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
	//Handles large file uploads by setting a maximum memory limit for parsing the multipart form data. The limit is set to 32 MB (32 << 20 bytes). If the request body exceeds this limit, an error will be returned. This helps prevent excessive memory usage and potential denial-of-service attacks.
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	//Reading file binary stream from body
	file, metadata, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	log.Printf("Uploaded File: %+v\n", metadata.Filename)

	//Creating the upload directory if it doesn't exist
	err = os.MkdirAll("./uploads", os.ModePerm)
	if err != nil {
		http.Error(w, "Error creating upload directory", http.StatusInternalServerError)
		return
	}

	fileName := uuid.New().String()
	ext := filepath.Ext(metadata.Filename)

	//Creating a new file in the upload directory with the same name as the uploaded file
	dstPath := filepath.Join("./uploads", filepath.Base(fileName)+ext)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Error in storing file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	//Copying the uploaded files's content to the newly created file
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Error in saving file", http.StatusInternalServerError)
		return
	}
	finalTime := time.Since(currTime)
	w.WriteHeader(http.StatusOK)
	response := Response{
		Message:  "File uploaded Successfully",
		FileId:   fileName,
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

	session := models.UploadSession{
		FileID:      uuid.New().String(),
		FileName:    fileName,
		TotalSize:   fileSize,
		ChunkSize:   ChunkSize,
		TotalChunks: totalChunks,
	}

	mu.Lock()
	uploadSessions[session.FileID] = session
	mu.Unlock()

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

	// Get file ID and chunk number
	fileID := r.PathValue("fileId")
	chunkNoStr := r.URL.Query().Get("chunkNo")

	if fileID == "" || chunkNoStr == "" {
		http.Error(w, "fileId and chunkNo are required", http.StatusBadRequest)
		return
	}

	// Convert chunk number to integer
	chunkNo, err := strconv.Atoi(chunkNoStr)
	if err != nil {
		http.Error(w, "Invalid chunk number", http.StatusBadRequest)
		return
	}

	// Retrieve upload session
	mu.RLock()
	session, exists := uploadSessions[fileID]
	mu.RUnlock()

	if !exists {
		http.Error(w, "Upload session not found", http.StatusNotFound)
		return
	}

	// Validate chunk number
	if chunkNo < 0 || chunkNo >= session.TotalChunks {
		http.Error(w, "Invalid chunk number", http.StatusBadRequest)
		return
	}

	// Get chunk file
	chunk, handler, err := r.FormFile("chunk")
	if err != nil {
		http.Error(w, "Chunk is required", http.StatusBadRequest)
		return
	}
	defer chunk.Close()

	// Every chunk can be smaller than ChunkSize,
	// but cannot exceed ChunkSize.
	if handler.Size <= 0 || handler.Size > session.ChunkSize {
		http.Error(w, "Invalid chunk size", http.StatusBadRequest)
		return
	}

	// Create directory for this upload
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

	// Store chunk using its chunk number
	chunkPath := filepath.Join(
		chunkDir,
		fmt.Sprintf("chunk-%d", chunkNo),
	)

	// Don't overwrite an already uploaded chunk
	if _, err := os.Stat(chunkPath); err == nil {
		http.Error(w, "Chunk already uploaded", http.StatusConflict)
		return
	}

	// Create chunk file
	dst, err := os.Create(chunkPath)
	if err != nil {
		http.Error(
			w,
			"Error creating chunk file",
			http.StatusInternalServerError,
		)
		return
	}

	// Copy chunk data to disk
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

	// Close file
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

	// Validate actual bytes written
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

	// Get file ID from URL
	fileID := r.PathValue("fileId")

	if fileID == "" {
		http.Error(w, "fileId is required", http.StatusBadRequest)
		return
	}

	// Retrieve upload session
	mu.RLock()
	session, exists := uploadSessions[fileID]
	mu.RUnlock()

	if !exists {
		http.Error(w, "Upload session not found", http.StatusNotFound)
		return
	}

	chunkDir := filepath.Join("./uploads", fileID, "chunks")

	// Check that every chunk exists
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

		// Every chunk must be:
		// > 0 bytes
		// <= configured chunk size
		if info.Size() <= 0 || info.Size() > session.ChunkSize {
			http.Error(
				w,
				fmt.Sprintf("Invalid size for chunk %d", i),
				http.StatusConflict,
			)
			return
		}
	}

	// Create final file path
	fileName := filepath.Base(session.FileName)

	finalPath := filepath.Join(
		"./uploads",
		fileID,
		fileName,
	)

	// Don't overwrite an existing final file
	if _, err := os.Stat(finalPath); err == nil {
		http.Error(
			w,
			"Final file already exists",
			http.StatusConflict,
		)
		return
	}

	// Create final file
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

	// Read chunks sequentially
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

	// Verify reconstructed file size
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

	// Close final file
	if err := finalFile.Close(); err != nil {
		os.Remove(finalPath)

		http.Error(
			w,
			"Error closing final file",
			http.StatusInternalServerError,
		)
		return
	}

	// Remove chunks after successful reconstruction
	if err := os.RemoveAll(chunkDir); err != nil {
		http.Error(
			w,
			"File created but failed to remove chunks",
			http.StatusInternalServerError,
		)
		return
	}

	// Remove completed upload session from memory
	mu.Lock()
	delete(uploadSessions, fileID)
	mu.Unlock()

	response := map[string]interface{}{
		"message":  "File uploaded successfully",
		"fileId":   fileID,
		"fileName": fileName,
		"size":     totalWritten,
		"path":     finalPath,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
