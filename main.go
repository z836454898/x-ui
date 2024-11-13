package main

import (
	"flag"
	"fmt"
	"github.com/op/go-logging"
	"log"
	"os"
	"os/signal"
	"syscall"
	_ "unsafe"
	"x-ui/config"
	"x-ui/database"
	"x-ui/logger"
	"x-ui/v2ui"
	"x-ui/web"
	"x-ui/web/global"
	"x-ui/web/service"
)

func runWebServer() {
	log.Printf("%v %v", config.GetName(), config.GetVersion())

	switch config.GetLogLevel() {
	case config.Debug:
		logger.InitLogger(logging.DEBUG)
	case config.Info:
		logger.InitLogger(logging.INFO)
	case config.Warn:
		logger.InitLogger(logging.WARNING)
	case config.Error:
		logger.InitLogger(logging.ERROR)
	default:
		log.Fatal("unknown log level:", config.GetLogLevel())
	}

	err := database.InitDB(config.GetDBPath())
	if err != nil {
		log.Fatal(err)
	}

	var server *web.Server

	server = web.NewServer()
	global.SetWebServer(server)
	err = server.Start()
	if err != nil {
		log.Println(err)
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGKILL)
	for {
		sig := <-sigCh

		switch sig {
		case syscall.SIGHUP:
			err := server.Stop()
			if err != nil {
				logger.Warning("stop server err:", err)
			}
			server = web.NewServer()
			global.SetWebServer(server)
			err = server.Start()
			if err != nil {
				log.Println(err)
				return
			}
		default:
			server.Stop()
			return
		}
	}
}

func resetSetting() {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	settingService := service.SettingService{}
	err = settingService.ResetSettings()
	if err != nil {
		fmt.Println("reset setting failed:", err)
	} else {
		fmt.Println("reset setting success")
	}
}

func updateSetting(port int, username string, password string) {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	settingService := service.SettingService{}

	if port > 0 {
		err := settingService.SetPort(port)
		if err != nil {
			fmt.Println("set port failed:", err)
		} else {
			fmt.Printf("set port %v success", port)
		}
	}
	if username != "" || password != "" {
		userService := service.UserService{}
		err := userService.UpdateFirstUser(username, password)
		if err != nil {
			fmt.Println("set username and password failed:", err)
		} else {
			fmt.Println("set username and password success")
		}
	}
}

func main() {
	// 检查命令行参数的数量。如果参数少于2个，则调用runWebServer()函数来启动Web服务器，并返回以终止程序的进一步执行
	if len(os.Args) < 2 {
		runWebServer()
		return
	}
	// 将命令行标志 -v 绑定到 showVersion 变量。如果在运行程序时使用了 -v 标志，showVersion 将被设置为 true。标志的默认值是 false，描述信息是 "show version"
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "show version")
	// 解析命令行参数。flag.Parse()函数解析命令行参数，并将它们保存到相应的变量中
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	v2uiCmd := flag.NewFlagSet("v2-ui", flag.ExitOnError)
	// 设置 db sql lite 配置文件路径
	var dbPath string
	v2uiCmd.StringVar(&dbPath, "db", "/etc/v2-ui/v2-ui.db", "set v2-ui db file path")
	// 设置 setting 配置
	settingCmd := flag.NewFlagSet("setting", flag.ExitOnError)
	var port int
	var username string
	var password string
	var reset bool
	// 将命令行标志 -reset 绑定到 reset 变量。如果在运行程序时使用了 -reset 标志，reset 将被设置为 true。标志的默认值是 false，描述信息是 "reset all setting"
	settingCmd.BoolVar(&reset, "reset", false, "reset all setting")
	// 将命令行标志 -port 绑定到 port 变量。如果在运行程序时使用了 -port 标志，port 将被设置为指定的值。标志的默认值是 0，描述信息是 "set panel port"
	settingCmd.IntVar(&port, "port", 0, "set panel port")
	// 将命令行标志 -username 绑定到 username 变量。如果在运行程序时使用了 -username 标志，username 将被设置为指定的值。标志的默认值是 ""，描述信息是 "set login username"
	settingCmd.StringVar(&username, "username", "", "set login username")
	// 将命令行标志 -password 绑定到 password 变量。如果在运行程序时使用了 -password 标志，password 将被设置为指定的值。标志的默认值是 ""，描述信息是 "set login password"
	settingCmd.StringVar(&password, "password", "", "set login password")
	// 重写 flag.Usage 函数，添加命令行参数的描述信息
	oldUsage := flag.Usage
	flag.Usage = func() {
		oldUsage()
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("    run            run web panel")
		fmt.Println("    v2-ui          migrate form v2-ui")
		fmt.Println("    setting        set settings")
	}

	flag.Parse()
	if showVersion {
		fmt.Println(config.GetVersion())
		return
	}

	switch os.Args[1] {
	case "run":
		err := runCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		runWebServer()
	case "v2-ui":
		err := v2uiCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		err = v2ui.MigrateFromV2UI(dbPath)
		if err != nil {
			fmt.Println("migrate from v2-ui failed:", err)
		}
	case "setting":
		err := settingCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		if reset {
			resetSetting()
		} else {
			updateSetting(port, username, password)
		}
	default:
		fmt.Println("except 'run' or 'v2-ui' or 'setting' subcommands")
		fmt.Println()
		runCmd.Usage()
		fmt.Println()
		v2uiCmd.Usage()
		fmt.Println()
		settingCmd.Usage()
	}
}
