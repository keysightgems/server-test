# Lab Reservation Service Controller

## Build

- **Clone this project**

  ```sh
  git clone --recursive https://bitbucket.it.keysight.com/scm/skybridge/lab-reservation-service.git
  cd lab-reservation-service/
  ```

- **For Release**

    ```sh
    # this stage will create a production/timelimited image (depending on input argument value "production"/"timelimited")
    ./ci.sh build_image production 
    # start container
    ## Parameters:
    ## --netbox-host string : NetBox hostname/ip:port (mandatory)
    ## --netbox-user-token string : NetBox User Token (mandatory)
    ## --framework-name string : Generated testbed file format (default "generic") (optional)
    ## --http-port int: HTTP Server Port (default 8080)
    ## --trs-l1s-controller string: Switch server running location with port (default "l1switchhost:l1switchport")
    ## --cleanup : Cleanup logs (and any unwanted assets) before starting service (optional)
    ## --log-level string : Log level for application - info/debug/trace (default "info") (optional)
    ## --no-stdout : Disable streaming logs to stdout
    docker run -d --net=host --name=laas-controller laas-controller:<version> --netbox-host "<hostname/ip:port>" --netbox-user-token "<user-token>"
    ```

- **For Development**

    ```sh
    # the project uses multiple build (defined in different Dockerfiles) for dev and prod environment;
    ./ci.sh build_image
    # Start container and you'll be placed inside the project dir in bash (ready to start development)
    docker run -it --net=host --name=laas-controller dev/laas-controller:latest
    ```

- **(Optional) Setup VSCode**

    After development container is ready,
    - Install Remote Explorer Extension in VSCode
    - Restart VSCode and choose `Containers` dropdown in `Remote Explorer`
    - (Optional) If your container is on a remote machine, setup a password-less SSH against it and put following line inside VSCode settings:
      ```json
      "docker.host": "ssh://username@hostname"
      ```
    - If you see the intended container listed, attach to it and change working directory to `/home/keysight/laas/controller`
    - Allow it to install extensions and other tools when prompted


## Quick Tour (for development)

**do.sh** covers most of what needs to be done manually. If you wish to extend it, just define a function (e.g. install_deps()) and call it like so: `./do.sh install_deps`.

```sh
# build and run controller in background (kill existing instances)
# new logs are generated inside `logs/`; binary is generated inside `bin/`
## Parameters:
    ## --netbox-host string : NetBox hostname/ip:port (mandatory)
    ## --netbox-user-token string : NetBox User Token (mandatory)
    ## --framework-name string : Generated testbed file format (default "generic") (optional)
    ## --http-port int: HTTP Server Port (default 8080)
    ## --trs-l1s-controller string: Switch server running location with port (default "l1switchhost:l1switchport")
    ## --cleanup : Cleanup logs (and any unwanted assets) before starting service (optional)
    ## --log-level string : Log level for application - info/debug/trace (default "info") (optional)
    ## --no-stdout : Disable streaming logs to stdout
./do.sh run --netbox-host "<hostname/ip:port>" --netbox-user-token "<user-token>"
# build controller
./do.sh build 
# kill controller running in background (if any)
./do.sh kill
# run unit / benchmark / coverage tests against all packages
./do.sh unit
# get missing go deps, generate stubs, generate certificates, run unit tests, build
# and generate artifacts inside `bin/`
./do.sh art
```

## Quick Tour (for Release)
There can be two types of release builds,
  - production: no expiry date
  - timelimited: with expiry date (as set in internal/timelimited/base.go)
      - starting with 7 days before expiry, a warning would be logged in container log when an api call is made; api would succeed
      - when expired, api call would fail and return error, same error would be logged in container log as well
      - after expiry, the validity of a container can not be extended; a new image with future expiry date need to be provided
### Steps to publish production image
```sh
  1.  # create the image
      ./ci.sh build_image production
  2.  # re-tag the image so that it does not have any internal tag e.g. production/timelimited
      ./ci.sh publish_internal production
  3.  # publish in external repository like github/ghcr.io
```
### Steps to publish time-expiry image
 ```sh
  1.  # update the new expiry date, as per need
      ## expiry date in internal/timelimited/base.go
  2.  # create the image
      ./ci.sh build_image timelimited
  3.  # re-tag the image so that it does not have any internal tag e.g. production/timelimited
      ./ci.sh publish_internal timelimited
  4.  # publish in external repository like github/ghcr.io
      ## Note: production and timelimited image should be posted in different directories of github/ghcr.io path
```
