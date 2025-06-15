package utils

import (
	"errors"
	"fmt"
	"sync"

	"github.com/bwmarrin/snowflake"
)

const (
	customEpoch = 1640995200000

	workerIDBits     = 5
	dataCenterIDBits = 5
	maxWorkerID      = -1 ^ (-1 << workerIDBits)     // 31
	maxDataCenter    = -1 ^ (-1 << dataCenterIDBits) // 31
)

// globalNode 存储 snowflake 节点实例
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
		// 设置自定义 Epoch
		snowflake.Epoch = customEpoch
	})

	// 将 workerID 和 dataCenterID 组合成一个10位的节点ID。
	// (workerID << dataCenterIDBits) | dataCenterID
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
	// 返回 snowflake.ID 类型
	return globalNode.Generate().Int64(), nil
}

// MustGenerateID 生成全局唯一ID
func MustGenerateID() int64 {
	if globalNode == nil {
		panic("snowflake generator not initialized, call InitSnowflake first")
	}
	return globalNode.Generate().Int64()
}
