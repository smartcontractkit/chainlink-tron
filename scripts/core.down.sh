#!/bin/bash

echo "Cleaning up core containers.."

echo "Checking for existing 'chainlink-tron.core' docker containers..."

for i in {1..4}
do
	echo " Checking for chainlink-tron.core.$i"
	dpid=$(docker ps -a | grep chainlink-tron.core.$i | awk '{print $1}')
	if [ -z "$dpid" ]; then
		echo "No docker core container running."
	else
		docker kill $dpid
		docker rm $dpid
	fi
done

echo "Cleanup finished."
