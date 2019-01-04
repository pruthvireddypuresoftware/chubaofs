// Copyright 2018 The Container File System Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package master

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/tiglabs/containerfs/raftstore"
	"github.com/tiglabs/containerfs/util/config"
	"github.com/tiglabs/containerfs/util/exporter"
	"github.com/tiglabs/containerfs/util/log"
	"net/http/httputil"
	"strconv"
	"sync"
)

// configuration keys
const (
	ClusterName       = "clusterName"
	ID                = "id"
	IP                = "ip"
	Port              = "port"
	LogLevel          = "logLevel"
	WalDir            = "walDir"
	StoreDir          = "storeDir"
	GroupID           = 1
	ModuleName        = "master"
	CfgRetainLogs     = "retainLogs"
	DefaultRetainLogs = 20000
)

// Server represents the server in a cluster
type Server struct {
	id           uint64
	clusterName  string
	ip           string
	port         string
	walDir       string
	storeDir     string
	retainLogs   uint64
	leaderInfo   *LeaderInfo
	config       *clusterConfig
	cluster      *Cluster
	rocksDBStore *raftstore.RocksDBStore
	raftStore    raftstore.RaftStore
	fsm          *MetadataFsm
	partition    raftstore.Partition
	wg           sync.WaitGroup
	reverseProxy *httputil.ReverseProxy
}

// NewServer creates a new server
func NewServer() *Server {
	return &Server{}
}

// Start starts a server
func (m *Server) Start(cfg *config.Config) (err error) {
	m.config = newClusterConfig()
	m.leaderInfo = &LeaderInfo{}
	m.reverseProxy = m.newReverseProxy()
	if err = m.checkConfig(cfg); err != nil {
		log.LogError(errors.ErrorStack(err))
		return
	}
	m.rocksDBStore = raftstore.NewRocksDBStore(m.storeDir)
	m.initFsm()
	m.initCluster()
	if err = m.createRaftServer(); err != nil {
		log.LogError(errors.ErrorStack(err))
		return
	}
	m.cluster.partition = m.partition
	m.cluster.idAlloc.partition = m.partition
	m.cluster.scheduleTask()

	// TODO unhandled error
	m.startHTTPService()
	exporter.Init(m.clusterName, ModuleName, cfg)
	m.wg.Add(1)
	return nil
}

// Shutdown closes the server
func (m *Server) Shutdown() {
	m.wg.Done()
}

// Sync waits for the execution termination of the server
func (m *Server) Sync() {
	m.wg.Wait()
}

func (m *Server) checkConfig(cfg *config.Config) (err error) {
	m.clusterName = cfg.GetString(ClusterName)
	m.ip = cfg.GetString(IP)
	m.port = cfg.GetString(Port)
	m.walDir = cfg.GetString(WalDir)
	m.storeDir = cfg.GetString(StoreDir)
	peerAddrs := cfg.GetString(cfgPeers)
	if m.ip == "" || m.port == "" || m.walDir == "" || m.storeDir == "" || m.clusterName == "" || peerAddrs == "" {
		return fmt.Errorf("%v,err:%v", badConfErr, "one of (ip,port,walDir,storeDir,clusterName) is null")
	}
	if m.id, err = strconv.ParseUint(cfg.GetString(ID), 10, 64); err != nil {
		return fmt.Errorf("%v,err:%v", badConfErr, err.Error())
	}
	if err = m.config.parsePeers(peerAddrs); err != nil {
		return
	}
	nodeSetCapacity := cfg.GetString(nodeSetCapacity)
	if nodeSetCapacity != "" {
		if m.config.nodeSetCapacity, err = strconv.Atoi(nodeSetCapacity); err != nil {
			return fmt.Errorf("%v,err:%v", badConfErr, err.Error())
		}
	}
	if m.config.nodeSetCapacity < 3 {
		m.config.nodeSetCapacity = defaultNodeSetCapacity
	}
	retainLogs := cfg.GetString(CfgRetainLogs)
	if retainLogs != "" {
		if m.retainLogs, err = strconv.ParseUint(retainLogs, 10, 64); err != nil {
			return fmt.Errorf("%v,err:%v", badConfErr, err.Error())
		}
	}
	if m.retainLogs <= 0 {
		m.retainLogs = DefaultRetainLogs
	}
	fmt.Println("retainLogs=", m.retainLogs)

	dataPartitionMissSec := cfg.GetString(dataPartitionMissSec)
	if dataPartitionMissSec != "" {
		if m.config.DataPartitionMissSec, err = strconv.ParseInt(dataPartitionMissSec, 10, 0); err != nil {
			return fmt.Errorf("%v,err:%v", badConfErr, err.Error())
		}
	}

	dataPartitionTimeOutSec := cfg.GetString(dataPartitionTimeOutSec)
	if dataPartitionTimeOutSec != "" {
		if m.config.DataPartitionTimeOutSec, err = strconv.ParseInt(dataPartitionTimeOutSec, 10, 0); err != nil {
			return fmt.Errorf("%v,err:%v", badConfErr, err.Error())
		}
	}

	everyLoadDataPartitionCount := cfg.GetString(everyLoadDataPartitionCount)
	if everyLoadDataPartitionCount != "" {
		if m.config.everyLoadDataPartitionCount, err = strconv.Atoi(everyLoadDataPartitionCount); err != nil {
			return fmt.Errorf("%v,err:%v", badConfErr, err.Error())
		}
	}
	if m.config.everyLoadDataPartitionCount <= 40 {
		m.config.everyLoadDataPartitionCount = 40
	}
	if releaseAfterLoadSeconds := cfg.GetString(releaseDataPartitionAfterLoadSeconds); releaseAfterLoadSeconds != "" {
		if m.config.releaseDataPartitionAfterLoadSeconds, err = strconv.ParseInt(releaseAfterLoadSeconds, 10, 64); err != nil {
			return fmt.Errorf("%v,err:%v", badConfErr, err.Error())
		}
	}

	return
}

func (m *Server) createRaftServer() (err error) {
	raftCfg := &raftstore.Config{NodeID: m.id, RaftPath: m.walDir, RetainLogs: m.retainLogs}
	if m.raftStore, err = raftstore.NewRaftStore(raftCfg); err != nil {
		return errors.Annotatef(err, "NewRaftStore failed! id[%v] walPath[%v]", m.id, m.walDir)
	}
	fmt.Println(m.config.peers)
	partitionCfg := &raftstore.PartitionConfig{
		ID:      GroupID,
		Peers:   m.config.peers,
		Applied: m.fsm.applied,
		SM:      m.fsm,
	}
	if m.partition, err = m.raftStore.CreatePartition(partitionCfg); err != nil {
		return errors.Annotate(err, "CreatePartition failed")
	}
	return
}
func (m *Server) initFsm() {
	m.fsm = newMetadataFsm(m.rocksDBStore)
	m.fsm.registerLeaderChangeHandler(m.handleLeaderChange)
	m.fsm.registerPeerChangeHandler(m.handlePeerChange)

	// TODO explain
	m.fsm.registerApplyHandler(m.handleApply)
	m.fsm.registerApplySnapshotHandler(m.handleApplySnapshot)
	m.fsm.restore()
}

func (m *Server) initCluster() {
	m.cluster = newCluster(m.clusterName, m.leaderInfo, m.fsm, m.partition, m.config)
	m.cluster.retainLogs = m.retainLogs
	m.loadMetadata()
}
