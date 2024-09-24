#!/bin/sh

# Avoid warnings for non-interactive apt-get install
export DEBIAN_FRONTEND=noninteractive
# Variables relevant for Go builds
HOME_LOCAL=${HOME}/.local
export GOPATH=${HOME}/go
export CGO_ENABLED=0
export PATH=${PATH}:${HOME_LOCAL}/bin:${HOME_LOCAL}/go/bin:${GOPATH}/bin
export GOOS=linux

# Check supported architectures
if [ "$(arch)" = "aarch64" ] || [ "$(arch)" = "arm64" ]
then
    export GOARCH="arm64"
elif [ "$(arch)" = "x86_64" ]
then
    export GOARCH="amd64"
else
    echo "Host architecture $(arch) is not supported"
    exit 1
fi

# ********* Dependency Versions **********
GO_VERSION=1.21.4
SWAGGER_UI_VERSION=3.50.0
TESTBED_API_VERSION=0.0.3
PROTOC_VERSION=3.17.3
# ****************************************

GOCOV_OUT=false
FS_ROOT=/home/keysight/laas/controller

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S.%N')      LOG: ${@}"
}

inf() {
    echo "\n$(date '+%Y-%m-%d %H:%M:%S.%N')    INFO: ${@}"
}

wrn() {
    echo "\n$(date '+%Y-%m-%d %H:%M:%S.%N')    WARNING: ${@}\n"
}

err() {
    echo "\n$(date '+%Y-%m-%d %H:%M:%S.%N')    ERROR: ${@}\n" >&2
}

install_keysight_root_cert() {
    # install keysight root certificate to be able to download from cetain
    # locations which are otherwise restricted in keysight network
    log "Installing Keysight Root Certificate..."
    cp etc/keysight-root.crt /usr/local/share/ca-certificates/ \
    && update-ca-certificates
}

get_protoc_zip() {
    if [ "${GOARCH}" = "amd64" ]
    then
        echo "protoc-${PROTOC_VERSION}-linux-x86_64.zip"
    elif [ "${GOARCH}" = "arm64" ]
    then
        echo "protoc-${PROTOC_VERSION}-linux-aarch_64.zip"
    else
        err "Cannot get protoc zip for GOARCH=${GOARCH}"
        return 1
    fi
}

get_protoc() {
    log "Installing protoc ..."
    ret=1
    protoczip=$(get_protoc_zip) || return 1
    # install protoc per https://github.com/protocolbuffers/protobuf#protocol-compiler-installation
    mkdir -p ${HOME_LOCAL} \
    && curl -kL -o ./protoc.zip \
        https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${protoczip} \
    && unzip -o ./protoc.zip -d ${HOME_LOCAL}/ \
    && echo 'export PATH=$PATH:$HOME/.local/bin' >> ${HOME}/.profile \
    && ret=0

    rm -rf ./protoc.zip
    return ${ret}
}

get_go_tar() {
    if [ "${GOARCH}" = "amd64" ]
    then
        echo "go${GO_VERSION}.linux-amd64.tar.gz"
    elif [ "${GOARCH}" = "arm64" ]
    then
        echo "go${GO_VERSION}.linux-arm64.tar.gz"
    else
        err "Cannot get go tar for GOARCH=${GOARCH}"
        return 1
    fi
}

get_go() {
    log "Installing Go ..."
    # install golang per https://golang.org/doc/install#tarball
    gotar=$(get_go_tar) || return 1
    mkdir -p ${HOME_LOCAL} \
    && curl -kL https://dl.google.com/go/${gotar} | tar -C ${HOME_LOCAL}/ -xzf - \
    && echo 'export PATH=$PATH:$HOME/.local/go/bin:$HOME/go/bin' >> ${HOME}/.profile \
    && echo 'export GOPATH=$HOME/go' >> ${HOME}/.profile \
    && go version
}

