#!/usr/bin/env bash
# TODO: this script needs to be replaced with a predefined K8s enviroment

echo "Cleaning up java-tron container.."

echo "Checking for existing 'chainlink-tron.java-tron' docker container..."
dpid=`docker ps -a | grep chainlink-tron.java-tron | awk '{print $1}'`;
if [ -z "$dpid" ]
then
    echo "No docker java-tron container running.";
else
    docker kill $dpid;
    docker rm $dpid;
fi

docker network rm chainlink-tron.network

echo "Cleanup finished."
