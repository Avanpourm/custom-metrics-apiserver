#!/bin/bash
set -x
ENV=dev

if [ -n "$1" ];then
    ENV=$1
fi

kubectl create -f ioguarder.yaml