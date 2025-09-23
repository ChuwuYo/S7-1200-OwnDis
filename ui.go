package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// UI 界面结构体
type UI struct {
	app    fyne.App
	window fyne.Window
	
	// 顶部状态栏组件
	connectionStatus *widget.Label
	runStatus        *widget.Label
	speedLevel       *widget.Label
	delayValue       *widget.Label
	currentOutput    *widget.Label
	
	// 控制按钮
	startButton *widget.Button
	stopButton  *widget.Button
	speedButton *widget.Button
	
	// 连接设置
	ipEntry   *widget.Entry
	portEntry *widget.Entry
	unitIDEntry *widget.Entry
	connectButton *widget.Button
	disconnectButton *widget.Button
	
	// IO状态显示
	dqStatus []*widget.Label // DQ 1-14
	diStatus []*widget.Label // DI 10001-10014
	
	// 温湿度显示
	temperature *widget.Label
	humidity    *widget.Label
	
	// 手动控制区
	dqControls []*widget.Check // DQ 1-14 手动控制复选框
}

// NewUI 创建新的用户界面
func NewUI() *UI {
	myApp := app.New()
	myWindow := myApp.NewWindow("跑马灯控制程序")
	myWindow.Resize(fyne.NewSize(800, 600))
	
	ui := &UI{
		app:    myApp,
		window: myWindow,
	}
	
	ui.initUI()
	return ui
}

// initUI 初始化用户界面
func (ui *UI) initUI() {
	// 初始化顶部状态栏
	ui.connectionStatus = widget.NewLabel("连接状态: 未连接")
	ui.runStatus = widget.NewLabel("运行状态: 停止")
	ui.speedLevel = widget.NewLabel("当前挡位: 无")
	ui.delayValue = widget.NewLabel("延时值: 0ms")
	ui.currentOutput = widget.NewLabel("当前输出点: 无")
	
	statusBar := container.NewVBox(
		ui.connectionStatus,
		ui.runStatus,
		ui.speedLevel,
		ui.delayValue,
		ui.currentOutput,
	)
	
	// 初始化连接设置
	ui.ipEntry = widget.NewEntry()
	ui.ipEntry.SetText("192.168.0.10")
	ui.ipEntry.SetPlaceHolder("192.168.0.10")

	ui.portEntry = widget.NewEntry()
	ui.portEntry.SetText("502")
	ui.portEntry.SetPlaceHolder("502")

	ui.unitIDEntry = widget.NewEntry()
	ui.unitIDEntry.SetText("1")
	ui.unitIDEntry.SetPlaceHolder("1")

	ui.connectButton = widget.NewButton("连接", func() {
		// TODO: 连接功能将在main.go中通过回调函数实现
	})

	ui.disconnectButton = widget.NewButton("断开", func() {
		// TODO: 断开功能将在main.go中通过回调函数实现
	})

	// 创建带标签的输入框，使用Border布局让输入框占据更多空间
	ipForm := container.NewBorder(nil, nil, widget.NewLabel("IP地址:"), nil, ui.ipEntry)
	portForm := container.NewBorder(nil, nil, widget.NewLabel("端口号:"), nil, ui.portEntry)
	unitIDForm := container.NewBorder(nil, nil, widget.NewLabel("Unit ID:"), nil, ui.unitIDEntry)

	// 连接按钮组
	connectButtons := container.NewHBox(ui.connectButton, ui.disconnectButton)

	connectForm := container.NewVBox(
		widget.NewLabel("PLC连接设置:"),
		ipForm,
		portForm,
		unitIDForm,
		connectButtons,
	)
	
	// 初始化控制按钮
	ui.startButton = widget.NewButton("启动", func() {
		// TODO: 启动功能将在main.go中通过回调函数实现
	})

	ui.stopButton = widget.NewButton("停止", func() {
		// TODO: 停止功能将在main.go中通过回调函数实现
	})

	ui.speedButton = widget.NewButton("速度切换", func() {
		// TODO: 速度切换功能将在main.go中通过回调函数实现
	})
	
	controlBar := container.NewHBox(
		ui.startButton,
		ui.stopButton,
		ui.speedButton,
	)
	
	// 初始化DQ状态显示 (1-14)
	ui.dqStatus = make([]*widget.Label, 14)
	dqGrid := container.NewGridWithColumns(7)
	for i := 0; i < 14; i++ {
		ui.dqStatus[i] = widget.NewLabel(fmt.Sprintf("Q%d.%d: OFF", i/8, i%8))
		dqGrid.Add(ui.dqStatus[i])
	}
	
	// 初始化DI状态显示 (10001-10014)
	ui.diStatus = make([]*widget.Label, 14)
	diGrid := container.NewGridWithColumns(7)
	for i := 0; i < 14; i++ {
		ui.diStatus[i] = widget.NewLabel(fmt.Sprintf("I%d.%d: OFF", i/8, i%8))
		diGrid.Add(ui.diStatus[i])
	}
	
	// 初始化温湿度显示
	ui.temperature = widget.NewLabel("温度: --℃")
	ui.humidity = widget.NewLabel("湿度: --%")
	
	envBox := container.NewVBox(
		ui.temperature,
		ui.humidity,
	)
	
	// 初始化手动控制区
	ui.dqControls = make([]*widget.Check, 14)
	controlGrid := container.NewGridWithColumns(7)
	for i := 0; i < 14; i++ {
		ui.dqControls[i] = widget.NewCheck(fmt.Sprintf("Q%d.%d", i/8, i%8), func(b bool) {
			// TODO: 手动控制功能将在main.go中通过回调函数实现
		})
		controlGrid.Add(ui.dqControls[i])
	}
	
	// 右侧主要内容区域
	rightContent := container.NewVBox(
		widget.NewLabel("PLC连接设置:"),
		connectForm,
		widget.NewSeparator(),
		widget.NewLabel("控制按钮:"),
		controlBar,
		widget.NewSeparator(),
		widget.NewLabel("数字输出状态 (DQ 1-14):"),
		dqGrid,
		widget.NewSeparator(),
		widget.NewLabel("数字输入状态 (DI 10001-10014):"),
		diGrid,
		widget.NewSeparator(),
		widget.NewLabel("环境数据:"),
		envBox,
		widget.NewSeparator(),
		widget.NewLabel("手动控制 (停止时可用):"),
		controlGrid,
	)

	// 左侧状态栏
	leftStatusBar := container.NewVBox(
		widget.NewLabel("状态信息:"),
		statusBar,
	)

	// 左右布局
	content := container.NewHBox(
		leftStatusBar,
		widget.NewSeparator(),
		rightContent,
	)
	
	ui.window.SetContent(content)
}

// Show 显示用户界面
func (ui *UI) Show() {
	ui.window.Show()
}

// Run 运行用户界面
func (ui *UI) Run() {
	ui.app.Run()
}