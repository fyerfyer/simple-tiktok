package utils

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bwmarrin/snowflake"
)

const (
	customEpoch = 1640995200000

	workerIDBits     = 5
	dataCenterIDBits = 5
	maxWorkerID      = -1 ^ (-1 << workerIDBits)
	maxDataCenter    = -1 ^ (-1 << dataCenterIDBits)
)

var globalNode *snowflake.Node
var initOnce sync.Once

// InitSnowflake 初始化全局雪花算法生成器节点
func InitSnowflake(workerID, dataCenterID int64) error {
	if workerID < 0 || workerID > maxWorkerID {
		return fmt.Errorf("worker ID %d out of range (0-%d)", workerID, maxWorkerID)
	}
	if dataCenterID < 0 || dataCenterID > maxDataCenter {
		return fmt.Errorf("datacenter ID %d out of range (0-%d)", dataCenterID, maxDataCenter)
	}

	var err error
	initOnce.Do(func() {
		snowflake.Epoch = customEpoch
	})

	nodeID := (workerID << dataCenterIDBits) | dataCenterID

	var n *snowflake.Node
	n, err = snowflake.NewNode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to create snowflake node: %w", err)
	}
	globalNode = n
	return nil
}

// GenerateID 生成全局唯一ID
func GenerateID() (int64, error) {
	if globalNode == nil {
		return 0, errors.New("snowflake generator not initialized, call InitSnowflake first")
	}
	return globalNode.Generate().Int64(), nil
}

// MustGenerateID 生成全局唯一ID
func MustGenerateID() int64 {
	if globalNode == nil {
		panic("snowflake generator not initialized, call InitSnowflake first")
	}
	return globalNode.Generate().Int64()
}

// GenerateEventID 生成事件ID
func GenerateEventID() string {
	id := MustGenerateID()
	return fmt.Sprintf("evt_%d", id)
}

// GenerateVideoFilename 生成唯一视频文件名
func GenerateVideoFilename(originalName string) string {
	id := MustGenerateID()
	ext := ""
	if idx := strings.LastIndex(originalName, "."); idx != -1 {
		ext = originalName[idx:]
	}
	return fmt.Sprintf("video_%d%s", id, ext)
}

// GenerateCoverFilename 生成封面文件名
func GenerateCoverFilename(videoFilename string) string {
	name := strings.TrimSuffix(videoFilename, filepath.Ext(videoFilename))
	return fmt.Sprintf("%s_cover.jpg", name)
}
