package imagegen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"net/http"
	"sort"

	openai "github.com/sashabaranov/go-openai"
	"golang.org/x/sync/errgroup"
)

type GeneratedImage struct {
	index       int
	contentType string
	data        []byte
}

type GenerationClient interface {
	GenerateImages(ctx context.Context, prompt string, numImages int, opaqueUserId string) ([]GeneratedImage, error)
}

type generationClient struct {
	c             *openai.Client
	convertToJpeg bool
	jpegQuality   int
}

func NewGenerationClient(openaiToken string) GenerationClient {
	return &generationClient{
		c:             openai.NewClient(openaiToken),
		convertToJpeg: true,
		jpegQuality:   80,
	}
}

func (c *generationClient) GenerateImages(ctx context.Context, prompt string, numImages int, opaqueUserId string) ([]GeneratedImage, error) {
	// Send a request to the OpenAI API to generate the desired number of images from
	// our prompt: this request will block until all images are ready
	res, err := c.c.CreateImage(ctx, openai.ImageRequest{
		Prompt:         prompt,
		Model:          openai.CreateImageModelDallE3,
		N:              numImages,
		Quality:        openai.CreateImageQualityStandard,
		Size:           openai.CreateImageSize1024x1024,
		Style:          openai.CreateImageStyleVivid,
		ResponseFormat: openai.CreateImageResponseFormatURL,
		User:           opaqueUserId,
	})
	if err != nil {
		// If our request was rejected with a 400 error, return ErrRejected so the
		// caller can propagate it as a client-level error
		apiError := &openai.APIError{}
		if errors.As(err, &apiError) && apiError.HTTPStatusCode == http.StatusBadRequest && apiError.Type == "invalid_request_error" {
			return nil, &rejectionError{apiError.Message}
		}
		return nil, err
	}

	// Kick off a goroutine for each image that was generated, downloading the PNG image
	// data, and buffering it in a channel
	jpegConversionQuality := 0
	if c.convertToJpeg {
		jpegConversionQuality = c.jpegQuality
	}
	imagesChan := make(chan GeneratedImage, numImages)
	wg, ctx := errgroup.WithContext(ctx)
	for index, data := range res.Data {
		thunk := getFetchImageDataFunc(ctx, index, data.URL, jpegConversionQuality, imagesChan)
		wg.Go(thunk)
	}
	if err := wg.Wait(); err != nil {
		return nil, err
	}

	// Read all items from the channel and return a sorted array of GeneratedImage
	// structs, each of which contains the image data buffered in-memory
	images := make([]GeneratedImage, 0, numImages)
	for len(imagesChan) > 0 {
		images = append(images, <-imagesChan)
	}
	sort.Slice(images, func(i, j int) bool { return images[i].index < images[j].index })
	return images, nil
}

// fetchImageData downloads the image at the given URL, then writes the raw bytes of
// that image data to the provided channel
func fetchImageData(ctx context.Context, index int, url string, jpegConversionQuality int, images chan<- GeneratedImage) error {
	// Download the OpenAI-hosted PNG image so we can store it permanently
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("got status %d from request for OpenAI-hosted image", res.StatusCode)
	}

	// Verify that OpenAI has linked us to a .png
	contentType := res.Header.Get("content-type")
	if contentType != "image/png" {
		return fmt.Errorf("got unexpected content-type '%s' for OpenAI-hosted image", contentType)
	}

	// Decode the PNG, reading it directly from the response body
	bmpData, err := png.Decode(res.Body)
	if err != nil {
		return fmt.Errorf("failed to decode PNG data for OpenAI-hosted image: %w", err)
	}

	// Preallocate a buffer that's roughly as large as the largest 1024x1024 JPEG
	// we can reasonably expect to produce, then write our compressed JPEG data into
	// it
	jpegBuffer := bytes.NewBuffer(make([]byte, 0, 512*1024))
	if err := jpeg.Encode(jpegBuffer, bmpData, &jpeg.Options{Quality: jpegConversionQuality}); err != nil {
		return fmt.Errorf("failed to encode JPEG image from decoded PNG image: %w", err)
	}

	// Write our image data into a channel (since we're potentially fetching multiple
	// images in parallel), and we're done fetching this image
	images <- GeneratedImage{
		index:       index,
		contentType: "image/jpeg",
		data:        jpegBuffer.Bytes(),
	}
	return nil
}

// getFetchImageDataFunc returns a thunk that will invoke fetchImageData with the given
// set of arguments. This is a workaround for a common pitfall re: attempting to enclose
// loop variables in a goroutine, which should be fixed in Go 1.22:
// https://go.dev/blog/loopvar-preview
func getFetchImageDataFunc(ctx context.Context, index int, url string, jpegConversionQuality int, images chan<- GeneratedImage) func() error {
	return func() error {
		return fetchImageData(ctx, index, url, jpegConversionQuality, images)
	}
}
