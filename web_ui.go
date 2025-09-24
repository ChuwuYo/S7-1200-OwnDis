package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// WebUI Web用户界面
type WebUI struct {
	server   *http.Server
	template *template.Template
	mu       sync.RWMutex

	// 控制器引用
	modbusClient     *ModbusClient
	marqueeController *MarqueeController
	manualController *ManualController
	config           *Config

	// 状态数据
	connectionStatus string
	runStatus        string
	speedLevel       int
	delayValue       int
	currentOutput    string
	dqStatus         [14]string
	diStatus         [14]string
	temperature      float64
	humidity         float64
}

// NewWebUI 创建新的Web用户界面
func NewWebUI(modbusClient *ModbusClient, marqueeController *MarqueeController, manualController *ManualController, config *Config) *WebUI {
	ui := &WebUI{
		modbusClient:     modbusClient,
		marqueeController: marqueeController,
		manualController: manualController,
		config:           config,
		connectionStatus: "未连接",
		runStatus:        "停止",
		speedLevel:       0,
		delayValue:       0,
		currentOutput:    "无",
		temperature:      25.0,
		humidity:         60.0,
	}

	// 初始化IO状态
	for i := 0; i < 14; i++ {
		ui.dqStatus[i] = "OFF"
		ui.diStatus[i] = "OFF"
	}

	ui.initTemplate()
	ui.startServer()
	return ui
}

