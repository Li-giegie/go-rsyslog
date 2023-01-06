package go_rsyslog

import (
	"fmt"
	"log"
	"testing"
)

// linux 环境下测试
func TestGo_rsyslog(t *testing.T) {
	// 日志输出在 /var/log/syslog_test/ 目录下
	// 启用信息、警告、错误 三种等级等级 不可以使用未启用的等级
	gr,err := NewDefault("syslog_test",LOG_INFO | LOG_WARNING | LOG_ERR)
	if err != nil {
		log.Fatalln(err)
	}
	defer gr.Close()
	fmt.Println(gr.Emerg("Emerg"))			// 错误的示例
	fmt.Println(gr.Alert("Alert"))			// 错误的示例
	fmt.Println(gr.Crit("Crit"))				// 错误的示例
	fmt.Println(gr.Err("Err"))
	fmt.Println(gr.Warning("Warning"))
	fmt.Println(gr.Notice("Notice"))			// 错误的示例
	fmt.Println(gr.Info("Info"))
	fmt.Println(gr.Debug("Debug"))			// 错误的示例


}
