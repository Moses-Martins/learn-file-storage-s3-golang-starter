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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"      

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	
	const maxUploadSize = 1 << 30 // 1GB
    r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

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

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized user", err)
		return
	}


	file, header, err := r.FormFile("video")
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

	if (mediatype != "video/mp4") {
		respondWithError(w, http.StatusBadRequest, "Cannot upload content of that type", nil)
		return
	}


	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Cannot create file path", err)
		return
    }

	defer os.Remove(tmpFile.Name()) // clean up the file when done
    defer tmpFile.Close()



	_, err = io.Copy(tmpFile, file)

	_, err = tmpFile.Seek(0, io.SeekStart)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Cannot reset the file pointer", err)
		return
    }

	parts := strings.Split(mediatype, "/")

	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)

	randomString := base64.RawURLEncoding.EncodeToString(randomBytes)


	processedPath , err := processVideoForFastStart(tmpFile.Name())
	if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Cannot create Processed video", err)
		return
    }

	fileProcessed, err := os.Open(processedPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot open Processed file", err)
		return
	}
	defer fileProcessed.Close()	



	aspect_ratio, err := getVideoAspectRatio(processedPath)
	if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Cannot get Aspect Ratio", err)
		return
    }

	if aspect_ratio == "16:9" {
		_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
        Bucket: aws.String(cfg.s3Bucket),
        Key:    aws.String("landscape/" + randomString + "." + parts[1]),
        Body:   fileProcessed, 
        ContentType: aws.String(mediatype),
    })

	data_url := fmt.Sprintf("https://%s/landscape/%s.%s", cfg.s3CfDistribution, randomString, parts[1])
	video.VideoURL = &data_url

	} else if aspect_ratio == "9:16" {
		_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
        Bucket: aws.String(cfg.s3Bucket),
        Key:    aws.String("portrait/" + randomString + "." + parts[1]),
        Body:   fileProcessed, 
        ContentType: aws.String(mediatype),
    })
	
	data_url := fmt.Sprintf("https://%s/portrait/%s.%s", cfg.s3CfDistribution, randomString, parts[1])
	video.VideoURL = &data_url
	} else {
		_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
        Bucket: aws.String(cfg.s3Bucket),
        Key:    aws.String("other/" + randomString + "." + parts[1]),
        Body:   fileProcessed, 
        ContentType: aws.String(mediatype),
    })
	
	data_url := fmt.Sprintf("https://%s/other/%s.%s", cfg.s3CfDistribution, randomString, parts[1])
	video.VideoURL = &data_url
	}


	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot update video", err)
		return
	}

	
	respondWithJSON(w, http.StatusOK, video)

}