// initTemplate 初始化HTML模板
func (ui *WebUI) initTemplate() {
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>S7-1200 跑马灯控制程序</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f0f0f0; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; }
        .header { text-align: center; margin-bottom: 30px; }
        .status-bar { display: flex; justify-content: space-around; margin-bottom: 20px; padding: 10px; background: #e8f4f8; border-radius: 5px; }
        .status-item { text-align: center; }
        .control-panel { display: flex; justify-content: center; gap: 20px; margin-bottom: 30px; }
        .btn { padding: 10px 20px; font-size: 16px; border: none; border-radius: 5px; cursor: pointer; }
        .btn-primary { background: #007bff; color: white; }
        .btn-danger { background: #dc3545; color: white; }
        .btn-success { background: #28a745; color: white; }
        .btn:disabled { background: #6c757d; cursor: not-allowed; }
        .connection-settings { background: #f8f9fa; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .form-group { margin-bottom: 15px; }
        .form-group label { display: inline-block; width: 100px; }
        .form-group input { padding: 5px; width: 200px; }
        .io-status { display: flex; gap: 30px; margin-bottom: 30px; }
        .io-grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 5px; }
        .io-item { padding: 5px; text-align: center; border: 1px solid #ddd; border-radius: 3px; }
        .io-item.on { background: #28a745; color: white; }
        .io-item.off { background: #dc3545; color: white; }
        .manual-control { background: #fff3cd; padding: 20px; border-radius: 5px; }
        .environment { background: #d1ecf1; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
        .footer { text-align: center; margin-top: 30px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>S7-1200 跑马灯控制程序</h1>
        </div>

        <div class="status-bar">
            <div class="status-item">
                <strong>连接状态:</strong> <span id="connectionStatus">{{.ConnectionStatus}}</span>
            </div>
            <div class="status-item">
                <strong>运行状态:</strong> <span id="runStatus">{{.RunStatus}}</span>
            </div>
            <div class="status-item">
                <strong>当前挡位:</strong> <span id="speedLevel">{{.SpeedLevel}}</span>
            </div>
            <div class="status-item">
                <strong>延时值:</strong> <span id="delayValue">{{.DelayValue}}ms</span>
            </div>
            <div class="status-item">
                <strong>当前输出点:</strong> <span id="currentOutput">{{.CurrentOutput}}</span>
            </div>
        </div>

        <div class="control-panel">
            <button class="btn btn-primary" onclick="startMarquee()">启动</button>
            <button class="btn btn-danger" onclick="stopMarquee()">停止</button>
            <button class="btn btn-success" onclick="switchSpeed()">速度切换</button>
        </div>

        <div class="connection-settings">
            <h3>PLC连接设置</h3>
            <div class="form-group">
                <label>IP地址:</label>
                <input type="text" id="ipInput" value="192.168.0.10">
            </div>
            <div class="form-group">
                <label>端口号:</label>
                <input type="text" id="portInput" value="502">
            </div>
            <div class="form-group">
                <label>Unit ID:</label>
                <input type="text" id="unitIdInput" value="1">
            </div>
            <button class="btn btn-primary" onclick="connectPLC()">连接</button>
            <button class="btn btn-success" onclick="saveConfig()">保存配置</button>
            <button class="btn btn-danger" onclick="disconnectPLC()">断开</button>
        </div>

        <div class="io-status">
            <div>
                                <h3>数字输出状态 (DQ 1-14) <button class="btn btn-primary" style="margin-left: 10px; padding: 5px 10px; font-size: 12px;" onclick="refreshDQStatus()">刷新</button></h3>
                <div class="io-grid" id="dqGrid">
                    {{range $i, $status := .DQStatus}}
                    {{if lt $i 8}}
                    <div class="io-item {{$status}}">Q0.{{$i}}</div>
                    {{else}}
                    <div class="io-item {{$status}}">Q1.{{sub $i 8}}</div>
                    {{end}}
                    {{end}}
                </div>
            </div>
            <div>
                <h3>数字输入状态 (DI 10001-10014) <button class="btn btn-primary" style="margin-left: 10px; padding: 5px 10px; font-size: 12px;" onclick="refreshDIStatus()">刷新</button></h3>
                <div class="io-grid" id="diGrid">
                    {{range $i, $status := .DIStatus}}
                    {{if lt $i 8}}
                    <div class="io-item {{$status}}">I0.{{$i}}</div>
                    {{else}}
                    <div class="io-item {{$status}}">I1.{{sub $i 8}}</div>
                    {{end}}
                    {{end}}
                </div>
            </div>
        </div>

        <div class="environment">
            <h3>环境数据</h3>
            <p><strong>温度:</strong> <span id="temperature">{{.Temperature}}℃</span></p>
            <p><strong>湿度:</strong> <span id="humidity">{{.Humidity}}%</span></p>
        </div>

        <div class="manual-control">
            <h3>手动控制 (停止时可用)</h3>
                <div class="io-grid" id="manualGrid">
                    {{range $i, $status := .DQStatus}}
                    <div class="io-item {{$status}}">
                        <label>
                            <input type="checkbox" onchange="toggleOutput({{$i}})" {{if eq $status "ON"}}checked{{end}}>
                            {{if lt $i 8}}Q0.{{$i}}{{else}}Q1.{{sub $i 8}}{{end}}
                        </label>
                    </div>
                    {{end}}
                </div>

        <div class="footer">
            <p>Web界面 - 自动刷新状态</p>
        </div>
    </div>

    <script>
        // 自动刷新状态
        setInterval(updateStatus, 1000);

        function updateStatus() {
            fetch('/status')
                .then(response => response.json())
                .then(data => {
                    document.getElementById('connectionStatus').textContent = data.ConnectionStatus;
                    document.getElementById('runStatus').textContent = data.RunStatus;
                    document.getElementById('speedLevel').textContent = data.SpeedLevel;
                    document.getElementById('delayValue').textContent = data.DelayValue + 'ms';
                    document.getElementById('currentOutput').textContent = data.CurrentOutput;
                    document.getElementById('temperature').textContent = data.Temperature + '℃';
                    document.getElementById('humidity').textContent = data.Humidity + '%';

                    // 更新IO状态
                    updateIOStatus('dqGrid', data.DQStatus);
                    updateIOStatus('diGrid', data.DIStatus);
                })
                .catch(err => console.log('Error:', err));
        }

        function updateIOStatus(gridId, statusArray) {
            const grid = document.getElementById(gridId);
            const items = grid.getElementsByClassName('io-item');
            for (let i = 0; i < items.length; i++) {
                items[i].className = 'io-item ' + statusArray[i].toLowerCase();
            }
        }

        function connectPLC() {
            const ip = document.getElementById('ipInput').value;
            const port = document.getElementById('portInput').value;
            const unitId = document.getElementById('unitIdInput').value;

            fetch('/connect', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ ip, port, unitId })
            })
            .then(response => response.json())
            .then(data => {
                alert(data.message);
                updateStatus();
            });
        }

        function disconnectPLC() {
            fetch('/disconnect', { method: 'POST' })
                .then(response => response.json())
                .then(data => {
                    alert(data.message);
                    updateStatus();
                });
        }

        function saveConfig() {
            const ip = document.getElementById('ipInput').value;
            const port = document.getElementById('portInput').value;
            const unitId = document.getElementById('unitIdInput').value;

            fetch('/save-config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ ip, port, unitId })
            })
            .then(response => response.json())
            .then(data => {
                if (data.error) {
                    alert('错误: ' + data.error);
                } else {
                    alert(data.message);
                }
            })
            .catch(err => {
                alert('保存配置失败: ' + err.message);
            });
        }

        function startMarquee() {
            fetch('/start', { method: 'POST' })
                .then(response => response.json())
                .then(data => {
                    alert(data.message);
                    updateStatus();
                });
        }

        function stopMarquee() {
            fetch('/stop', { method: 'POST' })
                .then(response => response.json())
                .then(data => {
                    alert(data.message);
                    updateStatus();
                });
        }

        function switchSpeed() {
            fetch('/switch-speed', { method: 'POST' })
                .then(response => response.json())
                .then(data => {
                    alert(data.message);
                    updateStatus();
                });
        }

        function toggleOutput(index) {
            const checkbox = event.target;
            const status = checkbox.checked; // true = ON, false = OFF

            fetch('/toggle-output', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ index, status })
            })
            .then(response => response.json())
            .then(data => {
                if (data.error) {
                    alert(data.error);
                    // 恢复复选框状态
                    checkbox.checked = !checkbox.checked;
                }
                // 无论成功失败都刷新状态
                updateStatus();
            })
            .catch(err => {
                alert('操作失败: ' + err.message);
                // 恢复复选框状态
                checkbox.checked = !checkbox.checked;
                updateStatus();
            });
        }

        function refreshDQStatus() {
            updateStatus();
        }

        function refreshDIStatus() {
            updateStatus();
        }
    </script>
</body>
</html>`

	ui.template = template.Must(template.New("webui").Funcs(funcMap).Parse(tmpl))
}

// startServer 启动Web服务器
func (ui *WebUI) startServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", ui.handleIndex)
	mux.HandleFunc("/status", ui.handleStatus)
	mux.HandleFunc("/connect", ui.handleConnect)
	mux.HandleFunc("/disconnect", ui.handleDisconnect)
	mux.HandleFunc("/start", ui.handleStart)
	mux.HandleFunc("/stop", ui.handleStop)
	mux.HandleFunc("/switch-speed", ui.handleSwitchSpeed)
	mux.HandleFunc("/toggle-output", ui.handleToggleOutput)
	mux.HandleFunc("/save-config", ui.handleSaveConfig)

	ui.server = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("启动Web服务器在 http://localhost:8080")
		if err := ui.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Web服务器错误: %v", err)
		}
	}()
}

// Show 显示用户界面
func (ui *WebUI) Show() {
	log.Println("请在浏览器中打开 http://localhost:8080")
}

// Run 运行用户界面
func (ui *WebUI) Run() {
	// 保持程序运行
	select {}
}

// handleIndex 处理主页请求
func (ui *WebUI) handleIndex(w http.ResponseWriter, r *http.Request) {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	data := struct {
		ConnectionStatus string
		RunStatus        string
		SpeedLevel       int
		DelayValue       int
		CurrentOutput    string
		DQStatus         [14]string
		DIStatus         [14]string
		Temperature      float64
		Humidity         float64
	}{
		ConnectionStatus: ui.connectionStatus,
		RunStatus:        ui.runStatus,
		SpeedLevel:       ui.speedLevel,
		DelayValue:       ui.delayValue,
		CurrentOutput:    ui.currentOutput,
		DQStatus:         ui.dqStatus,
		DIStatus:         ui.diStatus,
		Temperature:      ui.temperature,
		Humidity:         ui.humidity,
	}

	ui.template.Execute(w, data)
}

// handleStatus 处理状态请求
func (ui *WebUI) handleStatus(w http.ResponseWriter, r *http.Request) {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")

	// 构建DQ状态数组
	dqArray := make([]string, 14)
	for i, status := range ui.dqStatus {
		dqArray[i] = fmt.Sprintf(`"%s"`, status)
	}

	// 构建DI状态数组
	diArray := make([]string, 14)
	for i, status := range ui.diStatus {
		diArray[i] = fmt.Sprintf(`"%s"`, status)
	}

	fmt.Fprintf(w, `{
		"ConnectionStatus": "%s",
		"RunStatus": "%s",
		"SpeedLevel": %d,
		"DelayValue": %d,
		"CurrentOutput": "%s",
		"DQStatus": [%s],
		"DIStatus": [%s],
		"Temperature": %.1f,
		"Humidity": %.1f
	}`,
		ui.connectionStatus,
		ui.runStatus,
		ui.speedLevel,
		ui.delayValue,
		ui.currentOutput,
		strings.Join(dqArray, ","),
		strings.Join(diArray, ","),
		ui.temperature,
		ui.humidity,
	)
}

// handleConnect 处理连接请求
func (ui *WebUI) handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IP     string `json:"ip"`
		Port   string `json:"port"`
		UnitID string `json:"unitId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 实际调用Modbus客户端连接
	if ui.modbusClient != nil {
		if err := ui.modbusClient.Connect(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"message": "连接失败: %s"}`, err.Error())
			return
		}

		// 连接成功后保存配置
		if ui.config != nil {
			// 转换端口号
			port, err := strconv.Atoi(req.Port)
			if err == nil && port > 0 && port <= 65535 {
				ui.config.Port = port
			}

			// 转换Unit ID
			unitID, err := strconv.Atoi(req.UnitID)
			if err == nil && unitID >= 0 && unitID <= 255 {
				ui.config.UnitID = unitID
			}

			ui.config.IP = req.IP
			if err := ui.config.SaveConfig(); err != nil {
				// 保存配置失败不影响连接，但记录日志
				log.Printf("保存配置失败: %v", err)
			}
		}
	}

	ui.mu.Lock()
	ui.connectionStatus = "已连接"
	ui.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "连接成功"}`)
}

// handleDisconnect 处理断开请求
func (ui *WebUI) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 实际调用Modbus客户端断开
	if ui.modbusClient != nil {
		ui.modbusClient.Close()
	}

	ui.mu.Lock()
	ui.connectionStatus = "未连接"
	ui.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "断开连接成功"}`)
}

// handleStart 处理启动请求
func (ui *WebUI) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 实际调用跑马灯控制器启动
	if ui.marqueeController != nil {
		ui.marqueeController.Start()
	}

	ui.mu.Lock()
	ui.runStatus = "运行中"
	ui.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "启动成功"}`)
}

