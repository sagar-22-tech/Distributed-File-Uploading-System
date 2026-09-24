# Distributed File Upload System

A backend-focused distributed file upload system built with Go, designed to efficiently handle large file uploads through chunking, concurrent transfers, and scalable storage architecture.

## Current Features

* Chunked file uploads
* Metadata-based upload initialization
* Configurable chunk size
* Individual chunk validation
* Concurrent chunk uploads
* Sequential file reconstruction
* Upload session management
* Duplicate and missing chunk detection
* File size verification
* Automatic chunk cleanup
* Thread-safe in-memory session management

## Architecture

```text
Client
   │
   ├── Upload metadata
   │
   ▼
Upload Initialization
   │
   ├── fileId
   ├── chunkSize
   └── totalChunks
   │
   ▼
Chunk Uploads
   │
   ├── chunk-0
   ├── chunk-1
   ├── chunk-2
   └── ...
   │
   ▼
Upload Completion
   │
   ├── Validate chunks
   ├── Reconstruct file
   └── Verify final size
   │
   ▼
Final File
```

## Technology Stack

* Go
* Standard `net/http`
* REST APIs
* Multipart file uploads
* Concurrent HTTP requests
* Local filesystem storage

