package main

import (
	"fmt"
	"log"
	"strconv"
)

func main() {
	// 加载配置
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建Modbus客户端
	client := NewModbusClient(config)

	// 创建跑马灯控制器
	marquee := NewMarqueeController(client, config)
	
	// 创建并运行用户界面
	ui := NewUI()
	
	// 初始化UI中的连接参数
	ui.ipEntry.SetText(config.IP)
	ui.portEntry.SetText(strconv.Itoa(config.Port))
	ui.unitIDEntry.SetText(strconv.Itoa(config.UnitID))
	
	// 创建输入控制器
	inputController := NewInputController(client, marquee, ui, config)
	
	// 创建环境监测器
	environmentMonitor := NewEnvironmentMonitor(client, ui)
	defer environmentMonitor.Stop()

	// 创建手动控制器
	_ = NewManualController(client, marquee, ui)

	// 设置UI按钮回调函数
	ui.startButton.OnTapped = func() {
		if !marquee.IsRunning() {
			marquee.Start()
			ui.runStatus.SetText("运行状态: 运行中")
			ui.speedLevel.SetText(fmt.Sprintf("当前挡位: %d", marquee.GetSpeedLevel()))
			ui.delayValue.SetText(fmt.Sprintf("延时值: %dms", marquee.GetDelay()))
			ui.currentOutput.SetText(fmt.Sprintf("当前输出点: %s", marquee.GetCurrentOutputAddress()))
		}
	}

	ui.stopButton.OnTapped = func() {
		if marquee.IsRunning() {
			marquee.Stop()
			ui.runStatus.SetText("运行状态: 停止")
			ui.speedLevel.SetText("当前挡位: 无")
			ui.delayValue.SetText("延时值: 0ms")
			ui.currentOutput.SetText("当前输出点: 无")
		}
	}

	ui.speedButton.OnTapped = func() {
		if marquee.IsRunning() {
			marquee.SwitchSpeed()
			ui.speedLevel.SetText(fmt.Sprintf("当前挡位: %d", marquee.GetSpeedLevel()))
			ui.delayValue.SetText(fmt.Sprintf("延时值: %dms", marquee.GetDelay()))
		}
	}

	// 设置手动控制回调函数
	for i, check := range ui.dqControls {
		index := i // 捕获循环变量
		check.OnChanged = func(b bool) {
			if !marquee.IsRunning() && client.IsConnected() {
				// 只有在停止状态且已连接时才允许手动控制
				outputs := make([]bool, 14)
				outputs[index] = b
				client.WriteMultipleCoils(0, outputs)
			} else if check.Checked != b {
				// 如果不允许控制，恢复原来的状态
				check.SetChecked(!b)
			}
		}
	}
	
	// 设置连接按钮的回调函数
	ui.connectButton.OnTapped = func() {
		// 获取UI中的连接参数
		ip := ui.ipEntry.Text
		portStr := ui.portEntry.Text
		unitIDStr := ui.unitIDEntry.Text
		
		// 更新配置
		config.IP = ip
		port, err := strconv.Atoi(portStr)
		if err != nil {
			log.Printf("Invalid port: %v", err)
			return
		}
		config.Port = port
		
		unitID, err := strconv.Atoi(unitIDStr)
		if err != nil {
			log.Printf("Invalid unit ID: %v", err)
			return
		}
		config.UnitID = unitID
		
		// 保存配置
		err = config.SaveConfig()
		if err != nil {
			log.Printf("Failed to save config: %v", err)
		}
		
		// 更新客户端配置
		client.config = config
		
		// 连接到PLC
		err = client.Connect()
		if err != nil {
			log.Printf("Failed to connect to PLC: %v", err)
			ui.connectionStatus.SetText("连接状态: 连接失败")
			return
		}
	
		// 连接成功后立即测试连接是否有效
		if !client.IsConnected() {
			ui.connectionStatus.SetText("连接状态: 连接失败")
			log.Printf("Connection test failed")
			return
		}
	
		ui.connectionStatus.SetText("连接状态: 已连接")
		log.Println("Connected to PLC successfully")
	}

	// 设置断开按钮的回调函数
	ui.disconnectButton.OnTapped = func() {
		if client.IsConnected() {
			client.Close()
			ui.connectionStatus.SetText("连接状态: 未连接")
			log.Println("Disconnected from PLC")
		}
	}
	
	// 启动输入轮询
	inputController.Start()
	defer inputController.Stop()

	// 启动环境监测
	environmentMonitor.Start()
	
	// 显示界面
	ui.Show()
	ui.Run()
}