// handleStop 处理停止请求
func (ui *WebUI) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 实际调用跑马灯控制器停止
	if ui.marqueeController != nil {
		ui.marqueeController.Stop()
	}

	ui.mu.Lock()
	ui.runStatus = "停止"
	ui.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "停止成功"}`)
}

// handleSwitchSpeed 处理速度切换请求
func (ui *WebUI) handleSwitchSpeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 实际调用跑马灯控制器切换速度
	if ui.marqueeController != nil {
		ui.marqueeController.SwitchSpeed()
	}

	ui.mu.Lock()
	ui.speedLevel = (ui.speedLevel % 3) + 1
	ui.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "速度切换成功"}`)
}

// handleToggleOutput 处理输出状态设置请求
func (ui *WebUI) handleToggleOutput(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Index  int  `json:"index"`
		Status bool `json:"status"` // true = ON, false = OFF
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

		if req.Index < 0 || req.Index >= 14 {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"error": "无效的输出索引"}`)
		return
	}

	// 实际调用手动控制器设置输出状态
	if ui.manualController != nil {
		// 读取当前所有输出点状态
		resp, err := ui.modbusClient.ReadCoils(0, 14)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"error": "读取当前状态失败: %s"}`, err.Error())
			return
		}

				// 解析当前输出状态
		currentOutputs := parseCoilsResponse(resp, 14)

						// 设置指定输出点的状态
		currentOutputs[req.Index] = req.Status

		// 写入所有输出点状态到PLC
		_, err = ui.modbusClient.WriteMultipleCoils(0, currentOutputs)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"error": "设置输出状态失败: %s"}`, err.Error())
			return
		}
	}

			// 更新本地状态显示
	ui.mu.Lock()
	if req.Status {
		ui.dqStatus[req.Index] = "ON"
	} else {
		ui.dqStatus[req.Index] = "OFF"
	}
	ui.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "设置成功"}`)
}

