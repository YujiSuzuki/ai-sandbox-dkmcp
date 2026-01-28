#!/bin/bash
set -x
docker-compose -f ./cli_sandbox/docker-compose.yml --project-directory . build
