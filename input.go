package main

import (
	"time"
)

// InputController 输入控制器
type InputController struct {
	client        *ModbusClient
	marquee       *MarqueeController
	ui            *WebUI
	config        *Config
	previousInputs []bool  // 上一次的输入状态，用于边沿检测
	stopChan      chan bool
}

// NewInputController 创建新的输入控制器
func NewInputController(client *ModbusClient, marquee *MarqueeController, ui *WebUI, config *Config) *InputController {
	return &InputController{
		client:        client,
		marquee:       marquee,
		ui:            ui,
		config:        config,
		previousInputs: make([]bool, 14),
		stopChan:      make(chan bool),
	}
}

// Start 开始轮询输入点
func (ic *InputController) Start() {
	go ic.pollInputs()
}

// Stop 停止轮询输入点
func (ic *InputController) Stop() {
	ic.stopChan <- true
}

// pollInputs 轮询输入点状态
func (ic *InputController) pollInputs() {
	ticker := time.NewTicker(time.Duration(ic.config.PollIntervalMs) * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ic.stopChan:
			return
		case <-ticker.C:
			ic.readAndProcessInputs()
		}
	}
}

// readAndProcessInputs 读取并处理输入点状态
func (ic *InputController) readAndProcessInputs() {
	if ic.client.conn == nil {
		return
	}
	
	// 读取DI状态 (地址10001-10014，短地址0-13)
	resp, err := ic.client.ReadDiscreteInputs(0, 14)
	if err != nil {
		// TODO: 处理读取错误
		return
	}
	
	// 解析响应数据
	inputs := ic.parseDiscreteInputs(resp)
	
	// 更新UI显示
	ic.updateUI(inputs)
	
	// 处理按钮事件（边沿检测）
	ic.processButtonEvents(inputs)
	
	// 保存当前状态用于下次边沿检测
	copy(ic.previousInputs, inputs)
}

// parseDiscreteInputs 解析离散输入响应数据
func (ic *InputController) parseDiscreteInputs(resp []byte) []bool {
	if len(resp) < 2 {
		return make([]bool, 14)
	}
	
	// 第一个字节是字节数
	byteCount := int(resp[0])
	if byteCount < 2 {
		return make([]bool, 14)
	}
	
	// 解析输入状态
	inputs := make([]bool, 14)
	data := resp[1:]
	
	for i := 0; i < 14; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		
		if byteIndex < len(data) {
			inputs[i] = (data[byteIndex] & (1 << bitIndex)) != 0
		}
	}
	
	return inputs
}

// updateUI 更新UI显示
func (ic *InputController) updateUI(inputs []bool) {
	// 更新界面
	if ic.ui == nil {
		return
	}

	// 更新DI状态显示
	for i := 0; i < len(inputs); i++ {
		if inputs[i] {
			ic.ui.UpdateDIStatus(i, "ON")
		} else {
			ic.ui.UpdateDIStatus(i, "OFF")
		}
	}
}

// processButtonEvents 处理按钮事件（边沿检测）
func (ic *InputController) processButtonEvents(inputs []bool) {
	// 检查启动/速度切换按钮（假设为第一个输入点 I0.0）
	if len(inputs) > 0 && len(ic.previousInputs) > 0 {
		// 上升沿检测
		if inputs[0] && !ic.previousInputs[0] {
			if ic.marquee.IsRunning() {
				// 运行中，切换速度
				ic.marquee.SwitchSpeed()
			} else {
				// 未运行，启动跑马灯
				ic.marquee.Start()
			}
		}
	}
	
	// 检查停止按钮（假设为第二个输入点 I0.1）
	if len(inputs) > 1 && len(ic.previousInputs) > 1 {
		// 上升沿检测
		if inputs[1] && !ic.previousInputs[1] {
			if ic.marquee.IsRunning() {
				// 运行中，停止跑马灯
				ic.marquee.Stop()
			}
		}
	}
}