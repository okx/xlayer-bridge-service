Use this tools, you can easily transfer asset between L1 and L2.

## Bridge tools
```
cd ./test/scripts/bridge

./bridge <type> [0: L1->L2 OKB; 1: L1->L2 ETH; 2:L2->L1 OKB; 3: L2->L1 ETH]

```

## Query balance
### Query L2 OKB balance
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