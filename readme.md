# go-rsyslog 是一个基于syslog开发的日志服务

* [支持日志按级别输出](#)
* [支持输出到指定目录](#)
* [精简化参数配置](#)
* [rsyslog配置文件自动管理](#)

![Go1.19](https://img.shields.io/badge/Go-1.19-blue)
![rsyslog](https://img.shields.io/badge/log-rsyslog-gree)
![develop](https://img.shields.io/badge/develop-99-red)
![Go1.19](https://img.shields.io/badge/Go-1.19-yellow)
![Go1.19](https://img.shields.io/badge/environment-linux-blue)

    非linux环境程序会强制退出
## 目录结构
* go-rsyslog/
  * go-rsyslog.go 核心实现
  * go-rsyslog_test.go 完整的测试例子
  * readme.md readme文件
  * go.mod 包管理
  * go.sum 包管理

## 结构体对象
```go
type GoRSysLog struct {
	// 服务名 强制配置参数
	ServiceName string
	// 网络协议 参考rsyslog 支持的协议列表
	Network string
	// 远程的连接地址
	Raddr string
    // 服务名和日志等级之间的分隔符
	ServiceNameLevelSplitStr string
	// rsyslog配置文件目录
	RSysLogConfDir string
	// log输出目录
	LogSaveDir string
}
```
##使用教程

* [在开始使用前确保基于linux环境下](#)
* [在开始使用前确保具有rsyslog环境](#)

#### 仅需两个步骤 创建一个对象 、使用挂载其身上的函数

* [默认参数对象使用](#)
```go
// 参数：ServiceName 服务名 推荐当前程序的名称
// 初始化一个默认配置的GoRSysLog对象 推荐此用法理由是过多的参数看起来十分不友好
// 此对象会把产生的日志文件放进/var/log/[服务名]/		目录下
// rsyslog 的配置文件会放在/etc/rsyslog.d/		目录下 并且rsyslog 服务会重启
// priority 缺省值（nil）时表示使用所有日志等级 如果指定等级后，使用未指定的等级会输出失败，在定义对象是需要确定今后会使用到的所有等级

// NewDefault(”test“,LOG_DEBUG | LOG_INFO | LOG_NOTICE ....)
gr,err := NewDefault("syslog_test")
if err != nil {
    log.Fatalln(err)
}
// 释放资源
defer gr.Close()
// 2.输出一条最严重的错误
gr.Emerg("Emerg")

```

* [自定义参数的用法](#)
```go
// 参数：ServiceName 服务名，RSysLogConfDir rsyslog配置文件目录
// ServiceNameLevelSplitStr 服务名和日志等级之间的分隔符
// LogSaveDir 日志保存的目录
func New(ServiceName ,RSysLogConfDir,ServiceNameLevelSplitStr,LogSaveDir string)
```

* [远程连接rsyslog](#)
```go
// 参数：network 连接协议 raddr 连接地址 其他参数参考New函数
func NewRemote(network, raddr, ServiceName ,RSysLogConfDir,ServiceNameLevelSplitStr,LogSaveDir string)
```