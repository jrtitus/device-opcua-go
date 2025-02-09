#!/bin/bash
#
# Copyright (c) 2018
# Mainflux
#
# SPDX-License-Identifier: Apache-2.0
#

###
# Launches all EdgeX Go binaries (must be previously built).
#
# Expects that Consul is already installed and running.
#
###

DIR=$PWD
CMD=../cmd

function cleanup {
	pkill edgex-device-opcua
	exit 0
}

trap cleanup TERM QUIT INT

cd $CMD
exec -a edgex-device-opcua ./device-opcua -cp -d -o &
cd $DIR

while : ; do sleep 1 ; done
