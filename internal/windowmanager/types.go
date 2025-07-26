// Copyright 2025 EdgeStream Team
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package windowmanager provides window management functionality for stream processing.
// It includes window types, window managers, and trigger strategies for handling
// time-based and count-based windowing operations.
package windowmanager

import (
	"time"
)

// WindowType 窗口类型
type WindowType int

const (
	TimeWindowType WindowType = iota
	CountWindowType
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