// UpdateConnectionStatus 更新连接状态
func (ui *WebUI) UpdateConnectionStatus(status string) {
	ui.mu.Lock()
	ui.connectionStatus = status
	ui.mu.Unlock()
}

// UpdateRunStatus 更新运行状态
func (ui *WebUI) UpdateRunStatus(status string) {
	ui.mu.Lock()
	ui.runStatus = status
	ui.mu.Unlock()
}

// UpdateSpeedLevel 更新速度挡位
func (ui *WebUI) UpdateSpeedLevel(level int) {
	ui.mu.Lock()
	ui.speedLevel = level
	ui.mu.Unlock()
}

// UpdateDelayValue 更新延时值
func (ui *WebUI) UpdateDelayValue(delay int) {
	ui.mu.Lock()
	ui.delayValue = delay
	ui.mu.Unlock()
}

// UpdateCurrentOutput 更新当前输出点
func (ui *WebUI) UpdateCurrentOutput(address string) {
	ui.mu.Lock()
	ui.currentOutput = address
	ui.mu.Unlock()
}

// UpdateDQStatus 更新数字输出状态
func (ui *WebUI) UpdateDQStatus(index int, status string) {
	if index >= 0 && index < 14 {
		ui.mu.Lock()
		ui.dqStatus[index] = status
		ui.mu.Unlock()
	}
}

