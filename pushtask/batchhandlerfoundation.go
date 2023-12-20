package pushtask

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/0xPolygonHermez/zkevm-node/hex"
	"github.com/0xPolygonHermez/zkevm-node/jsonrpc/client"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

const (
	queryLatestCommitBatchNumMethod = ""
	queryLatestVerifyBatchNumMethod = ""
)

// BatchInfo just simple info, can get more from https://okg-block.larksuite.com/wiki/DqM0wLcm1i5fztk5iKAu2Tw6spg
type BatchInfo struct {
	number string
	blocks []string
}

func QueryLatestCommitBatch(rpcUrl string) (uint64, error) {
	return queryLatestBatchNum(rpcUrl, queryLatestCommitBatchNumMethod)
}

func QueryLatestVerifyBatch(rpcUrl string) (uint64, error) {
	return queryLatestBatchNum(rpcUrl, queryLatestVerifyBatchNumMethod)
}

func queryLatestBatchNum(rpcUrl string, methodName string) (uint64, error) {
	response, err := client.JSONRPCCall(rpcUrl, methodName)
	if err != nil {
		log.Errorf("query for %v error: %v", methodName, err)
		return 0, errors.Wrap(err, fmt.Sprintf("query %v error", methodName))
	}

	if response.Error != nil {
		log.Errorf("query for %v, back failed data, %v, %v", methodName, response.Error.Code, response.Error.Message)
		return 0, errors.Wrap(err, fmt.Sprintf("query %v failed", methodName))
	}

	var result string
	err = json.Unmarshal(response.Result, &result)
	if err != nil {
		log.Errorf("query for %v, parse json error: %v", methodName, err)
		return 0, errors.Wrap(err, fmt.Sprintf("query %v, parse json error", methodName))
	}

	bigBatchNumber := hex.DecodeBig(result)
	latestBatchNum := bigBatchNumber.Uint64()
	return latestBatchNum, nil
}

func QueryMaxBlockHashByBatchNum(rpcUrl string, batchNum uint64) (string, error) {
	response, err := client.JSONRPCCall(rpcUrl, "zkevm_getBatchByNumber", batchNum)
	if err != nil {
		log.Errorf("query for %v error: %v", "zkevm_getBatchByNumber", err)
		return "", errors.Wrap(err, fmt.Sprintf("query zkevm_getBatchByNumber error"))
	}

	if response.Error != nil {
		log.Errorf("query for zkevm_getBatchByNumber failed, %v, %v", response.Error.Code, response.Error.Message)
		return "", errors.Wrap(err, fmt.Sprintf("query zkevm_getBatchByNumber failed"))
	}

	var result BatchInfo
	err = json.Unmarshal(response.Result, &result)
	if err != nil {
		log.Errorf("query for %v, parse json error: %v", "zkevm_getBatchByNumber", err)
		return "", errors.Wrap(err, fmt.Sprintf("query zkevm_getBatchByNumber, parse json error"))
	}
	if result.blocks == nil || len(result.blocks) == 0 {
		log.Errorf("query for %v, blocks is empty: %v", "zkevm_getBatchByNumber", err)
		return "", nil
	}
	return result.blocks[len(result.blocks)-1], nil
}

func QueryBlockNumByBlockHash(ctx context.Context, client *ethclient.Client, blockHash string) (uint64, error) {
	block, err := client.BlockByHash(ctx, common.HexToHash(blockHash))
	if err != nil {
		log.Errorf("query for blockByHash error: %v", err)
		return 0, errors.Wrap(err, fmt.Sprintf("query blockByHash error"))
	}
	return block.Number().Uint64(), nil
}
