package media

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/disintegration/imaging"
	"github.com/go-kratos/kratos/v2/log"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

// FFmpegProcessor FFmpeg处理器实现
type FFmpegProcessor struct {
	tempDir string
	log     *log.Helper
}

// NewFFmpegProcessor 创建FFmpeg处理器
func NewFFmpegProcessor(tempDir string) *FFmpegProcessor {
	return &FFmpegProcessor{
		tempDir: tempDir,
		log:     log.NewHelper(log.GetLogger()),
	}
}

// GenerateThumbnail 生成缩略图
func (f *FFmpegProcessor) GenerateThumbnail(ctx context.Context, input io.Reader, output io.Writer, opts *ProcessorOptions) error {
	if opts == nil {
		opts = &ProcessorOptions{
			Width:    480,
			Height:   270,
			SeekTime: 5,
			Format:   "jpg",
			Quality:  80,
		}
	}

	// 创建临时文件
	inputFile, err := f.createTempFile(input, "input")
	if err != nil {
		return fmt.Errorf("create temp input file failed: %w", err)
	}
	defer os.Remove(inputFile)

	// 使用ffmpeg-go提取帧
	buf := bytes.NewBuffer(nil)
	err = ffmpeg.Input(inputFile).
		Filter("select", ffmpeg.Args{fmt.Sprintf("gte(n,%d)", opts.SeekTime*30)}). // 假设30fps
		Output("pipe:", ffmpeg.KwArgs{
			"vframes": 1,
			"format":  "image2",
			"vcodec":  "mjpeg",
		}).
		WithOutput(buf).
		Run()

	if err != nil {
		return fmt.Errorf("ffmpeg extract frame failed: %w", err)
	}

	// 使用imaging库调整大小
	img, err := imaging.Decode(buf)
	if err != nil {
		return fmt.Errorf("decode image failed: %w", err)
	}

	// 调整尺寸
	thumbnail := imaging.Resize(img, opts.Width, opts.Height, imaging.Lanczos)

	// 编码输出
	return imaging.Encode(output, thumbnail, imaging.JPEG, imaging.JPEGQuality(opts.Quality))
}

// TranscodeVideo 视频转码
func (f *FFmpegProcessor) TranscodeVideo(ctx context.Context, input io.Reader, output io.Writer, opts *ProcessorOptions) error {
	inputFile, err := f.createTempFile(input, "input")
	if err != nil {
		return fmt.Errorf("create temp input file failed: %w", err)
	}
	defer os.Remove(inputFile)

	outputFile := filepath.Join(f.tempDir, fmt.Sprintf("output_%d.mp4", time.Now().UnixNano()))
	defer os.Remove(outputFile)

	// 构建ffmpeg转码流水线
	stream := ffmpeg.Input(inputFile)

	// 添加视频滤镜
	if opts != nil && opts.Width > 0 && opts.Height > 0 {
		stream = stream.Filter("scale", ffmpeg.Args{fmt.Sprintf("%d:%d", opts.Width, opts.Height)})
	}

	// 执行转码
	err = stream.Output(outputFile, ffmpeg.KwArgs{
		"c:v":    "libx264",
		"preset": "medium",
		"crf":    "23",
		"c:a":    "aac",
		"b:a":    "128k",
	}).OverWriteOutput().Run()

	if err != nil {
		return fmt.Errorf("ffmpeg transcode failed: %w", err)
	}

	// 读取输出文件
	data, err := os.ReadFile(outputFile)
	if err != nil {
		return fmt.Errorf("read output file failed: %w", err)
	}

	_, err = output.Write(data)
	return err
}

// GetVideoInfo 获取视频元信息
func (f *FFmpegProcessor) GetVideoInfo(ctx context.Context, input io.Reader) (*VideoMetadata, error) {
	inputFile, err := f.createTempFile(input, "probe")
	if err != nil {
		return nil, fmt.Errorf("create temp input file failed: %w", err)
	}
	defer os.Remove(inputFile)

	// 使用ffmpeg-go获取视频信息
	probeData, err := ffmpeg.Probe(inputFile)
	if err != nil {
		return nil, fmt.Errorf("ffmpeg probe failed: %w", err)
	}

	// 解析probe数据转换为VideoMetadata
	// 这里需要根据实际的probe返回格式进行解析
	return f.parseProbeData(probeData)
}

// createTempFile 创建临时文件
func (f *FFmpegProcessor) createTempFile(reader io.Reader, prefix string) (string, error) {
	tempFile, err := os.CreateTemp(f.tempDir, prefix+"_*.tmp")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, reader)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

// parseProbeData 解析ffmpeg probe数据
func (f *FFmpegProcessor) parseProbeData(probeData string) (*VideoMetadata, error) {
	// 使用现有的VideoProcessor解析，避免重复实现
	// vp := NewVideoProcessor(100*1024*1024, []string{"video/mp4", "video/avi"}, 480, 270, 80)

	// TODO: 这里需要将probe数据转换为VideoMetadata
	// 暂时返回基本结构，具体实现需要解析JSON
	return &VideoMetadata{
		Format: "mp4",
		Width:  1920,
		Height: 1080,
	}, nil
}
