package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"gitlab.eoitek.net/EOI/ckman/model"
	"gitlab.eoitek.net/EOI/ckman/service/clickhouse"
	"gitlab.eoitek.net/EOI/ckman/service/zookeeper"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	ZkStatusDefaultPort int = 8080
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type ZookeeperController struct {
}

func NewZookeeperController() *ZookeeperController {
	ck := &ZookeeperController{}
	return ck
}

// @Summary 获取Zookeeper集群状态
// @Description 获取Zookeeper集群状态
// @version 1.0
// @Security ApiKeyAuth
// @Param clusterName path string true "cluster name" default(test)
// @Success 200 {string} json "{"code":200,"msg":"ok","data":[{"host":"192.168.102.116","version":"3.6.2","server_state":"follower","peer_state":"following - broadcast","avg_latency":0.4929,"approximate_data_size":141979,"znode_count":926}]}"
// @Router /api/v1/zk/status/{clusterName} [get]
func (zk *ZookeeperController) GetStatus(c *gin.Context) {
	var conf model.CKManClickHouseConfig

	clusterName := c.Param(ClickHouseClusterPath)
	con, ok := clickhouse.CkClusters.Load(clusterName)
	if !ok {
		model.WrapMsg(c, model.GET_ZK_STATUS_FAIL, model.GetMsg(model.GET_ZK_STATUS_FAIL),
			fmt.Sprintf("cluster %s does not exist", clusterName))
		return
	}

	conf = con.(model.CKManClickHouseConfig)
	zkList := make([]model.ZkStatusRsp, len(conf.ZkNodes))
	for index, node := range conf.ZkNodes {
		tmp := model.ZkStatusRsp{
			Host: node,
		}
		body, err := getZkStatus(node, ZkStatusDefaultPort)
		if err != nil {
			model.WrapMsg(c, model.GET_ZK_STATUS_FAIL, model.GetMsg(model.GET_ZK_STATUS_FAIL),
				fmt.Sprintf("get zookeeper node %s satus fail: %v", node, err))
			return
		}
		json.Unmarshal(body, &tmp)
		tmp.Version = tmp.Version[:strings.Index(tmp.Version, "--")]
		zkList[index] = tmp
	}

	model.WrapMsg(c, model.SUCCESS, model.GetMsg(model.SUCCESS), zkList)
}

func getZkStatus(host string, port int) ([]byte, error) {
	url := fmt.Sprintf("http://%s:%d/commands/mntr", host, port)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("%s", response.Status)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// @Summary 从Zookeeper中获取复制表状态
// @Description 从Zookeeper中获取复制表状态
// @version 1.0
// @Security ApiKeyAuth
// @Param clusterName path string true "cluster name" default(test)
// @Success 200 {string} json "{"code":200,"msg":"ok","data":[{"host":"192.168.102.116","version":"3.6.2","server_state":"follower","peer_state":"following - broadcast","avg_latency":0.4929,"approximate_data_size":141979,"znode_count":926}]}"
// @Router /api/v1/zk/replicated_table/{clusterName} [get]
func (zk *ZookeeperController) GetReplicatedTableStatus(c *gin.Context) {
	var conf model.CKManClickHouseConfig

	clusterName := c.Param(ClickHouseClusterPath)
	con, ok := clickhouse.CkClusters.Load(clusterName)
	if !ok {
		model.WrapMsg(c, model.GET_ZK_TABLE_STATUS_FAIL, model.GetMsg(model.GET_ZK_TABLE_STATUS_FAIL),
			fmt.Sprintf("cluster %s does not exist", clusterName))
		return
	}
	conf = con.(model.CKManClickHouseConfig)

	zkService, err := zookeeper.GetZkService(clusterName)
	if err != nil {
		model.WrapMsg(c, model.GET_ZK_TABLE_STATUS_FAIL, model.GetMsg(model.GET_ZK_TABLE_STATUS_FAIL),
			fmt.Sprintf("get zookeeper service fail: %v", err))
		return
	}

	tables, err := zkService.GetReplicatedTableStatus(&conf)
	if err != nil {
		model.WrapMsg(c, model.GET_ZK_TABLE_STATUS_FAIL, model.GetMsg(model.GET_ZK_TABLE_STATUS_FAIL), err)
		return
	}

	header := make([][]string, len(conf.Shards))
	for shardIndex, shard := range conf.Shards {
		replicas := make([]string, len(shard.Replicas))
		for replicaIndex, replica := range shard.Replicas {
			replicas[replicaIndex] = replica.HostName
		}
		header[shardIndex] = replicas
	}
	resp := model.ZkReplicatedTableStatusRsp{
		Header: header,
		Tables: tables,
	}

	model.WrapMsg(c, model.SUCCESS, model.GetMsg(model.SUCCESS),resp)
}