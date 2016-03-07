package main

import (
	"encoding/json"
	"fmt"
)

type MesosTask struct {
	Id   string `json:"id"`
	Host string `json:"host"`
}

type MesosResponse struct {
	Tasks []MesosTask `json:"tasks"`
}

type RestcommMetrics struct {
	LiveCalls             int
	LiveOutgoingCalls     int
	LiveIncomingCalls     int
	TotalCallsSinceUptime int
	CompletedCalls        int
	FailedCalls           int
}

type RestcommNode struct {
	InstanceId string
	TaskId     string
	Metrics    RestcommMetrics
}

type RestcommCluster struct {
	Nodes []RestcommNode
}

type AgentWriter interface {
	Write(data RestcommCluster)
}

type MonitorAgent struct {
	marathonHost string
	appId        string

	restcommUser     string
	restcommPswd     string
	restcommPort     int
	restcommMaxCalls int

	stopWorker chan int
	Writer     AgentWriter
}

func (self *MonitorAgent) GetClusterNodes() (*MesosResponse, error) {
	_, body, err := Get("http://" + self.marathonHost + "/v2/apps/" + self.appId + "/tasks")
	if err != nil {
		return nil, err
	}
	Trace.Println("Get tasks for", self.appId, ": ", string(body))

	var respData MesosResponse
	err = json.Unmarshal(body, &respData)
	if err != nil {
		return nil, err
	}
	return &respData, nil
}

func (self *MonitorAgent) CollectClusterMetrics(tasks *MesosResponse) (*RestcommCluster, error) {
	tasksCount := len(tasks.Tasks)
	Trace.Println("CollectClusterMetrics: tasksCount:", tasksCount)
	nodes := make([]RestcommNode, 0, tasksCount)
	for _, e := range tasks.Tasks {
		data, err := self.GetRestCommCallStat(e.Host)
		if err != nil {
			Error.Println("Get restcomm metrics error:", err)
			continue
		}
		data.TaskId = e.Id
		nodes = append(nodes, *data)
	}
	Trace.Println("CollectClusterMetrics: len(nodes):", len(nodes))
	return &RestcommCluster{nodes}, nil
}

func (self *MonitorAgent) GetRestCommCallStat(host string) (*RestcommNode, error) {

	url := fmt.Sprintf("http://%s:%s@%s:%d/restcomm/2012-04-24/Accounts/%s/Supervisor.json/metrics",
		self.restcommUser, self.restcommPswd, host, self.restcommPort, self.restcommUser)

	Trace.Println("Try get data by url:", url)

	_, body, err := Get(url)
	if err != nil {
		return nil, err
	}
	Trace.Println("RestcommMetrics:", string(body))

	var restcommData RestcommNode
	json.Unmarshal(body, &restcommData)

	return &restcommData, nil
}

func (self *MonitorAgent) CollectMetrics() {
	Trace.Println("CollectMetrics")
	tasks, err := self.GetClusterNodes()
	if err != nil {
		Error.Print("GetClusterNodes error", err)
		return
	}
	clusterInfo, err := self.CollectClusterMetrics(tasks)
	if err != nil {
		Error.Print("CollectClusterMetrics error", err)
		return
	}
	self.Writer.Write(*clusterInfo)
}

func (self *MonitorAgent) StartWorker(period int) {
	Info.Println("StartWorker")
	do := func() {
		self.CollectMetrics()
	}
	self.stopWorker = schedule(period, do)
}

func (self *MonitorAgent) StopWorker() {
	Info.Println("StopWorker")
	if self.stopWorker == nil {
		return
	}
	self.stopWorker <- 1
	self.stopWorker = nil
}
