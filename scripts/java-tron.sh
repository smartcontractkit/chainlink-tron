#!/usr/bin/env bash
#
# see https://github.com/tronprotocol/java-tron/blob/develop/docker for reference

dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

container_name="chainlink-tron.java-tron"
container_version="GreatVoyage-v4.7.3.1"

if [ $# -ne 1 ]; then
  genesis_address="TDRVFH1KLFhAmYvrXdk1hbuNQqgkVtdBX5"
else
  genesis_address="$1"
fi

set -e pipefail

bash "${dir}/java-tron.down.sh"

listen_ips=""
if [ "$(uname)" = "Darwin" ]; then
	echo "Listening on all interfaces on MacOS"
	listen_ips="0.0.0.0"
else
	docker_ip=$(docker network inspect bridge -f '{{range .IPAM.Config}}{{.Gateway}}{{end}}')
	if [ -z "${docker_ip}" ]; then
		echo "Could not fetch docker ip."
		exit 1
	fi
	listen_ips="127.0.0.1 ${docker_ip}"
fi

echo "Starting java-tron container"
echo "Genesis test account address: ${genesis_address}"

temp_dir=$(mktemp -d)
temp_conf="${temp_dir}/java-tron.conf"
sed "s/#genesis_address#/${genesis_address}/g" "${dir}/java-tron.conf" > "${temp_conf}"
echo "Created temp config: ${temp_conf}"

listen_args=()
for ip in $listen_ips; do
	listen_args+=("-p" "${ip}:16666:16666")
	listen_args+=("-p" "${ip}:16667:16667")
	listen_args+=("-p" "${ip}:16668:16668")
	listen_args+=("-p" "${ip}:16669:16669")
done

docker run \
	"${listen_args[@]}" \
  -d \
	--platform linux/amd64 \
	--name "${container_name}" \
  --mount "type=bind,source=${temp_conf},target=/java-tron.conf" \
  --entrypoint bash \
	"tronprotocol/java-tron:${container_version}" \
  "-c" \
	"./bin/FullNode -c /java-tron.conf & mkdir -p logs && touch ./logs/tron.log && tail -F ./logs/tron.log" \

echo "Waiting for tron container to become ready.."
start_time=$(date +%s)
prev_output=""
while true; do
	output=$(docker logs "${container_name}" 2>&1)
	if [[ "${output}" != "${prev_output}" ]]; then
		echo -n "${output#$prev_output}"
		prev_output="${output}"
	fi

	if [[ $output == *"All api services started."* ]]; then
		echo ""
		echo "java-tron is ready."
		exit 0
	fi

	current_time=$(date +%s)
	elapsed_time=$((current_time - start_time))

	if ((elapsed_time > 600)); then
		echo "Error: Command did not become ready within 600 seconds"
		exit 1
	fi

	sleep 3
done
