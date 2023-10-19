package queries_test

import (
	"context"
	"testing"

	"github.com/golden-vcr/server-common/querytest"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_RecordImageRequest(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.image_request")

	_, err := tx.Exec(`
		INSERT INTO showtime.viewer (twitch_user_id, twitch_display_name)
			VALUES ('1005', 'Joe')
	`)
	assert.NoError(t, err)

	err = q.RecordImageRequest(context.Background(), queries.RecordImageRequestParams{
		ImageRequestID:    uuid.MustParse("5e3a831b-699e-45f2-9587-048cbaeaf17d"),
		TwitchUserID:      "1005",
		SubjectNounClause: "a scary clown",
		Prompt:            "an image of a scary clown, dark background",
	})
	assert.NoError(t, err)

	querytest.AssertCount(t, tx, 1, `
		SELECT COUNT(*) FROM showtime.image_request
			WHERE id = '5e3a831b-699e-45f2-9587-048cbaeaf17d'
			AND twitch_user_id = '1005'
			AND subject_noun_clause = 'a scary clown'
			AND prompt = 'an image of a scary clown, dark background'
			AND created_at IS NOT NULL
			AND finished_at IS NULL
			AND error_message IS NULL
	`)
}

func Test_RecordImageRequestFailure(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.image_request")

	_, err := tx.Exec(`
		INSERT INTO showtime.viewer (twitch_user_id, twitch_display_name)
			VALUES ('2006', 'Abby')
	`)
	assert.NoError(t, err)

	err = q.RecordImageRequest(context.Background(), queries.RecordImageRequestParams{
		ImageRequestID:    uuid.MustParse("8071fb37-8318-4eec-a479-5b329d2fb6a9"),
		TwitchUserID:      "2006",
		SubjectNounClause: "several geese",
		Prompt:            "an image of several geese, dark background",
	})
	assert.NoError(t, err)

	res, err := q.RecordImageRequestFailure(context.Background(), queries.RecordImageRequestFailureParams{
		ImageRequestID: uuid.MustParse("8071fb37-8318-4eec-a479-5b329d2fb6a9"),
		ErrorMessage:   "something went wrong",
	})
	assert.NoError(t, err)
	numRows, err := res.RowsAffected()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), numRows)

	querytest.AssertCount(t, tx, 1, `
		SELECT COUNT(*) FROM showtime.image_request
			WHERE id = '8071fb37-8318-4eec-a479-5b329d2fb6a9'
			AND twitch_user_id = '2006'
			AND subject_noun_clause = 'several geese'
			AND prompt = 'an image of several geese, dark background'
			AND created_at IS NOT NULL
			AND finished_at IS NOT NULL
			AND error_message = 'something went wrong'
	`)

	// Attempting to record a result for an image_request that's already finished should
	// affect 0 rows
	res, err = q.RecordImageRequestFailure(context.Background(), queries.RecordImageRequestFailureParams{
		ImageRequestID: uuid.MustParse("8071fb37-8318-4eec-a479-5b329d2fb6a9"),
		ErrorMessage:   "a different thing went wrong, like, again",
	})
	assert.NoError(t, err)
	numRows, err = res.RowsAffected()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), numRows)

	// Attempting to record a result for an image_request with an invalid uuid should
	// affect 0 rows
	res, err = q.RecordImageRequestFailure(context.Background(), queries.RecordImageRequestFailureParams{
		ImageRequestID: uuid.MustParse("02448cd2-0663-47bd-bc5a-0296bcd27fff"),
		ErrorMessage:   "oh no",
	})
	assert.NoError(t, err)
	numRows, err = res.RowsAffected()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), numRows)
}

func Test_RecordImageRequestSuccess(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.image_request")

	_, err := tx.Exec(`
		INSERT INTO showtime.viewer (twitch_user_id, twitch_display_name)
			VALUES ('3007', 'Reginald')
	`)
	assert.NoError(t, err)

	err = q.RecordImageRequest(context.Background(), queries.RecordImageRequestParams{
		ImageRequestID:    uuid.MustParse("5e6115ea-d7ac-44aa-81a0-17a715bc984d"),
		TwitchUserID:      "3007",
		SubjectNounClause: "a platypus playing the saxaphone",
		Prompt:            "an image of a platypus playing the saxaphone, dark background",
	})
	assert.NoError(t, err)

	res, err := q.RecordImageRequestSuccess(context.Background(), uuid.MustParse("5e6115ea-d7ac-44aa-81a0-17a715bc984d"))
	assert.NoError(t, err)
	numRows, err := res.RowsAffected()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), numRows)

	querytest.AssertCount(t, tx, 1, `
		SELECT COUNT(*) FROM showtime.image_request
			WHERE id = '5e6115ea-d7ac-44aa-81a0-17a715bc984d'
			AND twitch_user_id = '3007'
			AND subject_noun_clause = 'a platypus playing the saxaphone'
			AND prompt = 'an image of a platypus playing the saxaphone, dark background'
			AND created_at IS NOT NULL
			AND finished_at IS NOT NULL
			AND error_message IS NULL
	`)

	// Attempting to record a result for an image_request that's already finished should
	// affect 0 rows
	res, err = q.RecordImageRequestSuccess(context.Background(), uuid.MustParse("5e6115ea-d7ac-44aa-81a0-17a715bc984d"))
	assert.NoError(t, err)
	numRows, err = res.RowsAffected()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), numRows)

	// Attempting to record a result for an image_request with an invalid uuid should
	// affect 0 rows
	res, err = q.RecordImageRequestSuccess(context.Background(), uuid.MustParse("1c98937b-406d-4358-aec5-b69edd460394"))
	assert.NoError(t, err)
	numRows, err = res.RowsAffected()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), numRows)
}

func Test_RecordImage(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.image_request")
	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.image")

	_, err := tx.Exec(`
		INSERT INTO showtime.viewer (twitch_user_id, twitch_display_name)
			VALUES ('4444', 'greasyJim')
	`)
	assert.NoError(t, err)

	err = q.RecordImageRequest(context.Background(), queries.RecordImageRequestParams{
		ImageRequestID:    uuid.MustParse("dfaf425a-17fa-4bf1-b49b-74ce354deb6f"),
		TwitchUserID:      "4444",
		SubjectNounClause: "a juicy hamburger",
		Prompt:            "an image of a juicy hamburger, dark background",
	})
	assert.NoError(t, err)

	err = q.RecordImage(context.Background(), queries.RecordImageParams{
		ImageRequestID: uuid.MustParse("dfaf425a-17fa-4bf1-b49b-74ce354deb6f"),
		Index:          0,
		Url:            "http://example.com/my-cool-image.png",
	})
	assert.NoError(t, err)

	querytest.AssertCount(t, tx, 1, `
		SELECT COUNT(*) FROM showtime.image
			WHERE image_request_id = 'dfaf425a-17fa-4bf1-b49b-74ce354deb6f'
			AND index = 0
			AND url = 'http://example.com/my-cool-image.png'
	`)
}
