package main

import (
	"encoding/binary"
	"math"
	"strconv"
	"time"
)

// EnvironmentMonitor 环境监测器
type EnvironmentMonitor struct {
	client   *ModbusClient
	ui       *UI
	stopChan chan bool
}

// NewEnvironmentMonitor 创建新的环境监测器
func NewEnvironmentMonitor(client *ModbusClient, ui *UI) *EnvironmentMonitor {
	return &EnvironmentMonitor{
		client:   client,
		ui:       ui,
		stopChan: make(chan bool),
	}
}

// Start 开始环境数据轮询
func (em *EnvironmentMonitor) Start() {
	go em.pollEnvironment()
}

// Stop 停止环境数据轮询
func (em *EnvironmentMonitor) Stop() {
	em.stopChan <- true
}

// pollEnvironment 轮询环境数据
func (em *EnvironmentMonitor) pollEnvironment() {
	ticker := time.NewTicker(2 * time.Second) // 每2秒读取一次
	defer ticker.Stop()

	for {
		select {
		case <-em.stopChan:
			return
		case <-ticker.C:
			em.readAndUpdateEnvironment()
		}
	}
}

// readAndUpdateEnvironment 读取并更新环境数据
func (em *EnvironmentMonitor) readAndUpdateEnvironment() {
	if em.client.conn == nil || em.ui == nil {
		return
	}

	// 读取温度
	temp, err := em.ReadTemperature()
	if err == nil {
		// 直接更新UI（在goroutine中调用）
		em.ui.temperature.SetText(em.FormatTemperature(temp))
	}

	// 读取湿度
	humid, err := em.ReadHumidity()
	if err == nil {
		// 直接更新UI（在goroutine中调用）
		em.ui.humidity.SetText(em.FormatHumidity(humid))
	}
}

// ReadTemperature 读取温度数据
func (em *EnvironmentMonitor) ReadTemperature() (float64, error) {
	if em.client.conn == nil {
		return 0, nil
	}
	
	// 读取输入寄存器30033 (短地址32)
	resp, err := em.client.ReadInputRegisters(32, 1)
	if err != nil {
		return 0, err
	}
	
	// 解析温度值
	tempRaw := em.parseRegisterValue(resp)
	
	// 转换为实际温度值
	// 温度（℃） = (值 / 27648) × 120 − 40
	temperature := (float64(tempRaw) / 27648.0) * 120.0 - 40.0
	
	return temperature, nil
}

// ReadHumidity 读取湿度数据
func (em *EnvironmentMonitor) ReadHumidity() (float64, error) {
	if em.client.conn == nil {
		return 0, nil
	}
	
	// 读取输入寄存器30034 (短地址33)
	resp, err := em.client.ReadInputRegisters(33, 1)
	if err != nil {
		return 0, err
	}
	
	// 解析湿度值
	humidRaw := em.parseRegisterValue(resp)
	
	// 转换为实际湿度值
	// 湿度（%） = (值 / 27648) × 100
	humidity := (float64(humidRaw) / 27648.0) * 100.0
	
	return humidity, nil
}

// parseRegisterValue 解析寄存器值（大端）
func (em *EnvironmentMonitor) parseRegisterValue(resp []byte) uint16 {
	if len(resp) < 3 {
		return 0
	}
	
	// 第一个字节是字节数
	byteCount := int(resp[0])
	if byteCount < 2 {
		return 0
	}
	
	// 大端解析两个字节
	value := binary.BigEndian.Uint16(resp[1:3])
	return value
}

// IsValidTemperature 检查温度值是否有效
func (em *EnvironmentMonitor) IsValidTemperature(temp float64) bool {
	// 简单的有效性检查
	return temp >= -40.0 && temp <= 80.0
}

// IsValidHumidity 检查湿度值是否有效
func (em *EnvironmentMonitor) IsValidHumidity(humid float64) bool {
	// 简单的有效性检查
	return humid >= 0.0 && humid <= 100.0
}

// FormatTemperature 格式化温度值
func (em *EnvironmentMonitor) FormatTemperature(temp float64) string {
	if em.IsValidTemperature(temp) {
		return roundToOneDecimal(temp) + "℃"
	}
	return "无效"
}

// FormatHumidity 格式化湿度值
func (em *EnvironmentMonitor) FormatHumidity(humid float64) string {
	if em.IsValidHumidity(humid) {
		return roundToOneDecimal(humid) + "%"
	}
	return "无效"
}

// roundToOneDecimal 保留一位小数
func roundToOneDecimal(value float64) string {
	return strconv.FormatFloat(math.Round(value*10)/10, 'f', 1, 64)
}