// UpdateDIStatus 更新数字输入状态
func (ui *WebUI) UpdateDIStatus(index int, status string) {
	if index >= 0 && index < 14 {
		ui.mu.Lock()
		ui.diStatus[index] = status
		ui.mu.Unlock()
	}
}

// UpdateTemperature 更新温度
func (ui *WebUI) UpdateTemperature(temp float64) {
	ui.mu.Lock()
	ui.temperature = temp
	ui.mu.Unlock()
}

// UpdateHumidity 更新湿度
func (ui *WebUI) UpdateHumidity(humidity float64) {
	ui.mu.Lock()
	ui.humidity = humidity
	ui.mu.Unlock()
}

// handleSaveConfig 处理保存配置请求
func (ui *WebUI) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IP     string `json:"ip"`
		Port   string `json:"port"`
		UnitID string `json:"unitId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 验证输入
	if req.IP == "" || req.Port == "" || req.UnitID == "" {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"error": "所有字段都不能为空"}`)
		return
	}

	// 转换端口号
	port, err := strconv.Atoi(req.Port)
	if err != nil || port <= 0 || port > 65535 {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"error": "无效的端口号"}`)
		return
	}

	// 转换Unit ID
	unitID, err := strconv.Atoi(req.UnitID)
	if err != nil || unitID < 0 || unitID > 255 {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"error": "无效的Unit ID"}`)
		return
	}

	// 更新配置
	if ui.config != nil {
		ui.config.IP = req.IP
		ui.config.Port = port
		ui.config.UnitID = unitID

		// 保存配置到文件
		if err := ui.config.SaveConfig(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"error": "保存配置失败: %s"}`, err.Error())
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "配置保存成功"}`)
}