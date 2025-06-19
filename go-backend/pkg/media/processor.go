package media

import (
	"context"
	"io"
)

// ProcessorOptions 处理器选项
type ProcessorOptions struct {
	Width    int
	Height   int
	SeekTime int64  // 截取时间点（秒）
	Format   string // 输出格式
	Quality  int    // 质量设置
}

// VideoProcessorInterface 视频处理器接口
type VideoProcessorInterface interface {
	// 生成缩略图
	GenerateThumbnail(ctx context.Context, input io.Reader, output io.Writer, opts *ProcessorOptions) error

	// 视频转码
	TranscodeVideo(ctx context.Context, input io.Reader, output io.Writer, opts *ProcessorOptions) error

	// 获取视频元信息
	GetVideoInfo(ctx context.Context, input io.Reader) (*VideoMetadata, error)
}

// TranscodeOptions 转码选项
type TranscodeOptions struct {
	VideoCodec string
	AudioCodec string
	Bitrate    int
	Resolution string
	Format     string
	Preset     string
	CRF        int
}

// ThumbnailOptions 缩略图选项
type ThumbnailOptions struct {
	Width    int
	Height   int
	SeekTime int64
	Format   string
	Quality  int
}
