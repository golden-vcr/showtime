package imagegen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/ledger"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/golden-vcr/showtime/internal/alerts"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/sync/errgroup"
)

const MaxSubjectLen = 120
const NumImagesToGeneratePerPrompt = 4

const ImageAlertType = "image-generation"
const ImageAlertPointsCost = 500

type Server struct {
	q          Queries
	ledger     ledger.Client
	generation GenerationClient
	storage    StorageClient
	alertsChan chan *alerts.Alert
}

func NewServer(q *queries.Queries, ledger ledger.Client, generation GenerationClient, storage StorageClient, alertsChan chan *alerts.Alert) *Server {
	return &Server{
		q:          q,
		ledger:     ledger,
		generation: generation,
		storage:    storage,
		alertsChan: alertsChan,
	}
}

func (s *Server) RegisterRoutes(c auth.Client, r *mux.Router) {
	// Require viewer access for all image generation routes: generating image alerts
	// requires the user to spend Golden VCR Fun Points
	r.Use(func(next http.Handler) http.Handler {
		return auth.RequireAccess(c, auth.RoleViewer, next)
	})

	// POST / submits a request to generate a new image to be displayed onscreen
	for _, root := range []string{"", "/"} {
		r.Path(root).Methods("POST").HandlerFunc(s.handleRequest)
	}
}

