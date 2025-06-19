package media

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	ffprobe "gopkg.in/vansante/go-ffprobe.v2"
)

// VideoMetadata 视频元数据
type VideoMetadata struct {
	Duration    float64 `json:"duration"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Size        int64   `json:"size"`
	Format      string  `json:"format"`
	Filename    string  `json:"filename"`
	Bitrate     string  `json:"bitrate"` // 改为string类型
	Framerate   string  `json:"framerate"`
	CodecName   string  `json:"codec_name"`
	AspectRatio string  `json:"aspect_ratio"`
}

// VideoProcessor 视频处理器
type VideoProcessor struct {
	maxFileSize      int64
	supportedFormats []string
	thumbnailWidth   int
	thumbnailHeight  int
	thumbnailQuality int
	log              *log.Helper
}

// NewVideoProcessor 创建视频处理器
func NewVideoProcessor(maxFileSize int64, supportedFormats []string, thumbWidth, thumbHeight, thumbQuality int) *VideoProcessor {
	return &VideoProcessor{
		maxFileSize:      maxFileSize,
		supportedFormats: supportedFormats,
		thumbnailWidth:   thumbWidth,
		thumbnailHeight:  thumbHeight,
		thumbnailQuality: thumbQuality,
		log:              log.NewHelper(log.GetLogger()),
	}
}

// ValidateFormat 验证视频格式
func (vp *VideoProcessor) ValidateFormat(filename string, size int64) error {
	if size > vp.maxFileSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", size, vp.maxFileSize)
	}

	if size <= 0 {
		return fmt.Errorf("invalid file size: %d", size)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	contentType := vp.getContentTypeByExt(ext)

	for _, format := range vp.supportedFormats {
		if format == contentType {
			return nil
		}
	}

	return fmt.Errorf("unsupported video format: %s", ext)
}

// ValidateVideoFile 验证视频文件内容
func (vp *VideoProcessor) ValidateVideoFile(ctx context.Context, reader io.Reader) error {
	header := make([]byte, 512)
	n, err := reader.Read(header)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file header: %w", err)
	}

	if n < 4 {
		return fmt.Errorf("file too small")
	}

	if !vp.isValidVideoHeader(header[:n]) {
		return fmt.Errorf("invalid video file format")
	}

	return nil
}

// GetMetadata 获取视频元数据
func (vp *VideoProcessor) GetMetadata(ctx context.Context, reader io.Reader) (*VideoMetadata, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	data, err := ffprobe.ProbeReader(ctx, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to probe video: %w", err)
	}

	var videoStream *ffprobe.Stream
	for _, stream := range data.Streams {
		if stream.CodecType == "video" {
			videoStream = stream
			break
		}
	}

	if videoStream == nil {
		return nil, fmt.Errorf("no video stream found")
	}

	// 转换Size字段
	var size int64
	if data.Format.Size != "" {
		if s, err := strconv.ParseInt(data.Format.Size, 10, 64); err == nil {
			size = s
		}
	}

	metadata := &VideoMetadata{
		Duration:    data.Format.DurationSeconds,
		Width:       videoStream.Width,
		Height:      videoStream.Height,
		Size:        size,
		Format:      data.Format.FormatName,
		Bitrate:     data.Format.BitRate, // 保持string类型
		CodecName:   videoStream.CodecName,
		AspectRatio: videoStream.DisplayAspectRatio,
	}

	if videoStream.RFrameRate != "" {
		metadata.Framerate = videoStream.RFrameRate
	} else if videoStream.AvgFrameRate != "" {
		metadata.Framerate = videoStream.AvgFrameRate
	}

	return metadata, nil
}

// GetMetadataFromFile 从文件获取视频元数据
func (vp *VideoProcessor) GetMetadataFromFile(ctx context.Context, filename string) (*VideoMetadata, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	data, err := ffprobe.ProbeURL(ctx, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to probe video file: %w", err)
	}

	var videoStream *ffprobe.Stream
	for _, stream := range data.Streams {
		if stream.CodecType == "video" {
			videoStream = stream
			break
		}
	}

	if videoStream == nil {
		return nil, fmt.Errorf("no video stream found")
	}

	var size int64
	if data.Format.Size != "" {
		if s, err := strconv.ParseInt(data.Format.Size, 10, 64); err == nil {
			size = s
		}
	}

	metadata := &VideoMetadata{
		Duration:    data.Format.DurationSeconds,
		Width:       videoStream.Width,
		Height:      videoStream.Height,
		Size:        size,
		Format:      data.Format.FormatName,
		Filename:    filepath.Base(filename),
		Bitrate:     data.Format.BitRate,
		CodecName:   videoStream.CodecName,
		AspectRatio: videoStream.DisplayAspectRatio,
	}

	if videoStream.RFrameRate != "" {
		metadata.Framerate = videoStream.RFrameRate
	} else if videoStream.AvgFrameRate != "" {
		metadata.Framerate = videoStream.AvgFrameRate
	}

	return metadata, nil
}

// IsValidVideo 检查是否为有效视频文件
func (vp *VideoProcessor) IsValidVideo(ctx context.Context, reader io.Reader, filename string, size int64) (bool, error) {
	if err := vp.ValidateFormat(filename, size); err != nil {
		return false, err
	}

	_, err := vp.GetMetadata(ctx, reader)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetVideoInfo 获取视频基本信息
func (vp *VideoProcessor) GetVideoInfo(filename string, size int64) (*VideoMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metadata, err := vp.GetMetadataFromFile(ctx, filename)
	if err != nil {
		return nil, err
	}

	metadata.Size = size
	return metadata, nil
}

// GenerateDefaultThumbnail 生成默认缩略图
func (vp *VideoProcessor) GenerateDefaultThumbnail(ctx context.Context) (io.Reader, error) {
	img := image.NewRGBA(image.Rect(0, 0, vp.thumbnailWidth, vp.thumbnailHeight))

	// 创建颜色
	grayColor := color.RGBA{R: 200, G: 200, B: 200, A: 255}

	// 填充默认颜色
	for y := 0; y < vp.thumbnailHeight; y++ {
		for x := 0; x < vp.thumbnailWidth; x++ {
			img.Set(x, y, grayColor)
		}
	}

	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: vp.thumbnailQuality})
	if err != nil {
		return nil, fmt.Errorf("failed to encode default thumbnail: %w", err)
	}

	return &buf, nil
}

// GetVideoQuality 获取视频质量等级
func (vp *VideoProcessor) GetVideoQuality(width, height int) string {
	if width >= 1920 && height >= 1080 {
		return "HD"
	} else if width >= 1280 && height >= 720 {
		return "SD"
	} else {
		return "LD"
	}
}

// CalculateVideoBitrate 计算建议的视频比特率
func (vp *VideoProcessor) CalculateVideoBitrate(width, height int, duration float64) int64 {
	pixels := int64(width * height)

	var baseBitrate int64
	if pixels >= 1920*1080 {
		baseBitrate = 5000000
	} else if pixels >= 1280*720 {
		baseBitrate = 3000000
	} else {
		baseBitrate = 1500000
	}

	return baseBitrate
}

// EstimateProcessingTime 估算视频处理时间
func (vp *VideoProcessor) EstimateProcessingTime(size int64) time.Duration {
	sizeInMB := size / (1024 * 1024)
	if sizeInMB < 1 {
		sizeInMB = 1
	}

	return time.Duration(sizeInMB) * time.Second
}

// ValidateVideoTitle 验证视频标题
func (vp *VideoProcessor) ValidateVideoTitle(title string) error {
	if len(title) == 0 {
		return fmt.Errorf("video title cannot be empty")
	}

	if len(title) > 100 {
		return fmt.Errorf("video title too long, max 100 characters")
	}

	if strings.Contains(title, "<") || strings.Contains(title, ">") {
		return fmt.Errorf("video title contains invalid characters")
	}

	return nil
}

// isValidVideoHeader 检查文件头部是否为有效视频格式
func (vp *VideoProcessor) isValidVideoHeader(header []byte) bool {
	if len(header) < 4 {
		return false
	}

	if len(header) >= 8 {
		if bytes.Contains(header[:32], []byte("ftyp")) {
			return true
		}
		if bytes.Contains(header[:12], []byte("ftypmp4")) ||
			bytes.Contains(header[:12], []byte("ftypisom")) ||
			bytes.Contains(header[:12], []byte("ftypM4V")) {
			return true
		}
	}

	if len(header) >= 12 {
		if bytes.Equal(header[:4], []byte("RIFF")) && bytes.Equal(header[8:12], []byte("AVI ")) {
			return true
		}
	}

	if len(header) >= 8 {
		if bytes.Contains(header[:20], []byte("ftypqt")) ||
			bytes.Contains(header[:20], []byte("moov")) {
			return true
		}
	}

	return false
}

// getContentTypeByExt 根据扩展名获取内容类型
func (vp *VideoProcessor) getContentTypeByExt(ext string) string {
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".avi":
		return "video/avi"
	case ".mov":
		return "video/quicktime"
	case ".mkv":
		return "video/x-matroska"
	case ".flv":
		return "video/x-flv"
	case ".wmv":
		return "video/x-ms-wmv"
	case ".webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

// GetSupportedFormats 获取支持的视频格式
func (vp *VideoProcessor) GetSupportedFormats() []string {
	return vp.supportedFormats
}

// GetMaxFileSize 获取最大文件大小限制
func (vp *VideoProcessor) GetMaxFileSize() int64 {
	return vp.maxFileSize
}
