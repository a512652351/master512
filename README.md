# StratumProxy
<b>声明：此源码仅供学习交流使用，不对您使用造成的后果负责！</b>  
 

## 编译
编译时需使用 tag： ``self_cfg`` 和 ``publish_log``

## Windows 直接下载运行 
https://github.com/ethpoolproxy/stratumproxy/releases

## Linux一键安装

```bash
bash <(curl -s -L https://raw.githubusercontent.com/ethpoolproxy/stratumproxy/master/install.sh)
```

---

### 查看运行情况
```bash
systemctl status stratumproxy
```
或者使用脚本查看日志

---
## Linux手动安装
```bash
wget https://github.com/ethpoolproxy/stratumproxy/releases/download/v1.3.1/stratumproxy_v1.3.1 -O /usr/bin/stratumproxy
wget https://raw.githubusercontent.com/ethpoolproxy/stratumproxy/stratumproxy.service -O /etc/systemd/system/stratumproxy.service
systemctl daemon-reload
systemctl enable --now stratumproxy
```

## 重要说明

```bigquery
 
本软件为0.3%的开发者费用,可以自行抓包验证
该软件系统占用极小,开最便宜的腾讯云服务器即可，脚本自带腾讯云云监控卸载工具
```
