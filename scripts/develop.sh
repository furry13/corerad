#!/bin/bash
set -x

sudo ip link del cradveth0 || true
sudo ip link del cradveth1 || true

sudo ip link add cradveth0 type veth peer name cradveth1
echo "1" | sudo tee "/proc/sys/net/ipv6/conf/cradveth0/forwarding"
sudo ip link set up cradveth0
sudo ip route add unreachable fd38:4ad5:6ad6::/48 dev lo
sudo ip addr add fd38:4ad5:6ad6::1/64 dev cradveth0
sudo ip link set up cradveth1
ip addr show dev cradveth0
ip addr show dev cradveth1
