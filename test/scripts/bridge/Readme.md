Use this tools, you can easily transfer asset between L1 and L2.

## Prerequisites
The tutorial for current version of the environment requires Foundryup, check out the links provided below:
- https://book.getfoundry.sh/getting-started/installation

## Bridge tools
```
cd x1-bridge-service

# setup all components 
make run

# transfer asset between L1 and L2.
cd ./test/scripts/bridge

# summury
go run main.go [0: L1->L2 OKB; 1: L1->L2 ETH; 2:L2->L1 OKB; 3: L2->L1 ETH]

# bridge OKB L1->L2, and query L2 OKB balance
go run main.go 0
cast balance 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8123

# bridge ETH L1->L2, and query L2 WETH balance
go run main.go 1
cast call 0x82109a709138A2953C720D3d775168717b668ba6 "balanceOf(address)" 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8123

# bridge OKB L2->L1, and query L1 OKB balance
go run main.go 2
cast call 0x82109a709138A2953C720D3d775168717b668ba6 "balanceOf(address)" 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8545

# bridge ETH L2->L1, and query L1 ETH balance
go run main.go 3
cast balance 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8545

```

## Query balance
### Query L2 OKB Balance
```
cast balance 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8123
``` 

### Query L2 WETH Balance
```
cast call 0x82109a709138A2953C720D3d775168717b668ba6 "balanceOf(address)" 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8123
```

### Query L1 OKB Balance
```
cast call 0x82109a709138A2953C720D3d775168717b668ba6 "balanceOf(address)" 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8545
```

### Query L1 ETH Balance
```
cast balance 0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225 --rpc-url http://127.0.0.1:8545
```