package main

import (
	"encoding/json"
	"errors"
	"github.com/dataart-telco/g2z"
	. "github.com/dataart-telco/monitoring/restcomm"
	. "github.com/dataart-telco/monitoring/log"
	"github.com/scalingdata/gcfg"
	"io"
	"io/ioutil"
	"os"
	"reflect"
)

type Config struct {
	Main struct {
		MarathonHost      string
		AppId             string
		LogLevel          string
		CollectorInterval int
	}

	Restcomm struct {
		Port     int
		User     string
		Pswd     string
		MaxCalls int
	}
}

type ZabbixAgent struct {
	LastState *RestcommCluster
}

func (self *ZabbixAgent) DataCollected(data *RestcommCluster) {
	self.toFile(data)
}

func (self *ZabbixAgent) toFile(data *RestcommCluster) {
	bytes, _ := json.Marshal(data)
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
	var data RestcommCluster
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		Error.Println("can not decaode file", err)
		return
	}
	self.LastState = &data
}

func (self *ZabbixAgent) GetNodes() ([]RestcommNode, error) {
	Trace.Println("GetNodes:", self.LastState)
	if self.LastState == nil {
		return nil, errors.New("no data")
	}
	return self.LastState.Nodes, nil
}

func (self *ZabbixAgent) GetMetrics(nodeId string) (*RestcommMetrics, error) {
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
	Trace.Println("Discovery local")
	self.fromFile()
	nodes, err := self.GetNodes()
	if err != nil {
		Error.Print("Discovery error:", err)
		return nil, err
	}
	discovery := make(g2z.DiscoveryData, 0, len(nodes))
	for _, node := range nodes {
		item := make(g2z.DiscoveryItem)
		item["APP_NAME"] = cfg.Main.AppId
		item["TASK_ID"] = node.TaskId
		item["INSTANCE_ID"] = node.InstanceId
		discovery = append(discovery, item)
	}
	Trace.Println("Discovery result:", discovery)
	return discovery, nil
}

func getUintVal(self *RestcommMetrics, fieldName string) (val uint64, err error) {
	defer func() {
		if r := recover(); r != nil {
			Error.Print("Catch exception", r)
			err = errors.New("field " + fieldName + " does not exist")
			val = 0
		}
	}()
	ref := reflect.ValueOf(self)
	field := reflect.Indirect(ref).FieldByName(fieldName)
	val = uint64(field.Int())
	return val, nil

}

func (self *ZabbixAgent) Metrics(request *g2z.AgentRequest) (uint64, error) {
	Trace.Println("Metrics:", request)
	self.fromFile()
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
	return getUintVal(node, fieldName)
}

func (self *ZabbixAgent) ClusterMetrics(request *g2z.AgentRequest) (float64, error) {
	Trace.Println("Cluster Metrics:", request)
	self.fromFile()
	if len(request.Params) < 1 {
		Error.Print("invalid params count")
		return 0, errors.New("invalid params count: Expected 1 args")
	}
	fieldName := request.Params[0]
	Trace.Print("Get Cluster Metrics for", fieldName)

	if self.LastState == nil {
		return 0.0, errors.New("Get Cluster Metrics: no data")
	}
	sum := 0.0
	count := float64(len(self.LastState.Nodes))
	for _, node := range self.LastState.Nodes {
		val, err := getUintVal(&node.Metrics, fieldName)
		if err != nil {
			return 0, errors.New("Get Cluster Metrics: no field - " + fieldName)
		}
		sum += float64(val)
	}
	return sum / count, nil
}

func (self *ZabbixAgent) ClusterSize(request *g2z.AgentRequest) (uint64, error) {
	Trace.Println("ClusterSize")
	self.fromFile()
	state := self.LastState
	if state == nil || state.Nodes == nil {
		return uint64(0), nil
	}
	return uint64(len(state.Nodes)), nil
}

var zabbixAgent = &ZabbixAgent{}
var cfg = &Config{}
var monitor *MonitorAgent

func init() {
	err := gcfg.ReadFileInto(cfg, "zabbix-agent.ini")
	if err != nil {
		Error.Println("can not read ini file", err)
		Info.Println("Use default settings")
		monitor = &MonitorAgent{MarathonHost: "127.0.0.1:8080", AppId: "restcomm",
			CollectorInterval: 10,
			Callback: zabbixAgent}

		monitor.Restcomm.Port = 8080
		monitor.Restcomm.User = "ACae6e420f425248d6a26948c17a9e2acf"
		monitor.Restcomm.Pswd = "42d8aa7cde9c78c4757862d84620c335"
		monitor.Restcomm.MaxCalls = 50
	} else {
		monitor = &MonitorAgent{MarathonHost: cfg.Main.MarathonHost, AppId: cfg.Main.AppId,
			CollectorInterval: cfg.Main.CollectorInterval,
			Callback: zabbixAgent}
		monitor.Restcomm.Port = cfg.Restcomm.Port
		monitor.Restcomm.User = cfg.Restcomm.User
		monitor.Restcomm.Pswd = cfg.Restcomm.Pswd
		monitor.Restcomm.MaxCalls = cfg.Restcomm.MaxCalls

		var traceHandle io.Writer
		if cfg.Main.LogLevel == "TRACE" {
			traceHandle = os.Stdout
		} else {
			traceHandle = ioutil.Discard
		}
		InitLog(traceHandle, os.Stdout, os.Stdout, os.Stderr)
	}

	monitor.StartWorker()
	g2z.RegisterDiscoveryItem("restcomm.discovery", "Restcomm Instances", zabbixAgent.Discovery)
	g2z.RegisterUint64Item("restcomm.metrics", "Instance Metrics", zabbixAgent.Metrics)
	g2z.RegisterDoubleItem("restcomm.cluster.metrics", "Cluster Metrics", zabbixAgent.ClusterMetrics)
	g2z.RegisterUint64Item("restcomm.cluster.size", "Cluster size", zabbixAgent.ClusterSize)

	g2z.RegisterUninitHandler(func() error {
		monitor.StopWorker()
		return nil
	})
}

func main(){
	
}