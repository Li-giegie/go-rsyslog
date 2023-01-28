package go_rsyslog
//package main

import (
	"bytes"
	"errors"
	"fmt"
	"log/syslog"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

type Priority int

const (
	LOG_EMERG Priority = 1 << iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

type levelInfo struct {
	Name string
	Info string
	SysLogLevel Priority
}

var lock sync.RWMutex

//说明
var LogLevel = map[Priority] levelInfo {
	1: {
		Name: "LOG_EMERG",
		Info: "几乎要宕机的状态，出现这问题说明要歇菜了，很严重。一般硬件有问题，导致内核无法正常工作，会有此信息。",
		SysLogLevel: 0,
	},
	2: {
		Name: "LOG_ALERT",
		Info: "比crit还严重",
		SysLogLevel: 1,
	},
	4: {
		Name: "LOG_CRIT",
		Info: "比error还严重，critical（临界点），该错误很严重",
		SysLogLevel: 2,
	},
	8: {
		Name: "LOG_ERR",
		Info: "一些重大的错误信息，比如配置文件写错导致daemon无法启动，通常可以根据error的信息就能修复问题",
		SysLogLevel: 3,
	},
	16: {
		Name: "LOG_WARNING",
		Info: "警示的信息，可能有问题，但是还不至于影响某个daemon的运行。基本上info、notice、warning这三个等级没啥事，就是告诉你一些基本信息",
		SysLogLevel: 4,
	},
	32: {
		Name: "LOG_NOTICE",
		Info: "虽然是正常信息，但比info还需要被注意到的一些信息内容",
		SysLogLevel: 5,
	},
	64: {
		Name: "LOG_INFO",
		Info: "基本信息说明，无伤大雅",
		SysLogLevel: 6,
	},
	128: {
		Name: "LOG_DEBUG",
		Info: "调试程序产生的的日志",
		SysLogLevel: 7,
	},
}

type GoRSysLog struct {
	ServiceName string
	Network string
	Raddr string
	priority Priority
	cache sync.Map
	ServiceNameLevelSplitStr string
	RSysLogConfDir string
	LogSaveDir string
	//IsDebug bool
}

func init()  {

	if runtime.GOOS != "linux" {
		fmt.Println("go-rsyslog不支持非linux环境下运行")
		os.Exit(1)
	}
}

// 参数：ServiceName 服务名 推荐当前程序的名称
// 初始化一个默认配置的GoRSysLog对象 推荐此用法理由是过多的参数看起来十分不友好
// 此对象会把产生的日志文件放进/var/log/[服务名]/		目录下
// rsyslog 的配置文件会放在/etc/rsyslog.d/			目录下 并且rsyslog 服务会重启
// priority 缺省值（nil）时表示使用所有日志等级 如果指定等级后，使用未指定的等级会输出失败，在定义对象是需要确定今后会使用到的所有等级
// NewDefault(”test“,LOG_DEBUG | LOG_INFO | LOG_NOTICE ....)
func NewDefault(ServiceName string,priority ...Priority) (*GoRSysLog,error) {
	return _new("", "", ServiceName ,"/etc/rsyslog.d/","@","/var/log/",priority...)
}

// 参数：ServiceName 服务名，RSysLogConfDir rsyslog配置文件目录
// ServiceNameLevelSplitStr 服务名和日志等级之间的分隔符
// LogSaveDir 日志保存的目录
func New(ServiceName ,RSysLogConfDir,ServiceNameLevelSplitStr,LogSaveDir string,priority ...Priority) (*GoRSysLog,error) {
	return _new("", "", ServiceName ,RSysLogConfDir,ServiceNameLevelSplitStr,LogSaveDir,priority...)
}

// 参数：network 连接协议 raddr 连接地址 其他参数参考New函数 priority 预先定义使用的所有日志等级 缺省时为所有包含日志等级
func NewRemote(network, raddr, ServiceName ,RSysLogConfDir,ServiceNameLevelSplitStr,LogSaveDir string,priority ...Priority) (*GoRSysLog,error) {
	return _new(network, raddr, ServiceName ,RSysLogConfDir,ServiceNameLevelSplitStr,LogSaveDir,priority...)
}

func _new(network, raddr, ServiceName ,RSysLogConfDir,ServiceNameLevelSplitStr,LogSaveDir string,priority ...Priority) (*GoRSysLog,error) {

	if priority == nil { priority = []Priority{LOG_EMERG | LOG_ALERT | LOG_CRIT | LOG_ERR | LOG_WARNING | LOG_NOTICE | LOG_INFO | LOG_DEBUG}}

	if ServiceName == "" {
		return nil, errors.New("不允许服务名为空")
	}

	if RSysLogConfDir[len(RSysLogConfDir)-1:] != "/" && RSysLogConfDir[len(RSysLogConfDir)-1:] != `\`{
		RSysLogConfDir += "/"
	}
	if LogSaveDir[len(LogSaveDir)-1:] != "/" && LogSaveDir[len(LogSaveDir)-1:] != `\`{
		LogSaveDir += "/"
	}
	if ServiceNameLevelSplitStr == "" {
		ServiceNameLevelSplitStr = "@"
	}

	 gyl := &GoRSysLog{
		ServiceName: ServiceName,
		Network:     network,
		Raddr:       raddr,
		priority:    priority[0],
		cache:       sync.Map{},
		ServiceNameLevelSplitStr: ServiceNameLevelSplitStr,
		RSysLogConfDir: RSysLogConfDir,
		LogSaveDir: LogSaveDir,
	}

	if err := initRSysLogConf(ServiceName,ServiceNameLevelSplitStr,RSysLogConfDir,LogSaveDir,priority[0]); err != nil {
		return nil, err
	}

	return gyl,nil
}

// 仅显示在显示器上，不写入文件 Print only on the monitor
func (w *GoRSysLog) Print(arg ...interface{}) {
	_,_ =fmt.Fprint(os.Stdout,arg)
}

// 仅显示在显示器上，不写入文件 Print only on the monitor
func (w *GoRSysLog) Println(arg ...interface{}) {
	_,_ =fmt.Fprintln(os.Stdout,arg)
}

// 仅显示在显示器上，不写入文件 Print only on the monitor
func (w *GoRSysLog) Printf(format string,arg ...interface{}) {
	_,_ =fmt.Fprintf(os.Stdout,format,arg)
}

// 显示后结束进程
func (w *GoRSysLog) Fprint(arg ...interface{}) {
	_,_ =fmt.Fprint(os.Stdout,arg)
	os.Exit(1)
}

// 显示后结束进程
func (w *GoRSysLog) Fprintln(arg ...interface{}) {
	_,_ =fmt.Fprintln(os.Stdout,arg)
	os.Exit(1)
}

// 显示后结束进程
func (w *GoRSysLog) Fprintf(format string,arg ...interface{}) {
	_,_ =fmt.Fprintf(os.Stdout,format,arg)
	os.Exit(1)
}

// Emerg logs a message with severity LOG_EMERG, ignoring the severity
// passed to
func (w *GoRSysLog) Emerg(arg ...interface{}) error {
	writer,err := w.loadOrCacheSyslogWriter(LOG_EMERG)
	if err != nil {
		return err
	}

	return writer.Emerg(splice(arg).String())
}

// Alert logs a message with severity LOG_ALERT, ignoring the severity
// passed to New.
func (w *GoRSysLog) Alert(arg ...interface{}) error {
	writer,err := w.loadOrCacheSyslogWriter(LOG_ALERT)
	if err != nil {
		return err
	}

	return writer.Alert(splice(arg).String())
}

// Crit logs a message with severity LOG_CRIT, ignoring the severity
// passed to New.
func (w *GoRSysLog) Crit(arg ...interface{}) error {
	writer,err := w.loadOrCacheSyslogWriter(LOG_CRIT)
	if err != nil {
		return err
	}

	return writer.Crit(splice(arg).String())
}

// Err logs a message with severity LOG_ERR, ignoring the severity
// passed to New.
func (w *GoRSysLog) Err(arg ...interface{}) error {
	writer,err := w.loadOrCacheSyslogWriter(LOG_ERR)
	if err != nil {
		return err
	}

	return writer.Err(splice(arg).String())
}

// Warning logs a message with severity LOG_WARNING, ignoring the
// severity passed to New.
func (w *GoRSysLog) Warning(arg ...interface{}) error {
	writer,err := w.loadOrCacheSyslogWriter(LOG_WARNING)
	if err != nil {
		return err
	}

	return writer.Warning(splice(arg).String())
}

// Notice logs a message with severity LOG_NOTICE, ignoring the
// severity passed to New.
func (w *GoRSysLog) Notice(arg ...interface{}) error {
	writer,err := w.loadOrCacheSyslogWriter(LOG_NOTICE)
	if err != nil {
		return err
	}

	return writer.Notice(splice(arg).String())
}

// Info logs a message with severity LOG_INFO, ignoring the severity
// passed to New.
func (w *GoRSysLog) Info(arg ...interface{}) error {
	writer,err := w.loadOrCacheSyslogWriter(LOG_INFO)
	if err != nil {
		return err
	}

	return writer.Info(splice(arg).String())
}

// Debug logs a message with severity LOG_DEBUG, ignoring the severity
// passed to New.
func (w *GoRSysLog) Debug(arg ...interface{}) error {
	writer,err := w.loadOrCacheSyslogWriter(LOG_DEBUG)
	if err != nil {
		return err
	}

	return writer.Debug(splice(arg).String())
}

func (w *GoRSysLog) loadOrCacheSyslogWriter(level Priority) (*syslog.Writer,error) {
	if w.priority & level != level {
		w.Println("输出日志失败没有启用当前日志等级：" , getLogLevel(level).Name)
		return nil,errors.New("输出日志失败没有启用当前日志等级：" + getLogLevel(level).Name)
	}
	var ServiceName_level = w.ServiceName + w.ServiceNameLevelSplitStr + getLogLevel(level).Name
	//var err error
	writer,ok := w.getCache(ServiceName_level)
	if !ok {
		fmt.Println("未在创建对象时传递预先定义此等级" + getLogLevel(level).Name +"日志输出：但使用了此等级的日志输出，不可使用预先未定义此等级")
		return nil, appendErr("未在创建对象时传递预先定义此等级" + getLogLevel(level).Name +"日志输出：但使用了此等级的日志输出，不可使用预先未定义此等级")
		//writer,err = syslog.Dial(w.Network,w.Raddr,syslog.Priority( getLogLevel(level).SysLogLevel ),ServiceName_level)
		//if err != nil {
		//	return nil, appendErr("连接syslog异常：",err)
		//}
		//w.setCache(ServiceName_level,writer)
	}

	return writer,nil

}

func (w *GoRSysLog) setCache(key string,val *syslog.Writer)  {
	w.cache.Store(key,val)
}

func (w *GoRSysLog) getCache(key string)  (*syslog.Writer,bool) {
	writer,ok := w.cache.Load(key)
	if !ok {
		return nil,false
	}
	syswriter,ok := writer.(*syslog.Writer)
	if !ok {
		return nil,false
	}

	return syswriter,true
}

func (w *GoRSysLog) Close()  {
	w.cache.Range(func(key, value any) bool {
		val := value.(*syslog.Writer)
		val.Close()
		return true
	})
}

func execCommand(strCommand string) (string,error) {

	cmd := exec.Command("/bin/bash", "-c", strCommand)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout  // 标准输出
	cmd.Stderr = &stderr  // 标准错误
	err := cmd.Run()
	if err != nil {
		return "",appendErr("run err：",err)
	}
	if stderr.Len() == 0{
		err = nil
	}else {
		err = errors.New(stderr.String())
	}

	return stdout.String(),err
}

// 服务名、服务名和等级分割字符、等级、配置文件保存目录、日志输出目录
func initRSysLogConf(ServiceName,ServiceNameLevelSplitStr,RSysLogConfDir,LogSaveDir string,priority Priority) error {

	var confBuf = new(bytes.Buffer)

	confBuf.WriteString("$ModLoad imjournal\n$imjournalRatelimitInterval 0\n$imjournalRatelimitBurst 0\n\n")
	var fullServName,outPutDir string

	for i:=LOG_EMERG;i<=LOG_DEBUG;i*=2{
		isStartLeve := (priority & i) == i
		if !isStartLeve {
			continue
		}
		fullServName = ServiceName + ServiceNameLevelSplitStr + getLogLevel(i).Name
		outPutDir = LogSaveDir + ServiceName + "/" +fullServName + ".log"
		confBuf.WriteString(fmt.Sprintf("if ($programname == '%v') then{\n\t%v\n}\n",fullServName,outPutDir))
	}


	err := os.WriteFile(RSysLogConfDir + ServiceName + ".conf" ,confBuf.Bytes(),0666)
	if err != nil {
		return appendErr("写入rsyslog配置文件失败：检查权限",err)
	}
	_, err = execCommand("systemctl restart rsyslog")
	if err != nil { err = appendErr("重启rsyslog服务失败",err) }

	return err
}

// append error
func appendErr(strOrErr ...interface{}) error  {
	return errors.New(splice(strOrErr).String())
}

// 拼接字符 splice
func splice(strOrErr ...interface{}) *bytes.Buffer {
	var errBuf = new(bytes.Buffer)
	for _, i2 := range strOrErr {
		errBuf.WriteString(fmt.Sprint(i2))
	}
	return errBuf
}

func getLogLevel(level Priority) *levelInfo {
	lock.Lock()
	v := LogLevel[level]
	lock.Unlock()
	return &v
}

//func main()  {
//	// 日志输出在 /var/log/syslog_test/ 目录下
//	gr,err := NewDefault("syslog_test1")
//	if err != nil {
//		log.Fatalln(err)
//	}
//	defer gr.Close()
//	fmt.Println(gr.Emerg("Emerg"))
//	fmt.Println(gr.Alert("Alert"))
//	fmt.Println(gr.Crit("Crit"))
//	fmt.Println(gr.Err("Err"))
//	fmt.Println(gr.Warning("Warning"))
//	fmt.Println(gr.Notice("Notice"))
//	fmt.Println(gr.Info("Info"))
//	fmt.Println(gr.Debug("Debug"))
//
//
//	gr2,err := NewDefault("syslog_test2",LOG_DEBUG | LOG_INFO )
//	if err != nil {
//		log.Fatalln(err)
//	}
//	defer gr2.Close()
//	fmt.Println(gr2.Emerg("Emerg"))
//	fmt.Println(gr2.Alert("Alert"))
//	fmt.Println(gr2.Crit("Crit"))
//	fmt.Println(gr2.Err("Err"))
//	fmt.Println(gr2.Warning("Warning"))
//	fmt.Println(gr2.Notice("Notice"))
//	fmt.Println(gr2.Info("Info"))
//	fmt.Println(gr2.Debug("Debug"))
//}