func (s *Server) handleRequest(res http.ResponseWriter, req *http.Request) {
	// Identify the user from their authorization token
	claims, err := auth.GetClaims(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// The request's Content-Type must indicate JSON if set
	contentType := req.Header.Get("content-type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		http.Error(res, "content-type not supported", http.StatusBadRequest)
		return
	}

	// Parse the image generation request from the request body
	var payload Request
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(res, fmt.Sprintf("invalid request payload: %v", err), http.StatusBadRequest)
		return
	}
	if payload.Subject == "" {
		http.Error(res, "invalid request payload: 'subject' is required", http.StatusBadRequest)
		return
	}
	if len(payload.Subject) > MaxSubjectLen {
		http.Error(res, "invalid request payload: 'subject' must be <= 120 characters", http.StatusBadRequest)
		return
	}

	// Ensure that the user has a viewer record in the database
	if err := s.q.RecordViewerIdentity(req.Context(), queries.RecordViewerIdentityParams{
		TwitchUserID:      claims.User.Id,
		TwitchDisplayName: claims.User.DisplayName,
	}); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Contact the ledger service to create a pending transaction
	imageRequestId := uuid.New()
	alertMetadata := json.RawMessage([]byte(fmt.Sprintf(`{"imageRequestId":"%s"}`, imageRequestId)))
	transaction, err := s.ledger.RequestAlertRedemption(req.Context(), auth.GetToken(req), ImageAlertPointsCost, string(ImageAlertType), &alertMetadata)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ledger.ErrNotEnoughPoints) {
			status = http.StatusConflict
		}
		http.Error(res, err.Error(), status)
		return
	}
	defer transaction.Finalize(req.Context())

	// Record our image generation request in the database, and prepare a function that
	// we can use to record its failure (prior to returning) in the event of any error
	prompt := formatPrompt(payload.Subject)
	if err := s.q.RecordImageRequest(req.Context(), queries.RecordImageRequestParams{
		ImageRequestID:    imageRequestId,
		TwitchUserID:      claims.User.Id,
		SubjectNounClause: payload.Subject,
		Prompt:            prompt,
	}); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	recordFailure := func(err error) error {
		_, dbErr := s.q.RecordImageRequestFailure(req.Context(), queries.RecordImageRequestFailureParams{
			ImageRequestID: imageRequestId,
			ErrorMessage:   err.Error(),
		})
		return dbErr
	}

	// Attempt to generate N images from our prompt, fetching the contents of each PNG
	// concurrently, converting them to JPEG, and buffering their image data in-memory
	generatedImages, err := s.generation.GenerateImages(req.Context(), prompt, NumImagesToGeneratePerPrompt, claims.User.Id)
	if err != nil {
		// Record the request as failed
		if dbErr := recordFailure(err); dbErr != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		// If the API rejected our prompt and refused to generate images, respond with a
		// 400 error
		if errors.Is(err, ErrRejected) {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		// Otherwise, it's a 500 error: something failed on our end
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sanity-check: ensure that we got the requested number of images
	if len(generatedImages) != NumImagesToGeneratePerPrompt {
		err := fmt.Errorf("invalid image generation result: expected to get %d images; got %d", NumImagesToGeneratePerPrompt, len(generatedImages))
		if dbErr := recordFailure(err); dbErr != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Kick off a goroutine for each image that was generated, uploading it to our
	// storage bucket and recording the new image in the database
	var wg errgroup.Group
	imageUrlsChan := make(chan string, len(generatedImages))
	for i := range generatedImages {
		image := &generatedImages[i]
		thunk := getStoreImageFunc(req.Context(), imageRequestId, s.q, s.storage, image, imageUrlsChan)
		wg.Go(thunk)
	}
	if err := wg.Wait(); err != nil {
		if dbErr := recordFailure(err); dbErr != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// If we successfully handled all generated images, mark the image generation
	// request as finished successfully
	if _, err := s.q.RecordImageRequestSuccess(req.Context(), imageRequestId); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate an alert that will display these images onscreen during the stream
	imageUrls := make([]string, 0, len(generatedImages))
	for len(imageUrlsChan) > 0 {
		imageUrls = append(imageUrls, <-imageUrlsChan)
	}
	sort.Strings(imageUrls)
	s.alertsChan <- &alerts.Alert{
		Type: alerts.AlertTypeGeneratedImages,
		Data: alerts.AlertData{
			GeneratedImages: &alerts.AlertDataGeneratedImages{
				Username:    claims.User.DisplayName,
				Description: payload.Subject,
				Urls:        imageUrls,
			},
		},
	}

	// We've successfully generated an alert from the user's request, so finalize the
	// transaction to deduct the points we debited from them - if we don't make it here,
	// our deferred called to transaction.Finalize will reject the transaction instead,
	// causing the debited points to be refunded
	if err := transaction.Accept(req.Context()); err != nil {
		http.Error(res, fmt.Sprintf("failed to finalize transaction: %v", err), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusNoContent)
}

func formatPrompt(subjectNounClause string) string {
	return fmt.Sprintf("a ghostly image of %s, with glitchy VHS artifacts, dark background", subjectNounClause)
}

func formatImageKey(imageRequestId uuid.UUID, index int) string {
	return fmt.Sprintf("%s/%s-%02d.jpg", imageRequestId, imageRequestId, index)
}

func storeImage(ctx context.Context, imageRequestId uuid.UUID, q Queries, storage StorageClient, image *GeneratedImage, imageUrlsChan chan<- string) error {
	// Store the image in our S3-compatible bucket
	key := formatImageKey(imageRequestId, image.index)
	imageUrl, err := storage.Upload(ctx, key, image.contentType, bytes.NewReader(image.data))
	if err != nil {
		return fmt.Errorf("failed to upload generated image to storage: %w", err)
	}
	imageUrlsChan <- imageUrl

	// Record the fact that we've received this generated image
	if err := q.RecordImage(ctx, queries.RecordImageParams{
		ImageRequestID: imageRequestId,
		Index:          int32(image.index),
		Url:            imageUrl,
	}); err != nil {
		return fmt.Errorf("failed to record newly-stored image URL in database: %w", err)
	}
	return nil
}

func getStoreImageFunc(ctx context.Context, imageRequestId uuid.UUID, q Queries, storage StorageClient, image *GeneratedImage, imageUrlsChan chan<- string) func() error {
	return func() error {
		return storeImage(ctx, imageRequestId, q, storage, image, imageUrlsChan)
	}
}
