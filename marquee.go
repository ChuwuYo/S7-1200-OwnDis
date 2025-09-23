package main

import (
	"fmt"
	"time"
)

// MarqueeController 跑马灯控制器
type MarqueeController struct {
	client       *ModbusClient
	config       *Config
	currentIndex int        // 当前点亮的输出点索引
	speedLevel   int        // 速度挡位 (1-3)
	isRunning    bool       // 是否正在运行
	stopChan     chan bool  // 停止信号通道
}

// NewMarqueeController 创建新的跑马灯控制器
func NewMarqueeController(client *ModbusClient, config *Config) *MarqueeController {
	return &MarqueeController{
		client:       client,
		config:       config,
		currentIndex: -1,
		speedLevel:   0,
		isRunning:    false,
		stopChan:     make(chan bool),
	}
}

// Start 启动跑马灯
func (m *MarqueeController) Start() {
	if m.isRunning {
		return
	}
	
	m.isRunning = true
	m.speedLevel = 1 // 默认启动为1挡
	
	// 启动跑马灯循环协程
	go m.run()
}

// Stop 停止跑马灯
func (m *MarqueeController) Stop() {
	if !m.isRunning {
		return
	}
	
	m.isRunning = false
	m.stopChan <- true
	
	// 清除所有输出点
	m.clearAllOutputs()
	
	// 重置状态
	m.currentIndex = -1
	m.speedLevel = 0
}

// SwitchSpeed 切换速度挡位
func (m *MarqueeController) SwitchSpeed() {
	if !m.isRunning {
		return
	}
	
	// 按顺序切换挡位: 1→2→3→1
	m.speedLevel++
	if m.speedLevel > 3 {
		m.speedLevel = 1
	}
}

// GetDelay 获取当前挡位延时值
func (m *MarqueeController) GetDelay() int {
	if m.speedLevel <= 0 || m.speedLevel > len(m.config.SpeedDelays) {
		return 1000 // 默认延时
	}
	return m.config.SpeedDelays[m.speedLevel-1]
}

// IsRunning 检查跑马灯是否正在运行
func (m *MarqueeController) IsRunning() bool {
	return m.isRunning
}

// GetSpeedLevel 获取当前速度挡位
func (m *MarqueeController) GetSpeedLevel() int {
	return m.speedLevel
}

// GetCurrentIndex 获取当前输出点索引
func (m *MarqueeController) GetCurrentIndex() int {
	return m.currentIndex
}

// GetCurrentOutputAddress 获取当前输出点地址
func (m *MarqueeController) GetCurrentOutputAddress() string {
	if m.currentIndex < 0 || m.currentIndex >= 14 {
		return "无"
	}
	return fmt.Sprintf("Q%d.%d", m.currentIndex/8, m.currentIndex%8)
}

// run 跑马灯主循环
func (m *MarqueeController) run() {
	// 输出点顺序: 1-14 (对应Q0.0-Q1.5)
	outputs := make([]bool, 14)
	
	ticker := time.NewTicker(time.Duration(m.GetDelay()) * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			if !m.isRunning {
				return
			}
			
			// 更新定时器间隔
			ticker.Reset(time.Duration(m.GetDelay()) * time.Millisecond)
			
			// 关闭当前点亮的输出点
			if m.currentIndex >= 0 {
				outputs[m.currentIndex] = false
			}
			
			// 移动到下一个输出点
			m.currentIndex++
			if m.currentIndex >= 14 {
				m.currentIndex = 0
			}
			
			// 点亮新的输出点
			outputs[m.currentIndex] = true
			
			// 写入到PLC
			if m.client.conn != nil {
				// 使用短地址0对应逻辑地址1
				m.client.WriteMultipleCoils(0, outputs)
			}
		}
	}
}

// clearAllOutputs 清除所有输出点
func (m *MarqueeController) clearAllOutputs() {
	outputs := make([]bool, 14)
	
	if m.client.conn != nil {
		// 使用短地址0对应逻辑地址1
		m.client.WriteMultipleCoils(0, outputs)
	}
}