package auth

import (
	"context"

	"github.com/golden-vcr/showtime/gen/queries"
)

type Queries interface {
	RecordViewerLogin(ctx context.Context, arg queries.RecordViewerLoginParams) error
}
