package main

import (
	"log"
	"os"
	"time"
)

func main() {
	// 设置日志输出到文件
	logFile, err := os.OpenFile("marquee_log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("无法打开日志文件: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// 加载配置
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建Modbus客户端
	client := NewModbusClient(config)

	// 创建跑马灯控制器
	marquee := NewMarqueeController(client, config)

	// 创建手动控制器
	manualController := NewManualController(client, marquee, nil)

	// 创建Web用户界面
	ui := NewWebUI(client, marquee, manualController, config)

	// 显示界面
	ui.Show()

	// 创建输入控制器
	inputController := NewInputController(client, marquee, ui, config)

	// 创建环境监测器
	environmentMonitor := NewEnvironmentMonitor(client, ui)
	defer environmentMonitor.Stop()

	// 启动状态更新goroutine
	go func() {
		for {
			if client.IsConnected() {
				ui.UpdateConnectionStatus("已连接")
			} else {
				ui.UpdateConnectionStatus("未连接")
			}

			if marquee.IsRunning() {
				ui.UpdateRunStatus("运行中")
				ui.UpdateSpeedLevel(marquee.GetSpeedLevel())
				ui.UpdateDelayValue(marquee.GetDelay())
				ui.UpdateCurrentOutput(marquee.GetCurrentOutputAddress())
			} else {
				ui.UpdateRunStatus("停止")
				ui.UpdateSpeedLevel(0)
				ui.UpdateDelayValue(0)
				ui.UpdateCurrentOutput("无")
			}

			// 更新IO状态（从PLC读取实际数据）
			if client.IsConnected() {
				// 读取DQ状态 (线圈0-13)
								if resp, err := client.ReadCoils(0, 14); err == nil {
					dqStatus := parseCoilsResponse(resp, 14)
					for i := 0; i < 14 && i < len(dqStatus); i++ {
						if dqStatus[i] {
							ui.UpdateDQStatus(i, "ON")
						} else {
							ui.UpdateDQStatus(i, "OFF")
						}
					}
				}

								// 读取DI状态 (离散输入 10001-10014)
				if resp, err := client.ReadDiscreteInputs(0, 14); err == nil {
					diStatus := parseDiscreteInputsResponse(resp, 14)
					for i := 0; i < 14 && i < len(diStatus); i++ {
						if diStatus[i] {
							ui.UpdateDIStatus(i, "ON")
						} else {
							ui.UpdateDIStatus(i, "OFF")
						}
					}
				}

				// DI状态由InputController处理，这里不需要重复更新
			} else {
				// 未连接时显示OFF状态
				for i := 0; i < 14; i++ {
					ui.UpdateDQStatus(i, "OFF")
					ui.UpdateDIStatus(i, "OFF")
				}
			}

			// 只有连接PLC时才更新环境数据
			if client.IsConnected() {
				// 从PLC读取实际环境数据
				if temp, err := environmentMonitor.ReadTemperature(); err == nil {
					ui.UpdateTemperature(temp)
				}
				if humid, err := environmentMonitor.ReadHumidity(); err == nil {
					ui.UpdateHumidity(humid)
				}
			} else {
				// 未连接时显示固定值
				ui.UpdateTemperature(25.0)
				ui.UpdateHumidity(60.0)
			}

			time.Sleep(200 * time.Millisecond)
		}
	}()

	// 启动输入轮询
	inputController.Start()
	defer inputController.Stop()

	// 启动环境监测
	environmentMonitor.Start()

	// 运行Web界面
	ui.Run()
}