#!/bin/sh

cd $1
REPO=$2
ENV=$3


# Project information
PROJECT="google_containers"
VERSION=${CI_BUILD_TAG}
# Use CI_COMMIT_TAG if gitlab updated to 9.0+
# VERSION=${CI_COMMIT_TAG}

case ${ENV} in
    "dev")
    IMAGE_TAG_DEV=${REGISTRY_DEV}/${PROJECT}/${REPO}:${VERSION}
    echo "Build image ${IMAGE_TAG_DEV}"
    docker build -t ${IMAGE_TAG_DEV} .

    echo "Push dev image ${IMAGE_TAG_DEV}"
    docker login -u ${HARBOR_USER_DEV} -p ${HARBOR_PASSWORD_DEV} ${REGISTRY_DEV} || exit 1
    docker push ${IMAGE_TAG_DEV} || exit 1

    echo "Delete image cache: ${IMAGE_TAG_DEV}"
    # Remove image cache to speed up builds
    # docker rmi ${IMAGE_TAG_DEV}
    ;;
    "pre")
    echo "Build image ${IMAGE_TAG_PRE}"
    IMAGE_TAG_PRE=${REGISTRY_PRE}/${PROJECT}/${REPO}:${VERSION}
    docker build -t ${IMAGE_TAG_PRE} .

    echo "Push pre image ${IMAGE_TAG_PRE}"
    docker login -u ${HARBOR_USER_PROD} -p ${HARBOR_PASSWORD_PROD} ${REGISTRY_PRE} || exit 1
    docker push ${IMAGE_TAG_PRE} || exit 1

    echo "Delete image cache: ${IMAGE_TAG_PRE}"
    # Remove image cache to speed up builds
    # docker rmi ${IMAGE_TAG_PRE}
    ;;
    "release")
    echo "Build image ${IMAGE_TAG_RELEASE}"
    IMAGE_TAG_RELEASE=${REGISTRY_RELEASE}/${PROJECT}/${REPO}:${VERSION}
    docker build -t ${IMAGE_TAG_RELEASE} .

    echo "Push release image ${IMAGE_TAG_RELEASE}"
    docker login -u ${HARBOR_USER_PROD} -p ${HARBOR_PASSWORD_PROD} ${REGISTRY_RELEASE} || exit 1
    docker push ${IMAGE_TAG_RELEASE} || exit 1

    echo "Delete image cache: ${IMAGE_TAG_RELEASE}"
    # Remove image cache to speed up builds
    # docker rmi ${IMAGE_TAG_RELEASE}
    ;;
    *)
    echo "Illegal environmental parameters"
    exit 1
    ;;
esac