package core

// Scheduler 统一调度接口
// Start: 启动调度，Stop: 停止调度
// 可扩展支持动态任务、优先级、事件驱动等

type Scheduler interface {
	Start() error
	Stop() error
}

// EdgeRunner 系统主流程入口
// Run: 启动全流程，负责加载配置、初始化、启动调度、健康监测等

type EdgeRunner interface {
	Run() error
}
