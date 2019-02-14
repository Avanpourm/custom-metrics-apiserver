#!/bin/bash
#
# 这个文件正常不会使用到。在进行本地开发测试时伪装成 gitlab 环境

set -x

export CI_PROJECT_NAME="custom-metrics-apiserver"
export REGISTRY_DEV="172.16.200.19:5000"
export REGISTRY_PRE="harbor.meitu-int.com"
export REGISTRY_RELEAS="harbor.meitu-int.com"
export HARBOR_USER_DEV="matrix"
export HARBOR_USER_PROD="matrix"
export HARBOR_PASSWORD_DEV="XYYixDg8ZKSeeT7z"
export HARBOR_PASSWORD_PROD="WcN1CHknUaP44Gm8"
export CI_BUILD_TAG="v0.0.1-test"
