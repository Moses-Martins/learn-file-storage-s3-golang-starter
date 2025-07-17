package main

import (
	"io"
	"os"
	"fmt"
	"mime"
	"strings"
	"net/http"
	"crypto/rand"
	"encoding/base64"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)


func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)
	
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}

	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type for thumbnail", nil)
		return
	}


	mediatype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", nil)
		return
	}

	if (mediatype != "image/jpeg") && (mediatype != "image/png") {
		respondWithError(w, http.StatusBadRequest, "Cannot upload content of that type", nil)
		return
	}


	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized user", err)
		return
	}

	parts := strings.Split(mediatype, "/")

	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)

	randomString := base64.RawURLEncoding.EncodeToString(randomBytes)

	filename := fmt.Sprintf("%s.%s", randomString, parts[1])
	path := filepath.Join(cfg.assetsRoot, "/", filename)


	file_dir, err := os.Create(path)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Cannot create file path", err)
		return
    }
    defer file_dir.Close()

	_, err = io.Copy(file_dir, file)


	data_url := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, randomString, parts[1])
	video.ThumbnailURL = &data_url
		

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
