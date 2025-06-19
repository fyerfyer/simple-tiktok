package media

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"io"

	"github.com/disintegration/imaging"
)

// ThumbnailGenerator 缩略图生成器
type ThumbnailGenerator struct {
	width     int
	height    int
	quality   int
	processor VideoProcessorInterface
}

// NewThumbnailGenerator 创建缩略图生成器
func NewThumbnailGenerator(width, height, quality int, processor VideoProcessorInterface) *ThumbnailGenerator {
	if width <= 0 {
		width = 480
	}
	if height <= 0 {
		height = 270
	}
	if quality <= 0 || quality > 100 {
		quality = 80
	}

	return &ThumbnailGenerator{
		width:     width,
		height:    height,
		quality:   quality,
		processor: processor,
	}
}

// GenerateFromVideo 从视频生成缩略图
func (t *ThumbnailGenerator) GenerateFromVideo(ctx context.Context, videoReader io.Reader, seekTime int64) (io.Reader, error) {
	if t.processor == nil {
		return t.GenerateDefault(ctx)
	}

	var buf bytes.Buffer
	opts := &ProcessorOptions{
		Width:    t.width,
		Height:   t.height,
		SeekTime: seekTime,
		Format:   "jpg",
		Quality:  t.quality,
	}

	err := t.processor.GenerateThumbnail(ctx, videoReader, &buf, opts)
	if err != nil {
		// 如果生成失败，返回默认缩略图
		return t.GenerateDefault(ctx)
	}

	return &buf, nil
}

// GenerateDefault 生成默认缩略图
func (t *ThumbnailGenerator) GenerateDefault(ctx context.Context) (io.Reader, error) {
	// 使用imaging库创建渐变背景
	img := imaging.New(t.width, t.height, color.NRGBA{200, 200, 200, 255})

	// 添加简单的渐变效果
	for y := 0; y < t.height; y++ {
		for x := 0; x < t.width; x++ {
			grayValue := uint8(200 - (y * 50 / t.height))
			grayColor := color.NRGBA{R: grayValue, G: grayValue, B: grayValue, A: 255}
			img.Set(x, y, grayColor)
		}
	}

	var buf bytes.Buffer
	err := imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(t.quality))
	if err != nil {
		return nil, fmt.Errorf("encode default thumbnail failed: %w", err)
	}

	return &buf, nil
}

// GenerateFromImage 从图片生成缩略图
func (t *ThumbnailGenerator) GenerateFromImage(ctx context.Context, imageReader io.Reader) (io.Reader, error) {
	img, err := imaging.Decode(imageReader)
	if err != nil {
		return nil, fmt.Errorf("decode image failed: %w", err)
	}

	// 使用imaging库的高质量缩放
	thumbnail := imaging.Resize(img, t.width, t.height, imaging.Lanczos)

	var buf bytes.Buffer
	err = imaging.Encode(&buf, thumbnail, imaging.JPEG, imaging.JPEGQuality(t.quality))
	if err != nil {
		return nil, fmt.Errorf("encode thumbnail failed: %w", err)
	}

	return &buf, nil
}

// GetSize 获取缩略图尺寸
func (t *ThumbnailGenerator) GetSize() (int, int) {
	return t.width, t.height
}

// SetQuality 设置质量
func (t *ThumbnailGenerator) SetQuality(quality int) {
	if quality > 0 && quality <= 100 {
		t.quality = quality
	}
}
