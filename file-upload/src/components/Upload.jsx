import { useEffect, useState } from "react";

const Upload = () => {
  const [file, setFile] = useState(null);
  const [fileResponse, setFileResponse] = useState(null);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploadStatus, setUploadStatus] = useState("");
  const [isUploadComplete, setIsUploadComplete] = useState(false);
  const [uploadDuration, setUploadDuration] = useState(null);

  const handleFile = (e) => {
    if (e.target.files.length > 0) {
      setFile(e.target.files[0]);
      setFileResponse(null);
      setUploadProgress(0);
      setUploadStatus("");
      setIsUploadComplete(false);
      setUploadDuration(null); 
    }
  };

  const removeHandler = () => {
    setFile(null);
    setFileResponse(null);
    setUploadProgress(0);
    setUploadStatus("");
    setIsUploadComplete(false);
    setUploadDuration(null);
  };

  
  useEffect(() => {
    const uploadFile = async () => {
      if (!file) return;
      setIsUploading(true);
      const payload = {
        fileName: file.name,
        fileSize: file.size,
      };

      try {
        const response = await fetch("http://localhost:8080/upload/init", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(payload),
        });

        if (!response.ok) {
          throw new Error(`Upload failed with status: ${response.status}`);
        }

        const result = await response.json();
        setFileResponse(result);
        return result;
      } catch (error) {
        console.error("Error during upload initialization:", error);
        setUploadStatus("Failed to initialize upload session.");
      } finally {
        setIsUploading(false);
      }
    };

    uploadFile();
  }, [file]);

  const handleChunkAndUpload = async () => {
    if (!file || !fileResponse) {
      alert("Please select a file and wait for server initialization.");
      return;
    }

    setIsUploading(true);
    setIsUploadComplete(false);
    setUploadDuration(null);
    
    let start = 0;
    let chunkIdx = 0;
    const startTime = performance.now();

    try {
      while (start < file.size) {
        const end = Math.min(start + fileResponse.ChunkSize, file.size);
        const chunk = file.slice(start, end);
        
        const formData = new FormData();
        formData.append("chunk", chunk);

        setUploadStatus(
          `Uploading chunk ${chunkIdx + 1} of ${fileResponse.TotalChunks}...`
        );

        const response = await fetch(
          `http://localhost:8080/upload/${fileResponse.FileID}?chunkNo=${chunkIdx}`, 
          {
            method: "POST",
            body: formData,
          }
        );

        if (!response.ok) {
          throw new Error(`Failed to upload chunk with ID ${chunkIdx}`);
        }

        chunkIdx++;
        const progress = Math.round((chunkIdx / fileResponse.TotalChunks) * 100);
        setUploadProgress(progress);

        start = end;
      }

     
      setUploadStatus("All chunks uploaded! Finalizing file on server...");
      
      const completeResponse = await fetch(
        `http://localhost:8080/upload/${fileResponse.FileID}/complete`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json"
          }
        }
      );

      if (!completeResponse.ok) {
        throw new Error("Failed to finalize and merge chunks on the server.");
      }

      
      const endTime = performance.now();
      const totalTimeInSeconds = ((endTime - startTime) / 1000).toFixed(2);
      
      
      if (totalTimeInSeconds > 60) {
        const minutes = Math.floor(totalTimeInSeconds / 60);
        const seconds = (totalTimeInSeconds % 60).toFixed(1);
        setUploadDuration(`${minutes}m ${seconds}s`);
      } else {
        setUploadDuration(`${totalTimeInSeconds} seconds`);
      }

      setUploadStatus("File uploaded and processed successfully!");
      setIsUploadComplete(true);

    } catch (error) {
      console.error("Upload error:", error);
      setUploadStatus(`Upload failed: ${error.message}`);
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <div className="flex flex-col items-center justify-center w-full max-w-xl mx-auto p-4">
      {/* Clickable Drag & Drop Container */}
      <label
        htmlFor="dropzone-file"
        className="flex flex-col items-center justify-center w-full h-64 border-2 border-dashed border-gray-300 rounded-lg cursor-pointer bg-gray-50 hover:bg-gray-100 transition-all duration-200"
      >
        <div className="flex flex-col items-center justify-center pt-5 pb-6 text-center px-4">
          <svg
            className="w-10 h-10 mb-3 text-gray-400"
            aria-hidden="true"
            xmlns="http://w3.org"
            fill="none"
            viewBox="0 0 20 16"
          >
            <path
              stroke="currentColor"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="2"
              d="M13 13h3a3 3 0 0 0 0-6h-.025A5.56 5.56 0 0 0 16 6.5 5.5 5.5 0 0 0 5.207 5.021C5.137 5.017 5.071 5 5 5a4 4 0 0 0 0 8h2.167M10 15V6m0 0L8 8m2-2 2 2"
            />
          </svg>
          <p className="mb-2 text-sm text-gray-500">
            <span className="font-semibold">Click to upload</span> or drag and drop
          </p>
        </div>
        <input
          id="dropzone-file"
          type="file"
          className="hidden"
          onChange={handleFile}
          disabled={isUploading}
        />
      </label>

      {/* Selected File Status Area */}
      {file && (
        <div className="mt-4 w-full bg-blue-50 border border-blue-200 rounded-lg p-4 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 animate-fade-in">
          <div className="flex items-center space-x-3 overflow-hidden w-full sm:w-auto">
            <span className="text-xl shrink-0" aria-hidden="true">📄</span>
            <div className="flex flex-col min-w-0">
              <span className="text-blue-700 font-medium text-sm truncate block">
                {file.name}
              </span>
              <span className="text-xs text-blue-500/80">
                {(file.size / 1024 / 1024).toFixed(2)} MB
              </span>
            </div>
          </div>

          <div className="flex items-center space-x-3 w-full sm:w-auto justify-end shrink-0">
            <button
              onClick={removeHandler}
              disabled={isUploading}
              className="px-3 py-1.5 text-xs font-semibold text-gray-500 hover:text-red-600 hover:bg-red-50 rounded-md transition duration-200 cursor-pointer disabled:opacity-50"
              type="button"
            >
              Remove
            </button>
            <button
              onClick={handleChunkAndUpload}
              disabled={isUploading || !fileResponse || isUploadComplete}
              className="px-4 py-1.5 text-xs font-semibold text-white bg-blue-600 hover:bg-blue-700 rounded-md shadow-sm transition duration-200 cursor-pointer disabled:opacity-50 flex items-center gap-2"
              type="button"
            >
              {isUploading ? "Uploading..." : isUploadComplete ? "Uploaded" : "Upload File"}
            </button>
          </div>
        </div>
      )}

      {/* Real-time uploading metrics container */}
      {(isUploading || uploadStatus) && (
        <div className="mt-4 w-full bg-gray-50 border border-gray-200 rounded-lg p-4">
          <p className="text-xs text-gray-600 mb-2 font-medium">{uploadStatus}</p>
          <div className="w-full bg-gray-200 rounded-full h-2.5">
            <div 
              className={`h-2.5 rounded-full transition-all duration-300 ${
                isUploadComplete ? 'bg-green-500' : 'bg-blue-600'
              }`} 
              style={{ width: `${uploadProgress}%` }}
            ></div>
          </div>
          <span className="text-xxs text-gray-400 mt-1 block text-right font-mono">{uploadProgress}%</span>
        </div>
      )}

      {/* MODIFIED UI SECTION: Success Card showing the calculated duration */}
      {isUploadComplete && (
        <div className="mt-4 w-full bg-green-50 border border-green-200 rounded-lg p-4 flex items-center gap-3 animate-fade-in">
          <span className="text-2xl text-green-500" aria-hidden="true">🎉</span>
          <div className="flex flex-col w-full">
            <div className="flex justify-between items-center">
              <span className="text-green-800 font-semibold text-sm">Success!</span>
              {/* Displays the time here */}
              {uploadDuration && (
                <span className="text-xxs font-mono bg-green-200/60 text-green-800 px-2 py-0.5 rounded-md">
                  ⏱ Time: {uploadDuration}
                </span>
              )}
            </div>
            <span className="text-xs text-green-600 mt-1">
              The backend has merged all chunks into the final file successfully.
            </span>
          </div>
        </div>
      )}

      {/* Server Upload Response Details Card */}
      {fileResponse && !isUploadComplete && (
        <div className="mt-6 w-full bg-white border border-gray-200 rounded-lg p-5 shadow-sm animate-fade-in">
          <h3 className="text-sm font-semibold text-gray-900 border-b border-gray-100 pb-3 mb-4 flex items-center gap-2">
            <span className="text-green-500">✓</span> Server Initialization Success
          </h3>
          <div className="space-y-3 text-xs">
            <div className="flex justify-between items-center py-1 border-b border-gray-50">
              <span className="text-gray-500 font-medium">File ID</span>
              <span className="font-mono text-gray-700 bg-gray-50 px-2 py-0.5 rounded select-all">
                {fileResponse.FileID}
              </span>
            </div>
            <div className="flex justify-between items-center py-1 border-b border-gray-50">
              <span className="text-gray-500 font-medium">Chunk Configuration</span>
              <span className="text-gray-800">
                {fileResponse.TotalChunks} chunks × {(fileResponse.ChunkSize / 1024 / 1024).toFixed(1)} MB
              </span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Upload;
