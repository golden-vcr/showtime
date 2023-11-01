package imagegen

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/golden-vcr/auth"
	authmock "github.com/golden-vcr/auth/mock"
	ledgermock "github.com/golden-vcr/ledger/mock"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/golden-vcr/showtime/internal/alerts"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_Server_handleRequest(t *testing.T) {
	tests := []struct {
		name                       string
		q                          *mockQueries
		initialBalance             int
		generation                 *mockGenerationClient
		storage                    *mockStorageClient
		body                       string
		wantStatus                 int
		wantBody                   string
		wantViewerIdentityRecorded bool
		wantRequestRecorded        bool
		wantRequestSucceeded       bool
		wantNumImagesRecorded      int
		wantImageDataStored        [][]byte
		wantAlert                  *alerts.AlertDataGeneratedImages
	}{
		{
			"subject is required",
			&mockQueries{},
			1000,
			&mockGenerationClient{},
			&mockStorageClient{},
			`{}`,
			http.StatusBadRequest,
			"invalid request payload: 'subject' is required",
			false,
			false,
			false,
			0,
			nil,
			nil,
		},
		{
			"normal usage",
			&mockQueries{},
			1000,
			&mockGenerationClient{},
			&mockStorageClient{},
			`{"subject":"a seal"}`,
			http.StatusNoContent,
			"",
			true,
			true,
			true,
			4,
			[][]byte{
				generateMockImageData(0),
				generateMockImageData(1),
				generateMockImageData(2),
				generateMockImageData(3),
			},
			&alerts.AlertDataGeneratedImages{
				Username:    "Jerry",
				Description: "a seal",
				Urls: []string{
					generateMockStorageUrl("*-01.jpg"),
					generateMockStorageUrl("*-02.jpg"),
					generateMockStorageUrl("*-03.jpg"),
					generateMockStorageUrl("*-04.jpg"),
				},
			},
		},
		{
			"insufficient point balance is propagated as 409 error",
			&mockQueries{},
			0,
			&mockGenerationClient{},
			&mockStorageClient{},
			`{"subject":"a seal"}`,
			http.StatusConflict,
			"not enough points",
			true,
			false,
			false,
			0,
			nil,
			nil,
		},
		{
			"rejection from image generation API is propagated as 400 error",
			&mockQueries{},
			1000,
			&mockGenerationClient{
				err: &rejectionError{"objectionable content detected"},
			},
			&mockStorageClient{},
			`{"subject":"something objectionable"}`,
			http.StatusBadRequest,
			"image generation request rejected: objectionable content detected",
			true,
			true,
			false,
			0,
			nil,
			nil,
		},
		{
			"other errors in image generation are 500 errors",
			&mockQueries{},
			1000,
			&mockGenerationClient{
				err: fmt.Errorf("mock error"),
			},
			&mockStorageClient{},
			`{"subject":"a seal"}`,
			http.StatusInternalServerError,
			"mock error",
			true,
			true,
			false,
			0,
			nil,
			nil,
		},
		{
			"storage errors are 500 errors",
			&mockQueries{},
			1000,
			&mockGenerationClient{},
			&mockStorageClient{
				err: fmt.Errorf("mock error"),
			},
			`{"subject":"a seal"}`,
			http.StatusInternalServerError,
			"failed to upload generated image to storage: mock error",
			true,
			true,
			false,
			0,
			nil,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alertsChan := make(chan *alerts.Alert, 8)
			s := &Server{
				q:          tt.q,
				ledger:     ledgermock.NewClient().Grant("mock-token", tt.initialBalance),
				generation: tt.generation,
				storage:    tt.storage,
				alertsChan: alertsChan,
			}
			handler := auth.RequireAccess(
				authmock.NewClient().Allow("mock-token", auth.RoleViewer, auth.UserDetails{
					Id:          "54321",
					Login:       "jerry",
					DisplayName: "Jerry",
				}), auth.RoleViewer,
				http.HandlerFunc(s.handleRequest),
			)
			req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(tt.body))
			req.Header.Set("authorization", "mock-token")
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			// Verify expected body and status code
			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			body := strings.TrimSuffix(string(b), "\n")
			assert.Equal(t, tt.wantStatus, res.Code)
			assert.Equal(t, tt.wantBody, body)

			// Verify expected database operations
			if tt.wantViewerIdentityRecorded {
				assert.Len(t, tt.q.recordViewerIdentityCalls, 1)
			} else {
				assert.Len(t, tt.q.recordViewerIdentityCalls, 0)
			}
			if tt.wantRequestRecorded {
				assert.Len(t, tt.q.recordImageRequestCalls, 1)
				if tt.wantRequestSucceeded {
					assert.Len(t, tt.q.recordImageRequestSuccessCalls, 1)
					assert.Len(t, tt.q.recordImageRequestFailureCalls, 0)
				} else {
					assert.Len(t, tt.q.recordImageRequestSuccessCalls, 0)
					assert.Len(t, tt.q.recordImageRequestFailureCalls, 1)
				}
			} else {
				assert.Len(t, tt.q.recordImageRequestCalls, 0)
				assert.Len(t, tt.q.recordImageRequestSuccessCalls, 0)
				assert.Len(t, tt.q.recordImageRequestFailureCalls, 0)
			}
			assert.Len(t, tt.q.recordImageCalls, tt.wantNumImagesRecorded)

			// Verify that images were uploaded to storage
			imageData := make([][]byte, 0)
			for _, image := range tt.storage.imagesUploaded {
				assert.Equal(t, "image/jpeg", image.contentType)
				imageData = append(imageData, image.data)
			}
			assert.ElementsMatch(t, tt.wantImageDataStored, imageData)

			// Verify that an alert was produced if expected
			if tt.wantAlert != nil {
				assert.Len(t, alertsChan, 1)
				alert := <-alertsChan
				assert.Equal(t, alerts.AlertTypeGeneratedImages, alert.Type)
				assert.Equal(t, tt.wantAlert.Username, alert.Data.GeneratedImages.Username)
				assert.Equal(t, tt.wantAlert.Description, alert.Data.GeneratedImages.Description)
				assert.Len(t, alert.Data.GeneratedImages.Urls, len(tt.wantAlert.Urls))
			} else {
				assert.Len(t, alertsChan, 0)
			}
		})
	}
}

