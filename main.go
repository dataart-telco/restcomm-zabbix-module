package main

import (
/*	"flag"
	"io"*/
	"os"
	/*"io/ioutil"*/
	"os/signal"
	"syscall"
	"time"
)

func WaitCtrlC() {
	var signal_channel chan os.Signal
	signal_channel = make(chan os.Signal, 2)
	signal.Notify(signal_channel, os.Interrupt, syscall.SIGTERM)
	<-signal_channel
}

func schedule(step int, what func()) chan int {
	ticker := time.NewTicker(time.Duration(step) * time.Second)
	quit := make(chan int)
	go func() {
		for {
			select {
				case <- quit:
					return
				case <-ticker.C:
					what()
			}
		}
	}()
	return quit
}

func main() {
	/*monitorHost := flag.String("url", "127.0.0.1", "Monitor server")
	appId := flag.String("appId", "restcomm", "App id")
	marathonHost := flag.String("m", "127.0.0.1:8080", "Marathon host")

	rPort := flag.Int("rPort", 8090, "Restcomm Port")
	rUser := flag.String("rUser", "ACae6e420f425248d6a26948c17a9e2acf", "Restcomm user")
	rPswd := flag.String("rPswd", "42d8aa7cde9c78c4757862d84620c335", "Restcomm password")
	maxCalls := flag.Int("max", 50, "Max calls")

	l := flag.String("l", "INFO", "Log level: TRACE, INFO")

	flag.Parse()

	var traceHandle io.Writer
	if *l == "TRACE" {
		traceHandle = os.Stdout
	} else {
		traceHandle = ioutil.Discard
	}
	InitLog(traceHandle, os.Stdout, os.Stdout, os.Stderr)

	Info.Println("Start agent with host =", *monitorHost, " and appId =", *appId, "| period 5 sec", "| marathon:", *marathonHost)

	agent := &MonitorAgent{marathonHost: *marathonHost, appId: *appId,
		restcommPort: *rPort, restcommUser: *rUser, restcommPswd: *rPswd, restcommMaxCalls: *maxCalls,
		Writer: &ZabbixAgent{}}

	agent.StartWorker(5)*/
	schedule(5, func(){
			zabbixAgent.Discovery(nil)
		})
	WaitCtrlC()
}

