package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "设备操作",
}

var importCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "导入设备 (YAML)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("导入设备：", args[0])
		// TODO: 调用后端 API 或本地导入逻辑
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "设备列表",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("设备列表：")
		// TODO: 调用后端 API 获取设备列表
	},
}

func init() {
	deviceCmd.AddCommand(importCmd, listCmd)
	rootCmd.AddCommand(deviceCmd)
}
