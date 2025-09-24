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
	<html lang="zh-CN">
	<head>
	    <title>S7-1200 跑马灯控制程序</title>
	    <meta charset="utf-8">
	    <meta name="viewport" content="width=device-width, initial-scale=1.0">
	    <link href="https://fonts.googleapis.com/css2?family=Roboto:wght@300;400;500;700&display=swap" rel="stylesheet">
	    <style>
	        /* Material Design 3 主题色 */
	        :root {
	            --md-sys-color-primary: #1976d2;
	            --md-sys-color-on-primary: #ffffff;
	            --md-sys-color-primary-container: #d3e4fd;
	            --md-sys-color-on-primary-container: #001b3e;
	            --md-sys-color-secondary: #565f71;
	            --md-sys-color-on-secondary: #ffffff;
	            --md-sys-color-secondary-container: #dae2f9;
	            --md-sys-color-on-secondary-container: #131c2b;
	            --md-sys-color-tertiary: #705575;
	            --md-sys-color-on-tertiary: #ffffff;
	            --md-sys-color-tertiary-container: #fdd7fc;
	            --md-sys-color-on-tertiary-container: #28132e;
	            --md-sys-color-error: #ba1a1a;
	            --md-sys-color-on-error: #ffffff;
	            --md-sys-color-error-container: #ffdad6;
	            --md-sys-color-on-error-container: #410002;
	            --md-sys-color-surface: #fef7ff;
	            --md-sys-color-on-surface: #1a1c1e;
	            --md-sys-color-surface-variant: #dde3ea;
	            --md-sys-color-on-surface-variant: #41484d;
	            --md-sys-color-outline: #71787e;
	            --md-sys-color-shadow: #000000;
	            --md-sys-color-surface-tint: #1976d2;
	        }

	        * {
	            box-sizing: border-box;
	        }

	        body {
	            font-family: 'Roboto', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
	            margin: 0;
	            padding: 24px;
	            background: linear-gradient(135deg, #e3f2fd 0%, #f3e5f5 100%);
	            color: var(--md-sys-color-on-surface);
	            line-height: 1.5;
	        }

	        .app-container {
	            max-width: 1400px;
	            margin: 0 auto;
	            display: grid;
	            gap: 24px;
	        }

	        /* 顶部应用栏 */
	        .app-bar {
	            background: var(--md-sys-color-surface);
	            border-radius: 28px;
	            padding: 16px 32px;
	            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
	            margin-bottom: 8px;
	        }

	        .app-title {
	            font-size: 28px;
	            font-weight: 500;
	            color: var(--md-sys-color-on-surface);
	            margin: 0;
	            text-align: center;
	        }

	        /* 状态卡片 */
	        .status-cards {
	            display: grid;
	            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
	            gap: 16px;
	            margin-bottom: 24px;
	        }

	        .status-card {
	            background: var(--md-sys-color-surface);
	            border-radius: 20px;
	            padding: 20px;
	            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
	            border: 1px solid var(--md-sys-color-surface-variant);
	            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
	        }

	        .status-card:hover {
	            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
	            transform: translateY(-2px);
	        }

	        .status-label {
	            font-size: 14px;
	            font-weight: 500;
	            color: var(--md-sys-color-on-surface-variant);
	            margin-bottom: 8px;
	            text-transform: uppercase;
	            letter-spacing: 0.5px;
	        }

	        .status-value {
	            font-size: 18px;
	            font-weight: 600;
	            color: var(--md-sys-color-on-surface);
	        }

	        .status-value.connected { color: #2e7d32; }
	        .status-value.running { color: #1976d2; }
	        .status-value.stopped { color: #d32f2f; }

	        /* 控制按钮组 */
	        .control-section {
	            background: var(--md-sys-color-surface);
	            border-radius: 24px;
	            padding: 32px;
	            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
	            margin-bottom: 24px;
	        }

	        .control-title {
	            font-size: 20px;
	            font-weight: 500;
	            color: var(--md-sys-color-on-surface);
	            margin: 0 0 24px 0;
	        }

	        .button-group {
	            display: flex;
	            gap: 16px;
	            justify-content: center;
	            flex-wrap: wrap;
	        }

	        .md-button {
	            min-width: 120px;
	            height: 48px;
	            padding: 0 24px;
	            border: none;
	            border-radius: 24px;
	            font-size: 16px;
	            font-weight: 500;
	            cursor: pointer;
	            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
	            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
	            text-transform: none;
	            letter-spacing: 0.5px;
	        }

	        .md-button:hover {
	            box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
	            transform: translateY(-1px);
	        }

	        .md-button:active {
	            box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
	            transform: translateY(0);
	        }

	        .md-button.filled {
	            background: var(--md-sys-color-primary);
	            color: var(--md-sys-color-on-primary);
	        }

	        .md-button.filled:hover {
	            background: #1565c0;
	        }

	        .md-button.outlined {
	            background: transparent;
	            color: var(--md-sys-color-primary);
	            border: 2px solid var(--md-sys-color-primary);
	        }

	        .md-button.outlined:hover {
	            background: var(--md-sys-color-primary);
	            color: var(--md-sys-color-on-primary);
	        }

	        /* 连接设置卡片 */
	        .config-card {
	            background: var(--md-sys-color-surface);
	            border-radius: 24px;
	            padding: 32px;
	            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
	            margin-bottom: 24px;
	        }

	        .config-title {
	            font-size: 20px;
	            font-weight: 500;
	            color: var(--md-sys-color-on-surface);
	            margin: 0 0 24px 0;
	        }

	        .form-grid {
	            display: grid;
	            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
	            gap: 24px;
	            margin-bottom: 24px;
	        }

	        .form-field {
	            display: flex;
	            flex-direction: column;
	        }

	        .form-label {
	            font-size: 14px;
	            font-weight: 500;
	            color: var(--md-sys-color-on-surface-variant);
	            margin-bottom: 8px;
	        }

	        .form-input {
	            height: 48px;
	            padding: 0 16px;
	            border: 2px solid var(--md-sys-color-surface-variant);
	            border-radius: 12px;
	            font-size: 16px;
	            background: var(--md-sys-color-surface);
	            color: var(--md-sys-color-on-surface);
	            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
	        }

	        .form-input:focus {
	            outline: none;
	            border-color: var(--md-sys-color-primary);
	            box-shadow: 0 0 0 3px rgba(25, 118, 210, 0.1);
	        }

	        /* IO状态卡片容器 */
	        .io-cards-container {
	            display: grid;
	            grid-template-columns: 1fr 1fr;
	            gap: 24px;
	            margin-bottom: 24px;
	        }

	        .io-card {
	            background: var(--md-sys-color-surface);
	            border-radius: 24px;
	            padding: 32px;
	            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
	        }

	        .io-title {
	            font-size: 20px;
	            font-weight: 500;
	            color: var(--md-sys-color-on-surface);
	            margin: 0 0 24px 0;
	            display: flex;
	            align-items: center;
	            justify-content: space-between;
	        }

	        .refresh-button {
	            background: var(--md-sys-color-secondary-container);
	            color: var(--md-sys-color-on-secondary-container);
	            border: none;
	            border-radius: 20px;
	            padding: 8px 16px;
	            font-size: 12px;
	            font-weight: 500;
	            cursor: pointer;
	            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
	        }

	        .refresh-button:hover {
	            background: var(--md-sys-color-on-secondary-container);
	            color: var(--md-sys-color-secondary-container);
	        }

	        .io-grid {
	            display: grid;
	            grid-template-columns: repeat(4, 1fr);
	            gap: 12px;
	        }

	        .io-item {
	            height: 64px;
	            border-radius: 16px;
	            display: flex;
	            align-items: center;
	            justify-content: center;
	            font-size: 14px;
	            font-weight: 500;
	            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
	            border: 2px solid transparent;
	            position: relative;
	        }

	        .io-item.on {
	            background: linear-gradient(135deg, #c8e6c9 0%, #a5d6a7 100%);
	            color: #1b5e20;
	            border-color: #4caf50;
	            box-shadow: 0 2px 8px rgba(76, 175, 80, 0.3);
	        }

	        .io-item.off {
	            background: linear-gradient(135deg, #ffcdd2 0%, #ef9a9a 100%);
	            color: #b71c1c;
	            border-color: #f44336;
	            box-shadow: 0 2px 8px rgba(244, 67, 54, 0.3);
	        }

	        .io-item:hover {
	            transform: translateY(-2px);
	            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
	        }

	        /* 环境数据卡片 */
	        .env-card {
	            background: var(--md-sys-color-surface);
	            border-radius: 24px;
	            padding: 32px;
	            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
	            margin-bottom: 24px;
	        }

	        .env-title {
	            font-size: 20px;
	            font-weight: 500;
	            color: var(--md-sys-color-on-surface);
	            margin: 0 0 24px 0;
	        }

	        .env-data {
	            display: flex;
	            gap: 32px;
	        }

	        .env-item {
	            flex: 1;
	            text-align: center;
	            padding: 20px;
	            background: var(--md-sys-color-surface-variant);
	            border-radius: 16px;
	        }

	        .env-label {
	            font-size: 14px;
	            color: var(--md-sys-color-on-surface-variant);
	            margin-bottom: 8px;
	        }

	        .env-value {
	            font-size: 24px;
	            font-weight: 600;
	            color: var(--md-sys-color-on-surface);
	        }

	        /* 手动控制卡片 */
	        .manual-card {
	            background: var(--md-sys-color-surface);
	            border-radius: 24px;
	            padding: 32px;
	            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
	        }

	        .manual-title {
	            font-size: 20px;
	            font-weight: 500;
	            color: var(--md-sys-color-on-surface);
	            margin: 0 0 24px 0;
	        }

	        .manual-grid {
	            display: grid;
	            grid-template-columns: repeat(4, 1fr);
	            gap: 12px;
	        }

	        .manual-item {
	            display: flex;
	            align-items: center;
	            justify-content: center;
	            padding: 12px;
	            background: var(--md-sys-color-surface-variant);
	            border-radius: 16px;
	            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
	        }

	        .manual-item:hover {
	            background: var(--md-sys-color-tertiary-container);
	        }

	        .manual-checkbox {
	            margin-right: 8px;
	            transform: scale(1.2);
	        }

	        /* 响应式设计 */
	        @media (max-width: 768px) {
	            body {
	                padding: 16px;
	            }

	            .app-title {
	                font-size: 24px;
	            }

	            .status-cards {
	                grid-template-columns: 1fr;
	            }

	            .button-group {
	                flex-direction: column;
	                align-items: center;
	            }

	            .io-cards-container {
	                grid-template-columns: 1fr;
	            }

	            .io-grid, .manual-grid {
	                grid-template-columns: repeat(2, 1fr);
	            }

	            .env-data {
	                flex-direction: column;
	            }
	        }

	        /* 动画效果 */
	        @keyframes fadeIn {
	            from { opacity: 0; transform: translateY(20px); }
	            to { opacity: 1; transform: translateY(0); }
	        }

	        .status-card, .control-section, .config-card, .io-card, .env-card, .manual-card {
	            animation: fadeIn 0.6s cubic-bezier(0.4, 0, 0.2, 1);
	        }
	    </style>
	</head>
<body>
    <div class="app-container">
        <!-- 应用标题栏 -->
        <div class="app-bar">
            <h1 class="app-title">S7-1200 跑马灯控制程序</h1>
        </div>

        <!-- 状态卡片组 -->
        <div class="status-cards">
            <div class="status-card">
                <div class="status-label">连接状态</div>
                <div class="status-value {{if eq .ConnectionStatus "已连接"}}connected{{else if eq .ConnectionStatus "连接失败"}}error{{else}}stopped{{end}}" id="connectionStatus">
                    {{.ConnectionStatus}}
                </div>
            </div>
            <div class="status-card">
                <div class="status-label">运行状态</div>
                <div class="status-value {{if eq .RunStatus "运行中"}}running{{else}}stopped{{end}}" id="runStatus">
                    {{.RunStatus}}
                </div>
            </div>
            <div class="status-card">
                <div class="status-label">当前挡位</div>
                <div class="status-value" id="speedLevel">{{.SpeedLevel}}</div>
            </div>
            <div class="status-card">
                <div class="status-label">延时值</div>
                <div class="status-value" id="delayValue">{{.DelayValue}}ms</div>
            </div>
            <div class="status-card">
                <div class="status-label">当前输出点</div>
                <div class="status-value" id="currentOutput">{{.CurrentOutput}}</div>
            </div>
        </div>

        <!-- 控制按钮区域 -->
        <div class="control-section">
            <h2 class="control-title">跑马灯控制</h2>
            <div class="button-group">
                <button class="md-button filled" onclick="startMarquee()">启动</button>
                <button class="md-button outlined" onclick="stopMarquee()">停止</button>
                <button class="md-button outlined" onclick="switchSpeed()">速度切换</button>
            </div>
        </div>

        <!-- PLC连接设置 -->
        <div class="config-card">
            <h2 class="config-title">PLC 连接设置</h2>
            <div class="form-grid">
                <div class="form-field">
                    <div class="form-label">IP 地址</div>
                    <input type="text" class="form-input" id="ipInput" value="192.168.0.10" placeholder="请输入PLC IP地址">
                </div>
                <div class="form-field">
                    <div class="form-label">端口号</div>
                    <input type="text" class="form-input" id="portInput" value="502" placeholder="请输入端口号">
                </div>
                <div class="form-field">
                    <div class="form-label">Unit ID</div>
                    <input type="text" class="form-input" id="unitIdInput" value="1" placeholder="请输入设备ID">
                </div>
            </div>
            <div class="button-group">
                <button class="md-button filled" onclick="connectPLC()">连接 PLC</button>
                <button class="md-button outlined" onclick="saveConfig()">保存配置</button>
                <button class="md-button outlined" onclick="disconnectPLC()">断开连接</button>
            </div>
        </div>

        <!-- IO状态显示区域 -->
        <div class="io-cards-container">
            <!-- 数字输出状态 -->
            <div class="io-card">
                <div class="io-title">
                    <h3 style="margin: 0;">数字输出状态</h3>
                    <button class="refresh-button" onclick="refreshDQStatus()">刷新</button>
                </div>
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

            <!-- 数字输入状态 -->
            <div class="io-card">
                <div class="io-title">
                    <h3 style="margin: 0;">数字输入状态</h3>
                    <button class="refresh-button" onclick="refreshDIStatus()">刷新</button>
                </div>
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

        <!-- 环境数据 -->
        <div class="env-card">
            <h2 class="env-title">环境监测</h2>
            <div class="env-data">
                <div class="env-item">
                    <div class="env-label">温度</div>
                    <div class="env-value" id="temperature">{{.Temperature}}°C</div>
                </div>
                <div class="env-item">
                    <div class="env-label">湿度</div>
                    <div class="env-value" id="humidity">{{.Humidity}}%</div>
                </div>
            </div>
        </div>

        <!-- 手动控制 -->
        <div class="manual-card">
            <h2 class="manual-title">手动控制</h2>
            <p style="color: var(--md-sys-color-on-surface-variant); margin-bottom: 24px;">停止状态下可手动控制输出点，运行时自动保护</p>
            <div class="manual-grid" id="manualGrid">
                {{range $i, $status := .DQStatus}}
                <div class="manual-item">
                    <label style="display: flex; align-items: center; cursor: pointer;">
                        <input type="checkbox" class="manual-checkbox" onchange="toggleOutput({{$i}})" {{if eq $status "ON"}}checked{{end}}>
                        <span style="margin-left: 8px; font-weight: 500;">
                            {{if lt $i 8}}Q0.{{$i}}{{else}}Q1.{{sub $i 8}}{{end}}
                        </span>
                    </label>
                </div>
                {{end}}
            </div>
        </div>
    </div>

    <script>
        // 自动刷新状态
        setInterval(updateStatus, 1000);

        function updateStatus() {
            fetch('/status')
                .then(response => response.json())
                .then(data => {
                    // 更新状态卡片
                    updateStatusCard('connectionStatus', data.ConnectionStatus);
                    updateStatusCard('runStatus', data.RunStatus);
                    document.getElementById('speedLevel').textContent = data.SpeedLevel;
                    document.getElementById('delayValue').textContent = data.DelayValue + 'ms';
                    document.getElementById('currentOutput').textContent = data.CurrentOutput;
                    document.getElementById('temperature').textContent = data.Temperature + '°C';
                    document.getElementById('humidity').textContent = data.Humidity + '%';

                    // 更新IO状态
                    updateIOStatus('dqGrid', data.DQStatus);
                    updateIOStatus('diGrid', data.DIStatus);
                    updateManualControlCheckboxes(data.DQStatus);
                })
                .catch(err => console.error('状态更新失败:', err));
        }

        function updateStatusCard(elementId, status) {
            const element = document.getElementById(elementId);
            element.textContent = status;

            // 移除所有状态类
            element.className = element.className.replace(/\b(connected|running|stopped|error)\b/g, '');

            // 根据状态添加对应的CSS类
            if (status === '已连接') {
                element.classList.add('connected');
            } else if (status === '运行中') {
                element.classList.add('running');
            } else if (status === '停止' || status === '未连接' || status === '连接失败') {
                element.classList.add('stopped');
            }
        }

        function updateManualControlCheckboxes(statusArray) {
            const grid = document.getElementById('manualGrid');
            const checkboxes = grid.querySelectorAll('input[type="checkbox"]');
            checkboxes.forEach((checkbox, index) => {
                checkbox.checked = (statusArray[index] === 'ON');
            });
        }

        function updateIOStatus(gridId, statusArray) {
            const grid = document.getElementById(gridId);
            const items = grid.querySelectorAll('.io-item');
            items.forEach((item, index) => {
                // 移除所有状态类
                item.className = item.className.replace(/\b(on|off)\b/g, '');
                // 添加新的状态类
                item.classList.add(statusArray[index].toLowerCase());
            });
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
	ui.mu.Lock()
	defer ui.mu.Unlock()

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