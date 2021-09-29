# PlatON JOB TASK

## How to use?

### git clone code

### create config dir

```
cd platonjob
mkdir -pv config
```

### platon job task settings

-   chainId: 100 # platon 主网/alaya 链 ID
-   async: false # true 异步操作，本地节点打包，出块时操作，gas 费用为 0 | false：同步操作，实时获取当前 gasPrice 操作
-   rawURL: http://127.0.0.1:6789 # 节点连接地址
-   arp: lat # lat 或 atp
-   rewardBlock: 10000 # 结算周期到 10000 开始执行获取委托收益，可以默认不需要改动
-   rewardGasLimit: 50000 # 领取委托收益 gaslimit，可以默认不需要改动
-   delegateBlock: 3000 # 结算周期到 3000 开始执行委节点，可以默认不需要改动
-   delegateGasLimit: 50000 # 委托节点 gaslimit，可以默认不需要改动
-   minDelegate: 10 # 最小质押金额，默认 alaya 是 1，platon 是 10，可自定义
-   dstAddr: "" # 汇总地址，暂时未实现
-   addrs: # 地址列表

### change and copy example-config.yaml under config dir

```
cp ./example-config.yaml ./config/config.yaml
```

### build

```
CGO_ENABLED=0 GOOS=linux go build
```

#### run

```
./platonjob
```
