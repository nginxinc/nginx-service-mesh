#! /bin/bash

export BUILD_ROOT="/opt/build/repo"

directories=("${BUILD_ROOT}/api/" "${BUILD_ROOT}/examples/" "${BUILD_ROOT}/helm-chart/crds/")
for dir in ${directories[@]}; do
    cp -Rv $dir ${BUILD_ROOT}/docs/content/
done
