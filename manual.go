package main

// ManualController 手动控制器
type ManualController struct {
	client  *ModbusClient
	marquee *MarqueeController
	ui      *WebUI
}

// NewManualController 创建新的手动控制器
func NewManualController(client *ModbusClient, marquee *MarqueeController, ui *WebUI) *ManualController {
	return &ManualController{
		client:  client,
		marquee: marquee,
		ui:      ui,
	}
}

// SetOutput 设置指定输出点的状态
func (mc *ManualController) SetOutput(index int, value bool) error {
	if mc.marquee.IsRunning() {
		// 跑马灯运行时不允许手动控制
		return nil
	}
	
	if mc.client.conn == nil {
		return nil
	}
	
	if index < 0 || index >= 14 {
		return nil
	}
	
	// 读取当前所有输出点状态
	currentOutputs := make([]bool, 14)
	
	// 设置指定输出点
	currentOutputs[index] = value
	
	// 写入到PLC
	_, err := mc.client.WriteMultipleCoils(0, currentOutputs)
	return err
}

// SetAllOutputs 设置所有输出点的状态
func (mc *ManualController) SetAllOutputs(values []bool) error {
	if mc.marquee.IsRunning() {
		// 跑马灯运行时不允许手动控制
		return nil
	}
	
	if mc.client.conn == nil {
		return nil
	}
	
	if len(values) != 14 {
		return nil
	}
	
	// 写入到PLC
	_, err := mc.client.WriteMultipleCoils(0, values)
	return err
}

// ToggleOutput 设置指定输出点的状态（根据复选框状态）
func (mc *ManualController) ToggleOutput(index int) error {
	if mc.marquee.IsRunning() {
		// 跑马灯运行时不允许手动控制
		return nil
	}

	if mc.client.conn == nil {
		return nil
	}

	if index < 0 || index >= 14 {
		return nil
	}

	// 读取当前所有输出点状态
	resp, err := mc.client.ReadCoils(0, 14)
	if err != nil {
		return err
	}

	// 解析当前输出状态
	currentOutputs := parseCoilsResponse(resp, 14)

	// 注意：这里不切换状态，而是等待WebUI传递实际状态
	// WebUI会根据复选框的checked状态来设置值

	// 写入到PLC
	_, err = mc.client.WriteMultipleCoils(0, currentOutputs)
	return err
}

// IsManualControlAllowed 检查是否允许手动控制
func (mc *ManualController) IsManualControlAllowed() bool {
	return !mc.marquee.IsRunning()
}