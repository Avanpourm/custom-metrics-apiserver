#!/bin/bash
set -e

env(){
    export BUILD_DIR=./_build
    export TARGET_OS=linux
    export TARGET_ARCH=amd64
    export VERSION=`git describe --contains`
    # export PRJ=`git config --get remote.origin.url | sed 's/^.*\@//'| sed 's/\.git$//' | sed 's/:/\//' | sed 's/https\/\/\///'`
    export PRJ="github.com/kubernetes-incubator/custom-metrics-apiserver"
    export TARGET_NAME=`echo $PRJ | cut -d "/" -f3`
    export PRJ_HOST=`echo $PRJ | cut -d "/" -f1`
    export PRJ_NAME=`echo $PRJ | cut -d "/" -f2`
    export BUILD_DATE=`date -u +'%Y-%m-%dT%H:%M:%SZ'`
    export GIT_REVISION=`git rev-parse --short HEAD`
    export TEMP_DIR=""
    export GOPATH=""

    export BUILD_TAG=""
    export METHOD=""
    export BASEIMAGE="busybox"
}

set_gopath(){
    if [ ! -d "$GOPATH" ];then
        # Gen the GOPATH
        export TEMP_DIR=`mktemp -d /tmp/golang.build.XXXXXX`
        export GOPATH=$TEMP_DIR
        echo "SET GOPATH: $GOPATH"
    fi
    mkdir -p "$GOPATH/src/$PRJ_HOST/$PRJ_NAME"
    if [ ! -d "$GOPATH/src/$PRJ" ];then
        echo "$GOPATH/src/$PRJ" 
        ln -s $(pwd) "$GOPATH/src/$PRJ"
    fi
    cd "$GOPATH/src/$PRJ"
}

tail_clean(){
    if [ -d "$TEMP_DIR" ];then
        echo "in rm $TEMP_DIR"
        rm -r $TEMP_DIR
    fi
}

fmt(){
    find . -type f -name "*.go" | grep -v "./vendor*" | xargs gofmt -s -w
}

test(){
    echo "========== TEST ==========="
    echo "PWD "`pwd`
    #GOOS=$TARGET_OS  GOARCH=$TARGET_ARCH go test -short -v -cover ./... $FLAGS
    go test -short -v -cover ./... $FLAGS
}

build(){

    CGO_ENABLED=0 GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" \
        go build -v -ldflags \
        "-X $PRJ/version.ServerVersion=$VERSION -X $PRJ/version.GitCommit=$GIT_REVISION -X $PRJ/version.BuildDate=$BUILD_DATE" \
        -o $TARGET_NAME $PRJ/cmd
        
    if [ $? -ne 0 ]; then
        echo "Failed to build $TARGET_NAME"
        exit 1
    fi
    echo "Build $TARGET_NAME, OS is $TARGET_OS, Arch is $TARGET_ARCH"

    # Make dir
    mkdir -p "$BUILD_DIR/bin"
    mkdir -p "$BUILD_DIR/conf"
    mkdir -p "$BUILD_DIR/log"

    cp $TARGET_NAME "$BUILD_DIR/"
    cp deploy/Dockerfile $BUILD_DIR
    sed -i -e "s|BASEIMAGE|$BASEIMAGE|g" $BUILD_DIR/Dockerfile
    echo "Building $TARGET_NAME succeeded."

}



help()
{
    echo "Usage: ci-build.sh [-h] [-m method] [-t tag] [-p GOPATH] [-d output dir]"
    echo "       'Please run it in the scripts dir'"
    echo "       Options:"
    echo "          -m            : method"
    echo "          -h            : help info"
    echo "          -p            : set GOPATH"
    echo "          -d            : set output dir"
    echo "          -t            : build tag:  [ pre | dev | online | clean ]"
    echo ""

}

set_ops(){
    while getopts "t:m:p:d:h" arg
    do
        case $arg in
            t)
                export BUILD_TAG=$OPTARG
                ;;
            m)
                export METHOD=$OPTARG
                ;;
            p)
                export GOPATH=$OPTARG
                ;;
            d)
                export BUILD_DIR=$OPTARG
                ;;
            h)
                help;
                exit 0
                ;;
            ?)
                echo "****** [error] unknow argument"
                help
                exit 1
                ;;
        esac
    done
    if [ -z "$METHOD" ];then
        help;
        exit 1;
    fi
}

method(){
    case $METHOD in
        'all')
            build;
        ;;
        'test')
            test;
        ;;
        'clean')
            clean
        ;;
        'build')
            build;
        ;;
    esac
}

main(){
    env;
    set_ops $*; 
    set_gopath;
    fmt;
    method;
    tail_clean;
}

main $*;
exit 0;
