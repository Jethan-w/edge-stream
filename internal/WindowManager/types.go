package windowmanager

import (
	"time"
)

// WindowType 窗口类型
type WindowType int

const (
	TIME_WINDOW WindowType = iota
	COUNT_WINDOW
)

// Window 窗口接口
type Window interface {
	GetID() string
	GetWindowType() WindowType
	AddData(data interface{})
	GetData() []interface{}
	IsReady() bool
	Reset()
}

// WindowManager 窗口管理器接口
type WindowManager interface {
	CreateTimeWindow(duration time.Duration) Window
	CreateCountWindow(count int) Window
	ProcessData(data interface{}) error
}

// TriggerStrategy 触发策略接口
type TriggerStrategy interface {
	ShouldTrigger(window Window) bool
}

// OutputProcessor 输出处理器接口
type OutputProcessor interface {
	Process(window Window) error
}