get_swagger_ui() {
    log "Fetching swagger ui ..."
    # get swagger-ui from https://github.com/swagger-api/swagger-ui/releases
    curl -kL -o ./swagger.tar.gz \
        https://github.com/swagger-api/swagger-ui/archive/v${SWAGGER_UI_VERSION}.tar.gz \
    && rm -rf web/docs && mkdir -p web/docs \
    && tar --strip-components=2 -C web/docs -xzvf swagger.tar.gz swagger-ui-${SWAGGER_UI_VERSION}/dist/ \
    && rm -rf swagger.tar.gz \
    && rm -rf web/docs/*.map \
    && log "Getting openapi.json and patching swagger ui index.html" \
    && curl -kL -o web/docs/openapi.json https://github.com/open-traffic-generator/testbed/releases/download/v${TESTBED_API_VERSION}/openapi.json \
    && log 'Replacing "https://*.swagger.json" with "./openapi.json"' \
    && sed -i -e "s/\(\"https.\+\"\)/\".\/openapi.json\"/g" web/docs/index.html
}

install_deps() {
	# Dependencies required by this project
    apt-get update -y --no-install-recommends \
    && apt-get install -y --no-install-recommends curl git unzip ca-certificates \
    && install_keysight_root_cert \
    && get_go \
    && get_protoc \
    && get_swagger_ui
}

kill_bin() {
    for variant in development production timelimited
    do
        pkill -f bin/${variant}/controller 2>/dev/null || true
    done 
}

clean() {
    kill_bin
    rm -rf logs bin
}

get_go_deps() {
    # download all dependencies mentioned in go.mod
    log "Dowloading go mod dependencies ..."
    go mod download

    go install -v "github.com/axw/gocov/gocov@v1.0.0" \
    && go install -v "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.2" \
    && go install -v "github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.1.0" \
	&& go install -v "google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0" \
	&& go install -v "google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1" \
    && go get "gopkg.in/yaml.v3@v3.0.1"
}

get_testbed_api() {
    log "Fetching testbed api ..."
    # get testbed api artifacts and create a copy in .build directory for processing
    rm -rf api/testbed && mkdir -p api/testbed/.build \
    && curl -kL -o api/testbed/openapi.yaml https://github.com/open-traffic-generator/testbed/releases/download/v${TESTBED_API_VERSION}/openapi.yaml \
    && curl -kL -o api/testbed/opentestbed.proto https://github.com/open-traffic-generator/testbed/releases/download/v${TESTBED_API_VERSION}/opentestbed.proto \
    && cp api/testbed/*.* api/testbed/.build \
    && sed -i 's^./opentestbed;opentestbed^keysight/laas/controller/pkg/stubs/testbed/grpc/testbedgrpc;testbedgrpc^g' \
    $(find api/testbed/.build -maxdepth 1 -type f -name '*.proto')
}

gen_testbed_stubs() {
    echo "\nGenerating testbed stubs ...\n"
    rm -rf pkg/stubs/testbed && mkdir -p pkg/stubs/testbed/http && mkdir -p pkg/stubs/testbed/grpc

    # grpc and http code-gen
    protoc -Iapi/testbed/.build                         \
        --plugin=${GOPATH}/bin/protoc-gen-go-grpc       \
        --plugin=${GOPATH}/bin/protoc-gen-go            \
        --go_out=/home/                                 \
        --go-grpc_out=/home/                            \
        $(find api/testbed/.build -maxdepth 1 -type f -name 'opentestbed.proto') \
    && ${GOPATH}/bin/oapi-codegen \
        -generate "types,gorilla" \
        -package "testbedhttp" \
        api/testbed/.build/openapi.yaml > pkg/stubs/testbed/http/opentestbed_http.go
}

gen_code() {
    # get (and generate) all APIs / stubs / skeletons (Future)
    log "Generating code...\n"

    get_testbed_api \
    && gen_testbed_stubs
}

gen_certs() {
    log "Generating certs ..."
    # Thanks to: https://www.digitalocean.com/community/tutorials/openssl-essentials-working-with-ssl-certificates-private-keys-and-csrs
    TARGET_DIR=certs
    NUM_OF_DAYS=3650
    SUBJ="/C=US/ST=California/L=Santa Rosa/O=Keysight Technologies/CN=www.keysight.com"

    mkdir -p $TARGET_DIR
    rm -rf $TARGET_DIR/{server.key,server.crt,server.csr}
    # https://github.com/openssl/openssl/issues/7754#issuecomment-598808341
    touch ~/.rnd
    # create a private key
    openssl genrsa -out $TARGET_DIR/server.key 2048
    # generate certificate signing request (CSR) from existing private key
    openssl req -new -sha256 \
        -key $TARGET_DIR/server.key \
        -out $TARGET_DIR/server.csr \
        -subj "${SUBJ}"
    # generate self-signed certificate from existing CSR and private key
    openssl x509 -req -sha256 \
        -in $TARGET_DIR/server.csr \
        -signkey $TARGET_DIR/server.key \
        -out $TARGET_DIR/server.crt \
        -days $NUM_OF_DAYS
}

get_branch() {
    # in CI, branch name is usually hard to determine because a particular
    # commit is checked out instead of branch name
    # hence, we rely on GIT_CURRENT_BRANCH variable which may be set by user
    # while executing this script, otherwise we fallback to deriving it anyway
    if [ -z "${GIT_CURRENT_BRANCH}" ]
    then
        # does not work in 'detached HEAD', see comments above
        GIT_CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
        validate_branch_name ${GIT_CURRENT_BRANCH} || return 1
    fi
    echo "${GIT_CURRENT_BRANCH}"
}

validate_branch_name() {
    branch=${1}
    if [ "${branch}" != "main" ]
    then
        gbranch=$(echo ${branch} | grep -Eo "^dev-[0-9a-z-]+")
        if [ "${branch}" != "${gbranch}" ]
        then
            err "Current branch name '${branch}' is not compatible with pattern '^dev-[0-9a-z-]+'"
            return 1
        fi
    fi
}

get_git_describe() {
    # output of git describe
    # tag1-1-gb772e8b
    # ^    ^  ^
    # |    |  |
    # |    |  git hash of the commit
    # |    |
    # |   number of commits after the tag
    # |
    # |
    # Most recent tag

    # cache the output for an execution - although it's highly unlikely to change
    if [ -z "${GIT_DESCRIBE}" ]
    then
        GIT_DESCRIBE=$(git describe --tags 2>/dev/null || echo v0.0.1-1-$(git rev-parse --short HEAD))
    fi
    echo "${GIT_DESCRIBE}"
}

get_build_version() {
    # check the format in get_git_describe
    get_git_describe | cut -d- -f1 | grep -Eo "[0-9.]+"
}

get_build_revision() {
    # check the format in get_git_describe
    rev=$(get_git_describe | cut -d- -f2)

    brn=$(get_branch) || return 1
    if [ "${brn}" = "main" ]
    then 
        echo "${rev}"
    else
        echo "${brn}-${rev}"
    fi
}

get_build_commit_hash() {
    get_git_describe | cut -d- -f3 | sed -e "s/^g//g"
}

get_version() {
    if [ -z "${CURRENT_VERSION}" ]
    then
        ver=$(get_build_version) || return 1
        rev=$(get_build_revision) || return 1
        CURRENT_VERSION="${ver}-${rev}"
    fi
    echo "${CURRENT_VERSION}"
}

get_build_date() {
    echo $(date -u +"%Y-%b-%d")
}

go_fmt() {
    log "Formatting code ..."
    # exclude stubgs from being formatted
    fmtdirs="cmd/ config/ internal/"
    if [ ! -z "${CI_CONTEXT}" ]
    then
        log "CI Build is in progress, checking whether formatting changed any files ..."
        diffout=$(gofmt -d -s ${fmtdirs})
        # print what needs to be formatted
        echo "${diffout}"
        
        [ -z "${diffout}" ] || return 1
    else
        gofmt -s -w -e ${fmtdirs} || return 1
        log "Successfully formatted !"
    fi
}

go_lint() {
    log "Linting code ..."

    lintdir=$([ -z "${1}" ] && echo "./..." || echo ${1})
    golangci-lint run --build-tags timelimited --timeout 30m -v ${lintdir} --skip-dirs "tests/ api/ "
}


go_build() {
    flavour=${1}
    pkg=keysight/laas/controller/config
    ver=$(get_build_version) || return 1
    rev=$(get_build_revision) || return 1
    sha=$(get_build_commit_hash) || return 1
    date=$(get_build_date) || return 1 

    mkdir -p bin/${flavour} \
    && go build -v -tags ${flavour} \
        -ldflags "-X ${pkg}.BuildVersion=${ver} -X ${pkg}.BuildRevision=${rev} -X ${pkg}.BuildCommitHash=${sha} -X ${pkg}.BuildDate=${date} -X ${pkg}.BuildFlavour=${flavour}"\
        -o bin/${flavour}/ ./cmd/controller
}

go_mod_tidy() {
    log "Applying go mod tidy ..."
    go mod tidy
    if [ ! -z "${CI_CONTEXT}" ]
    then
        log "CI Build is in progress, checking whether go.mod/go.sum is dirty ..."
        if [ ! -z "$(git status --porcelain go.mod go.sum)" ]
        then
            err "Files go.mod/go.sum are dirty, please manually execute './do.sh build' and push changes"
            return 1
        fi
    fi
}

build() {
    go_mod_tidy || return 1
    go_fmt || return 1
    go_lint || return 1

    for variant in development production timelimited
    do
        go_build ${variant}
    done

    ver=$(get_version) || return 1

    echo ${ver} > bin/version
    date -u >> bin/version
}

run() {
    kill_bin
    # change binary name from development to production/timelimited, if that 
    # behaviour needs to be simulated in dev setup
    bin/development/controller "${@}" &
}

go_unit_test() {
    flavour=development
    if [ ! -z "${1}" ]
    then
        flavour=${1}
    fi
    log "Starting unit tests for '${flavour}' ..."
    mkdir -p bin/${flavour} \
    && rm -rf bin/${flavour}/coverage.out \
    && rm -rf bin/${flavour}/coverage.html \
    && go test -v -count 1 -p 1 -timeout 10m \
        -bench=. -cover -coverprofile bin/${flavour}/coverage.out \
        ./cmd/... ./config/... ./internal/... \
        -tags ${flavour} ${@} \
    && go tool cover -func bin/${flavour}/coverage.out | tee bin/${flavour}/unit-tests-coverage.log && go tool cover -html=bin/${flavour}/coverage.out -o bin/${flavour}/coverage.html 
}

go_unit_coverage() {
    flavour=development
    if [ ! -z "${1}" ]
    then
        flavour=${1}
    fi
    ## To get the detailed coverage output per function basis in UT
    rm -rf bin/${flavour}/gocov.out
    log "Generating GOCOV output for '${flavour}' ..."
    ${GOPATH}/bin/gocov convert bin/${flavour}/coverage.out | ${GOPATH}/bin/gocov annotate - > bin/${flavour}/gocov.out
    rm -rf bin/${flavour}/coverage.out

}


run_unit_tests() {
    kill_bin
    for flavour in production development timelimited
    do
        go_unit_test ${flavour} || return 1
        go_unit_coverage ${flavour} || return 1
    done
}

case $1 in
    deps   )
        install_deps
        ;;
    kill   )
        kill_bin
        ;;
    clean  )
        clean
        ;;
    build	)
		build
		;;
	run	)
        # pass all args (except $1) to run
        shift 1
		build && run "${@}" --no-stdout
		;;
	unit	)
        # pass all args (except $1) to run
        shift 1
        run_unit_tests ${@}
		;;
	get 	)
		get_go_deps
		;;
    gen     )
        gen_code
        ;;
	certs 	)
		gen_certs
		;;
    art		)
		get_go_deps && gen_certs && gen_code && run_unit_tests && build
		;;
    version )
        get_version
        ;;
	*		)
        $1 || echo "usage: $0 [deps|kill|clean|build|run|unit|get|gen|certs|art|version]"
		;;
esac

