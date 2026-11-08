package thumbnail

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"io"

	"github.com/disintegration/imaging"
)

type imagingGenerator struct {
	width  int
	height int
}

func NewImagingGenerator(width, height int) Generator {
	return &imagingGenerator{width: width, height: height}
}

func (g *imagingGenerator) Generate(ctx context.Context, r io.Reader, mimeType string) (*GenerateResult, error) {
	// 1. Decode original image
	src, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("thumbnail.Generate: failed to decode image: %w", err)
	}

	// 2. Resize and crop to fill width x height while preserving aspect ratio
	thumb := imaging.Fill(src, g.width, g.height, imaging.Center, imaging.Lanczos)

	// 3. Encode to JPEG into buffer
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, thumb, &jpeg.Options{Quality: 80}); err != nil {
		return nil, fmt.Errorf("thumbnail.Generate: failed to encode jpeg: %w", err)
	}

	return &GenerateResult{
		Reader:      &buf,
		Size:        int64(buf.Len()),
		ContentType: "image/jpeg",
	}, nil
}
