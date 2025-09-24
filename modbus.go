package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"
)

// ModbusClient Modbus TCP客户端结构体
type ModbusClient struct {
	conn   net.Conn
	config *Config
	tid    uint16 // 事务ID
}

// 地址类型常量
const (
	COILS_START_ADDR          = 1     // 线圈起始地址
	DISCRETE_INPUTS_START_ADDR = 10001 // 离散输入起始地址
	INPUT_REGISTERS_START_ADDR = 30001 // 输入寄存器起始地址
	HOLDING_REGISTERS_START_ADDR = 40001 // 保持寄存器起始地址
)

// NewModbusClient 创建新的Modbus客户端
func NewModbusClient(config *Config) *ModbusClient {
	return &ModbusClient{
		config: config,
		tid:    1,
	}
}

// Connect 建立TCP连接
func (m *ModbusClient) Connect() error {
	address := fmt.Sprintf("%s:%d", m.config.IP, m.config.Port)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return err
	}
	m.conn = conn
	return nil
}

// Close 关闭连接
func (m *ModbusClient) Close() error {
	if m.conn != nil {
		err := m.conn.Close()
		m.conn = nil
		return err
	}
	return nil
}

// Reconnect 断线重连
func (m *ModbusClient) Reconnect() error {
	// 先关闭现有连接
	m.Close()
	
	// 尝试重新连接
	return m.Connect()
}

// IsConnected 检查是否已连接
func (m *ModbusClient) IsConnected() bool {
	return m.conn != nil
}

// nextTID 获取下一个事务ID
func (m *ModbusClient) nextTID() uint16 {
	tid := m.tid
	m.tid++
	if m.tid == 0 {
		m.tid = 1
	}
	return tid
}

// sendAndReceive 发送请求并接收响应
func (m *ModbusClient) sendAndReceive(pdu []byte) ([]byte, error) {
	// 检查连接状态
	if m.conn == nil {
		return nil, fmt.Errorf("not connected")
	}
	
	// 构造MBAP头
	tid := m.nextTID()
	mbap := make([]byte, 7)
	binary.BigEndian.PutUint16(mbap[0:2], tid)     // 事务ID
	binary.BigEndian.PutUint16(mbap[2:4], 0)       // 协议ID
	binary.BigEndian.PutUint16(mbap[4:6], uint16(len(pdu)+1)) // 长度
	mbap[6] = byte(m.config.UnitID)                // Unit ID

	// 组合请求
	request := append(mbap, pdu...)

	// 发送请求
	_, err := m.conn.Write(request)
	if err != nil {
		// 发送失败，标记连接断开
		m.conn = nil
		return nil, err
	}

	// 读取MBAP头
	respMBAP := make([]byte, 7)
	_, err = m.conn.Read(respMBAP)
	if err != nil {
		// 读取失败，标记连接断开
		m.conn = nil
		return nil, err
	}

	// 解析响应长度
	length := binary.BigEndian.Uint16(respMBAP[4:6])

	// 读取PDU数据
	respPDU := make([]byte, length-1)
	_, err = m.conn.Read(respPDU)
	if err != nil {
		// 读取失败，标记连接断开
		m.conn = nil
		return nil, err
	}

	// 验证事务ID
	respTID := binary.BigEndian.Uint16(respMBAP[0:2])
	if respTID != tid {
		return nil, fmt.Errorf("transaction ID mismatch: expected %d, got %d", tid, respTID)
	}

	return respPDU, nil
}

// ReadCoils 读取线圈 (功能码 0x01)
func (m *ModbusClient) ReadCoils(startAddr uint16, quantity uint16) ([]byte, error) {
	pdu := make([]byte, 5)
	pdu[0] = 0x01                              // 功能码
	binary.BigEndian.PutUint16(pdu[1:3], startAddr)   // 起始地址
	binary.BigEndian.PutUint16(pdu[3:5], quantity)    // 线圈数量

	return m.sendAndReceive(pdu)
}

// ReadDiscreteInputs 读取离散输入 (功能码 0x02)
func (m *ModbusClient) ReadDiscreteInputs(startAddr uint16, quantity uint16) ([]byte, error) {
	pdu := make([]byte, 5)
	pdu[0] = 0x02                              // 功能码
	binary.BigEndian.PutUint16(pdu[1:3], startAddr)   // 起始地址
	binary.BigEndian.PutUint16(pdu[3:5], quantity)    // 输入数量

	return m.sendAndReceive(pdu)
}

