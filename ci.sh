#!/bin/sh

# Variables relevant for CI
if [ ! -z "${CI_COMMIT_REF_NAME}" ]
then
    export GIT_CURRENT_BRANCH=${CI_COMMIT_REF_NAME}
    export CI_CONTEXT=1
fi

get_version() {
    # cache the output so we don't end up fetching version everytime
    if [ -z "${CONTROLLER_VERSION}" ]
    then
        CONTROLLER_VERSION=$(./do.sh get_version) || return 1
    fi
    echo "${CONTROLLER_VERSION}"
}

build_image() {
    version=$(get_version) || return 1

    log "Building 'dev' image with version '${version}' ..."
    # Need to tag image with 'latest' so that this image can be referenced in
    # Dockerfiles for production images without knowledge of current version
    docker build --progress plain -t dev/laas-controller:latest \
        --build-arg GIT_CURRENT_BRANCH=${GIT_CURRENT_BRANCH} \
        --build-arg CI_CONTEXT=${CI_CONTEXT} \
        -f docker/Dockerfile.dev . || return 1
    docker tag dev/laas-controller:latest dev/laas-controller:${version} || return 1

    for variant in "$@"
    do
        log "Building '${variant}' image with version '${version}' ..."
        docker build --progress plain -t ${variant}/laas-controller:${version} -f docker/Dockerfile.${variant} .  || return 1
    done
}

publish_internal() {
    version=$(get_version) || return 1
    variant=${1}

    docker tag ${variant}/laas-controller:${version} laas-controller:${version}
}

help() {
    grep "() {" ${0} | cut -d\  -f1
}

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S.%N')     LOG: ${@}"
}

inf() {
    echo "\n$(date '+%Y-%m-%d %H:%M:%S.%N')    INFO: ${@}"
}

wrn() {
    echo "\n$(date '+%Y-%m-%d %H:%M:%S.%N') WARNING: ${@}\n"
}

err() {
    echo "\n$(date '+%Y-%m-%d %H:%M:%S.%N')   ERROR: ${@}\n" >&2
}

err_exit() {
    err "${1}"
    if [ ! -z ${2} ]
    then
        exit ${2}
    fi
}

# This switch-case calls a function with the same name as the first argument
# passed to this script and passes rest of the arguments to the function itself
exec_func() {
    inf "Executing '${@}'"
    # shift positional arguments so that arg 2 becomes arg 1, etc.
    cmd=${1}
    shift 1
    ${cmd} ${@} 2>&1 || err_exit "Failed executing: ${cmd} ${@}" 1
}

case $1 in
    *   )
        exec_func ${@}
    ;;
esac
