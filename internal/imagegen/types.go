package imagegen

import (
	"context"
	"database/sql"

	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/google/uuid"
)

// Queries represents the subset of database functionality required to handle image
// generation requests
type Queries interface {
	GetCurrentScreening(ctx context.Context) (queries.GetCurrentScreeningRow, error)
	RecordViewerIdentity(ctx context.Context, arg queries.RecordViewerIdentityParams) error
	RecordImageRequest(ctx context.Context, arg queries.RecordImageRequestParams) error
	RecordImageRequestFailure(ctx context.Context, arg queries.RecordImageRequestFailureParams) (sql.Result, error)
	RecordImageRequestSuccess(ctx context.Context, imageRequestID uuid.UUID) (sql.Result, error)
	RecordImage(ctx context.Context, arg queries.RecordImageParams) error
}

// Request is a user-submitted request to generate one or more images with a given
// subject, to be overlaid on the video during a stream
type Request struct {
	Subject string `json:"subject"`
}