// mockQueries implements imagegen.Queries for testing
type mockQueries struct {
	recordViewerIdentityCalls      []queries.RecordViewerIdentityParams
	recordImageRequestCalls        []queries.RecordImageRequestParams
	recordImageRequestFailureCalls []queries.RecordImageRequestFailureParams
	recordImageRequestSuccessCalls []uuid.UUID
	recordImageCalls               []queries.RecordImageParams
}

type mockSqlResult struct {
	numRowsAffected int64
}

func (r mockSqlResult) LastInsertId() (int64, error) {
	return -1, fmt.Errorf("not supported")
}

func (r mockSqlResult) RowsAffected() (int64, error) {
	return r.numRowsAffected, nil
}

func (m *mockQueries) RecordViewerIdentity(ctx context.Context, arg queries.RecordViewerIdentityParams) error {
	m.recordViewerIdentityCalls = append(m.recordViewerIdentityCalls, arg)
	return nil
}

func (m *mockQueries) RecordImageRequest(ctx context.Context, arg queries.RecordImageRequestParams) error {
	m.recordImageRequestCalls = append(m.recordImageRequestCalls, arg)
	return nil
}

func (m *mockQueries) RecordImageRequestFailure(ctx context.Context, arg queries.RecordImageRequestFailureParams) (sql.Result, error) {
	if !m.hasRecordedImageRequest(arg.ImageRequestID) {
		return mockSqlResult{0}, nil
	}
	m.recordImageRequestFailureCalls = append(m.recordImageRequestFailureCalls, arg)
	return mockSqlResult{1}, nil
}

func (m *mockQueries) RecordImageRequestSuccess(ctx context.Context, imageRequestID uuid.UUID) (sql.Result, error) {
	if !m.hasRecordedImageRequest(imageRequestID) {
		return mockSqlResult{0}, nil
	}
	m.recordImageRequestSuccessCalls = append(m.recordImageRequestSuccessCalls, imageRequestID)
	return mockSqlResult{1}, nil
}

func (m *mockQueries) RecordImage(ctx context.Context, arg queries.RecordImageParams) error {
	m.recordImageCalls = append(m.recordImageCalls, arg)
	return nil
}

func (m *mockQueries) hasRecordedImageRequest(imageRequestID uuid.UUID) bool {
	for _, call := range m.recordImageRequestCalls {
		if call.ImageRequestID == imageRequestID {
			return true
		}
	}
	return false
}

// mockGenerationClient implements imagegen.GenerationClient for testing
type mockGenerationClient struct {
	err error
}

func (m *mockGenerationClient) GenerateImages(ctx context.Context, prompt string, numImages int, opaqueUserId string) ([]GeneratedImage, error) {
	if m.err != nil {
		return nil, m.err
	}
	images := make([]GeneratedImage, 0, numImages)
	for index := 0; index < numImages; index++ {
		images = append(images, GeneratedImage{
			index:       index,
			contentType: "image/jpeg",
			data:        generateMockImageData(index),
		})
	}
	return images, nil
}

// mockStorageClient implements imagegen.StorageClient for testing
type mockStorageClient struct {
	err            error
	imagesUploaded []struct {
		contentType string
		data        []byte
	}
	mu sync.Mutex
}

func (m *mockStorageClient) Upload(ctx context.Context, key string, contentType string, data io.ReadSeeker) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return "", m.err
	}
	b, err := io.ReadAll(data)
	if err != nil {
		return "", err
	}
	m.imagesUploaded = append(m.imagesUploaded, struct {
		contentType string
		data        []byte
	}{
		contentType: contentType,
		data:        b,
	})
	return generateMockStorageUrl(key), nil
}

// generateMockImageData returns dummy data to represent the image data generated from a
// specific prompt: the result is not valid image data, but it can be passed around in
// tests as a placeholder for that data
func generateMockImageData(index int) []byte {
	s := fmt.Sprintf("mock image data for index %d", index)
	return []byte(s)
}

// generateMockStorageUrl returns a fake URL where a file with the given key should be
// stored
func generateMockStorageUrl(key string) string {
	return fmt.Sprintf("http://my-cool-storage-bucket.biz/%s", key)
}
