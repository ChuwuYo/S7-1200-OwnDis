package main

// ManualController 手动控制器
type ManualController struct {
	client  *ModbusClient
	marquee *MarqueeController
	ui      *UI
}

// NewManualController 创建新的手动控制器
func NewManualController(client *ModbusClient, marquee *MarqueeController, ui *UI) *ManualController {
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

// IsManualControlAllowed 检查是否允许手动控制
func (mc *ManualController) IsManualControlAllowed() bool {
	return !mc.marquee.IsRunning()
}