// ReadInputRegisters 读取输入寄存器 (功能码 0x04)
func (m *ModbusClient) ReadInputRegisters(startAddr uint16, quantity uint16) ([]byte, error) {
	pdu := make([]byte, 5)
	pdu[0] = 0x04                              // 功能码
	binary.BigEndian.PutUint16(pdu[1:3], startAddr)   // 起始地址
	binary.BigEndian.PutUint16(pdu[3:5], quantity)    // 寄存器数量

	return m.sendAndReceive(pdu)
}

// WriteSingleCoil 写入单个线圈 (功能码 0x05)
func (m *ModbusClient) WriteSingleCoil(addr uint16, value bool) ([]byte, error) {
	pdu := make([]byte, 5)
	pdu[0] = 0x05                              // 功能码
	binary.BigEndian.PutUint16(pdu[1:3], addr)        // 地址
	if value {
		binary.BigEndian.PutUint16(pdu[3:5], 0xFF00)  // ON值
	} else {
		binary.BigEndian.PutUint16(pdu[3:5], 0x0000)  // OFF值
	}

	return m.sendAndReceive(pdu)
}

// WriteMultipleCoils 写入多个线圈 (功能码 0x0F)
func (m *ModbusClient) WriteMultipleCoils(startAddr uint16, values []bool) ([]byte, error) {
	// 计算字节数
	byteCount := (len(values) + 7) / 8
	
	// 构造线圈值字节
	coilBytes := make([]byte, byteCount)
	for i, value := range values {
		if value {
			coilBytes[i/8] |= 1 << (i % 8)
		}
	}

	// 构造PDU
	pduLen := 6 + byteCount
	pdu := make([]byte, pduLen)
	pdu[0] = 0x0F                                    // 功能码
	binary.BigEndian.PutUint16(pdu[1:3], startAddr)         // 起始地址
	binary.BigEndian.PutUint16(pdu[3:5], uint16(len(values))) // 线圈数量
	pdu[5] = byte(byteCount)                         // 字节数
	copy(pdu[6:], coilBytes)                         // 线圈值

	return m.sendAndReceive(pdu)
}

// CalculateShortAddress 计算短地址（偏移量）
func CalculateShortAddress(logicAddr uint16, addrType uint16) uint16 {
	switch addrType {
	case COILS_START_ADDR:
		return logicAddr - COILS_START_ADDR
	case DISCRETE_INPUTS_START_ADDR:
		return logicAddr - DISCRETE_INPUTS_START_ADDR
	case INPUT_REGISTERS_START_ADDR:
		return logicAddr - INPUT_REGISTERS_START_ADDR
	case HOLDING_REGISTERS_START_ADDR:
		return logicAddr - HOLDING_REGISTERS_START_ADDR
	default:
		return logicAddr
	}
}

// parseCoilsResponse 解析线圈响应数据
func parseCoilsResponse(resp []byte, quantity int) []bool {
	if len(resp) < 2 {
		return make([]bool, quantity)
	}

	byteCount := int(resp[1])
	data := resp[2:]

	if len(data) < byteCount {
		return make([]bool, quantity)
	}

	coils := make([]bool, quantity)

	for i := 0; i < quantity; i++ {
		byteIndex := i / 8
		bitIndex := i % 8

		if byteIndex < len(data) {
			coils[i] = (data[byteIndex] & (1 << bitIndex)) != 0
		}
	}

	log.Printf("Parsed Coils: %v", coils)
	return coils
}

// parseDiscreteInputsResponse 解析离散输入响应数据
func parseDiscreteInputsResponse(resp []byte, quantity int) []bool {
	if len(resp) < 2 {
		return make([]bool, quantity)
	}

	byteCount := int(resp[1])
	data := resp[2:]

	if len(data) < byteCount {
		return make([]bool, quantity)
	}

	inputs := make([]bool, quantity)

	for i := 0; i < quantity; i++ {
		byteIndex := i / 8
		bitIndex := i % 8

		if byteIndex < len(data) {
			inputs[i] = (data[byteIndex] & (1 << bitIndex)) != 0
		}
	}

	return inputs
}