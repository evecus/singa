# singa

基于 sing-box 的透明代理管理面板，nftables 规则严格移植自 [v2rayA](https://github.com/v2rayA/v2rayA)。

## 功能

- 上传 `config.json`，自动解析 inbound 端口
- 透明代理模式：`tproxy`（TCP+UDP）、`redirect`（TCP）、`tun`、`system_proxy`
- 局域网代理开关（自动开启 `ip_forward`）
- nftables 规则下发 / 清理（进程退出自动清理）
- 实时日志流（SSE）

## 使用

### 前置要求

- Linux，需要 `root` 权限（nftables 操作）
- 已安装 `/usr/bin/sing-box`
- 已安装 `nft`（nftables）

### 运行

```bash
# 下载对应架构的二进制
chmod +x singa-linux-amd64
sudo ./singa-linux-amd64
```

面板默认监听 `:8080`，浏览器打开 `http://localhost:8080`。

### 目录结构

```
singa          # 二进制
data/
  config.json  # 上传的 sing-box 配置
  run/         # sing-box 工作目录（-D 参数）
  singa-nft.conf  # 运行时生成的 nftables 规则文件（自动管理）
```

### config.json 要求

你的 `config.json` 中需要有对应模式的 inbound，例如 tproxy 模式：

```json
{
  "inbounds": [
    {
      "type": "tproxy",
      "tag": "tproxy-in",
      "listen": "::",
      "listen_port": 7893
    }
  ]
}
```

redirect 模式同理，`"type": "redirect"`。tun / system_proxy 模式不需要特定 inbound。

## 构建

```bash
# 1. 构建前端
cd web && npm install && npm run build && cd ..

# 2. 构建二进制
go mod tidy
go build -trimpath -ldflags="-s -w" -o singa .
```

或直接推送 tag 触发 GitHub Actions：

```bash
git tag v1.0.0
git push origin v1.0.0
```

## nftables 规则说明

规则逻辑完全移植自 v2rayA：

- **tproxy**：使用 `fwmark 0x40/0xc0` + 策略路由 table 100，PREROUTING/OUTPUT hook，connmark 保持连接状态，DNS 强制走代理防污染，本机接口 IP 动态写入 `@interface` set 防环路
- **redirect**：NAT REDIRECT，私有/保留地址段硬编码白名单，本机接口 IP 动态维护
- 两种模式均支持 IPv6（自动检测）
- 进程退出（包括被 kill）时自动执行 `nft delete table inet v2raya` 并清理策略路由
