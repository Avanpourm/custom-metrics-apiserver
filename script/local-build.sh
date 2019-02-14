#!/bin/bash
cd "$(dirname "$0")/.."
CUR_PATH=`pwd`

source ./script/ci-env.sh
BUILD_PATH="$CUR_PATH/_tmp"
OUTPUT_PATH="$CUR_PATH/_output"
rm -rf $BUILD_PATH
rm -rf $OUTPUT_PATH
mkdir "$BUILD_PATH" 
mkdir "$OUTPUT_PATH"
./script/ci-build.sh -p "$BUILD_PATH"  -d "$OUTPUT_PATH"  -m test

if [ $? -ne 0 ]; then
        echo "Failed test"
        exit 1
fi

./script/ci-build.sh -p "$BUILD_PATH"  -d "$OUTPUT_PATH" -m build

if [ $? -ne 0 ]; then
        echo "Failed build"
        exit 1
fi


script/ci-publish.sh "$OUTPUT_PATH"  "${CI_PROJECT_NAME}"  pre
