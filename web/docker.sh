#!/bin/sh

DOCKER_TAG='2.70.3'

# Google Cloud
PROJECT_ID='personal-306017'

# Define variables for a few color codes, if NO_COLOR is unset
if [ -z "$NO_COLOR" ]; then
  # dark red
  RED='\033[0;31m'
  # light green
  GREEN='\033[1;32m'
  # light blue
  BLUE='\033[1;34m'
  # no color
  OFF='\033[0m'
fi

build() {
  # Build a docker image
  docker build . -f Dockerfile -t "roboticoverlords.org/orbiton:3000/orbiton:$DOCKER_TAG"
}

launch() {
  # Run the docker image
  docker run \
    -i \
    -t \
    --rm \
    -p 8080 \
    --name orbiton_dev \
    --network host \
    "roboticoverlords.org/orbiton:3000/orbiton:$DOCKER_TAG"
}

main() {
  # Tag the resulting docker image, then either push or run
  case $1 in
    build)
      # Just build
      build
      ;;
    buildpush)
      # Build, tag and push
      build && \
        docker tag "roboticoverlords.org/orbiton:3000/orbiton:$DOCKER_TAG" \
          "europe-north1-docker.pkg.dev/personal-306017/homepage/orbiton:$DOCKER_TAG" && \
        docker push europe-north1-docker.pkg.dev/personal-306017/homepage/orbiton:$DOCKER_TAG
        ;;
    *)
      # Build the docker image and start a container
      build && launch
  esac
}

main "$@"
