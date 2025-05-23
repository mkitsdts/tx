package utils

import (
	"fmt"
	"sync"
	"time"
)

const (
	nodeBits  uint8 = 10                    // 机器ID的位数
	stepBits  uint8 = 12                    // 序列号的位数
	nodeMax   int64 = -1 ^ (-1 << nodeBits) // 机器ID的最大值
	stepMax   int64 = -1 ^ (-1 << stepBits) // 序列号的最大值
	timeShift uint8 = nodeBits + stepBits   // 时间戳左移位数
	nodeShift uint8 = stepBits              // 机器ID左移位数
)

var epoch int64 = 1577836800000 // 2020-01-01 00:00:00 作为起始时间

// Snowflake 定义雪花算法结构
type Snowflake struct {
	mu        sync.Mutex
	timestamp int64
	node      int64
	step      int64
}

func NewSnowflake(node int64) (*Snowflake, error) {
	if node < 0 || node > nodeMax {
		return nil, fmt.Errorf("node ID must be between 0 and %d", nodeMax)
	}
	return &Snowflake{
		timestamp: 0,
		node:      node,
		step:      0,
	}, nil
}

// GenerateID 生成唯一ID
func (s *Snowflake) GenerateID() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixNano() / 1000000 // 当前时间戳（毫秒）

	if s.timestamp == now {
		// 如果是同一时间生成的，则进行毫秒内序列
		s.step = (s.step + 1) & stepMax
		if s.step == 0 {
			// 序列号已经达到最大值，等待下一毫秒
			for now <= s.timestamp {
				now = time.Now().UnixNano() / 1000000
			}
		}
	} else {
		// 时间戳改变，毫秒内序列重置
		s.step = 0
	}

	s.timestamp = now

	// 生成ID
	return fmt.Sprint(((now - epoch) << timeShift) | (s.node << nodeShift) | s.step)
}

// 单例模式使用
var defaultSnowflake *Snowflake
var once sync.Once

const machineId = 1

// GenerateId 保持原有API兼容性
func GenerateId() string {
	once.Do(func() {
		flake, _ := NewSnowflake(machineId)
		defaultSnowflake = flake
	})
	return defaultSnowflake.GenerateID()
}
