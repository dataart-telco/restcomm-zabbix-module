package main

import (
	"encoding/json"
	"github.com/cavaliercoder/g2z"
	"errors"
	"reflect"
	"github.com/scalingdata/gcfg"
	"io"
	"io/ioutil"
	"os"
)

type Config struct {

	Main struct {
		MarathonHost string
		AppId string
		LogLevel string
	}

	Restcomm struct {
		Port int
		User string
		Pswd string
		MaxCalls int
	}
}

/*type State struct {
	LastState *RestcommCluster
	Test2 RestcommCluster
}*/

type ZabbixAgent struct {
	LastState *RestcommCluster
}

func (self *ZabbixAgent) Write(data *RestcommCluster) {
	self.toFile(data)
}

func (self *ZabbixAgent) toFile(data *RestcommCluster) {
	bytes, _ := json.Marshal(data)
	//Info.Printf("ZabbixAgent %p dump: %s", self, string(bytes))
	err := ioutil.WriteFile("/tmp/last_data.json", bytes, 0777)
	if err != nil {
		Error.Println("can not write data file", err)
	}
}

func (self *ZabbixAgent) fromFile() {
	bytes, err := ioutil.ReadFile("/tmp/last_data.json")
	if err != nil {
		Error.Println("can not read data file", err)
		return
	}
	Info.Printf("ZabbixAgent %p read: %s", self, string(bytes))
	var data RestcommCluster
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		Error.Println("can not decaode file", err)
		return
	}
	self.LastState = &data
}

func (self *ZabbixAgent) GetNodes() ([]RestcommNode, error){
	Info.Println("GetNodes:", self.LastState,)
	if self.LastState == nil {
		return nil, errors.New("no data")
	}
	return self.LastState.Nodes, nil
}

func (self *ZabbixAgent) GetMetrics(nodeId string) (*RestcommMetrics, error){
	if self.LastState == nil {
		return nil, errors.New("GetMetrics: no data")
	}
	for _, node := range self.LastState.Nodes {
		if node.InstanceId == nodeId {
			return &node.Metrics, nil
		}
	}
	return nil, errors.New("can not find node: " + nodeId)
}

func (self *ZabbixAgent) Discovery(request *g2z.AgentRequest) (g2z.DiscoveryData, error) {
	Info.Println("Discovery local")
	self.fromFile();
	nodes, err := self.GetNodes()
	if err != nil {
		Error.Print("Discovery error:", err)
		return nil, err
	}
	Info.Println("Discovery local nodes:", nodes)
	discovery := make(g2z.DiscoveryData, 0, len(nodes))
	for i, node := range nodes {
		Info.Println("Discovery local for:", i)
		item := make(g2z.DiscoveryItem)
		item["APP_NAME"] = cfg.Main.AppId
		item["TASK_ID"] = node.TaskId
		item["INSTANCE_ID"] = node.InstanceId
		discovery = append(discovery, item)
	}
	Info.Println("Discovery result:", discovery)
	return discovery, nil
}

func (self *ZabbixAgent) Metrics(request *g2z.AgentRequest) (uint64, error) {
	Info.Println("Metrics:", request)
	self.fromFile();
	if len(request.Params) < 2 {
		Error.Print("invalid params count")
		return 0, errors.New("invalid params count: Expected 2 args")
	}
	nodeId := request.Params[0]
	fieldName := request.Params[1]
	Trace.Print("Get Metrics from", nodeId, "for", fieldName)
	node, err := self.GetMetrics(nodeId)
	if err != nil {
		return 0, err
	}
	defer func() {
		if r := recover(); r != nil {
			Error.Print("Catch exception", r)
		}
	}()
	ref := reflect.ValueOf(node)
	field := reflect.Indirect(ref).FieldByName(fieldName)
	return uint64(field.Int()), nil
}

var zabbixAgent = &ZabbixAgent{}
var cfg = &Config{}
var monitor *MonitorAgent

func init() {

	err := gcfg.ReadFileInto(cfg, "zabbix-agent.ini")
	if err != nil {
		Error.Println("can not read ini file", err)
		Info.Println("Use default settings")
		monitor = &MonitorAgent{marathonHost: "127.0.0.1:8080", appId: "restcomm",
			restcommPort: 8080, restcommUser: "ACae6e420f425248d6a26948c17a9e2acf", restcommPswd: "42d8aa7cde9c78c4757862d84620c335", restcommMaxCalls: 50,
			Writer: zabbixAgent}
	} else {
		monitor = &MonitorAgent{marathonHost: cfg.Main.MarathonHost, appId: cfg.Main.AppId,
			restcommPort: cfg.Restcomm.Port, restcommUser: cfg.Restcomm.User, restcommPswd: cfg.Restcomm.Pswd, restcommMaxCalls: cfg.Restcomm.MaxCalls,
			Writer: zabbixAgent}

		var traceHandle io.Writer
		if cfg.Main.LogLevel == "TRACE" {
			traceHandle = os.Stdout
		} else {
			traceHandle = ioutil.Discard
		}
		InitLog(traceHandle, os.Stdout, os.Stdout, os.Stderr)
	}
	monitor.StartWorker(5)
	g2z.RegisterDiscoveryItem("restcomm.discovery", "Restcomm Instances", zabbixAgent.Discovery)
	g2z.RegisterUint64Item("restcomm.metrics", "Restcomm Metrics", zabbixAgent.Metrics)
	g2z.RegisterUninitHandler(func() error {
		monitor.StopWorker();
		return nil
	})
}
