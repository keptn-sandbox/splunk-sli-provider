#!/bin/bash

# This script builds and pushes a new docker image of the splunk service and releases a new helm chart of the splunk service.
# It takes the following arguments:
# -d: the path to the splunk service directory
# -u: the docker registry username
# -v: the version of the service to build and push
# -l: show the logs of the splunk service pod
# -b: build and push or not the docker image of the service
# -t: archive the helm chart of the service and use it to release the service
# Example: ./build-splunk-sli-provider.sh -d ../. -u dockerUsername -v 0.1.0 -l

function show_usage {
	echo "Usage: $(basename "$0") -d <directory> -u <dockerUsername> -v <version>"
	echo "Default values:"
	echo "  - directory: ../."
	echo "  - version: latest"

	exit 1
}

function checking_pod_termination() {
	local pod_name=""

	while true; do
		# Get the current output from the kubectl command
		pod_name=$(kubectl -n keptn get pods | grep splunk-sli-provider | awk '{print $1}')

		# Compare the current output with the previous output
		if [[ -z $pod_name ]]; then
			break
		fi

		# Sleep for 3 seconds before the next iteration
		sleep 3
	done
}

function checking_pod_running() {
	local pod_state=""

	while true; do
		# Get the current output from the kubectl command
		pod_state=$(kubectl -n keptn get pods | grep splunk-sli-provider | awk '{print $3}')

		# Compare the current output with the previous output
		if [[ $pod_state == "Running" ]]; then
			break
		fi

		# Sleep for 3 seconds before the next iteration
		sleep 3
	done
}

function get_splunk_pod() {
	echo $(kubectl -n keptn get pods | grep splunk | awk '{print $1}')
}

# Initialize variables to store option values
servicePath=""
dockerUsername=""
serviceVersion=""
show_logs=false
build_and_push=false
archive_chart=false

# Parse script parameters
while getopts ":d:u:v:l:b" opt; do
	case "$opt" in
	d)
		servicePath="$OPTARG"
		;;
	u)
		dockerUsername="$OPTARG"
		;;
	v)
		serviceVersion="$OPTARG"
		;;
	l)
		show_logs=true
		;;
	b)
		build_and_push=true
		;;
	t)
		archive_chart=true
		;;
	\?)
		echo "Invalid option: -$OPTARG" >&2
		show_usage
		;;
	:)
		echo "Option -$OPTARG requires an argument." >&2
		show_usage
		;;
	esac
done

# check if the splunk service directory is specified
if [[ -z $servicePath ]]; then
	echo -e "The path to the splunk service directory is not specified. Using default path: ../.\n $servicePath"
	servicePath="../."
fi

# check if the docker registry is specified
if [[ -z $serviceVersion ]]; then
	echo -e "The version of the service to build and push is not specified. Using default version: latest \n"
	serviceVersion="latest"
fi

if $build_and_push; then
	# check if the docker registry is specified
	if [[ -z $dockerUsername ]]; then
		echo -e "The docker registry username is not specified.\n"
		show_usage
	fi
	docker build . -t ghcr.io/$dockerUsername/splunk-sli-provider:$serviceVersion --network=host && docker push ghcr.io/$dockerUsername/splunk-sli-provider:$serviceVersion
fi

cd $servicePath # go to the splunk service directory

# build and push a new docker image of the service

# remove an existing helm chart of the splunk service
# if [[ $(get_splunk_pod) ]]; then
# 	helm uninstall -n keptn splunk-sli-provider
# 	echo "Waiting for the previous splunk pods to be terminated"
# 	checking_pod_termination
# fi

# release the new chart
chartName=splunk-sli-provider-$serviceVersion.tgz
if $archive_chart; then
	tar -czvf $chartName chart/
	helm upgrade --install -n keptn splunk-sli-provider $chartName --set splunkservice.existingSecret=splunk-sli-provider-secret --set splunkservice.image.tag=$serviceVersion
else
	helm upgrade --install -n keptn splunk-sli-provider https://github.com/keptn-sandbox/splunk-sli-provider/releases/download/$serviceVersion/$chartName --set splunkservice.existingSecret=splunk-service-secret --set splunkservice.image.tag=$serviceVersion
fi

if $show_logs; then
	# show the logs of the splunk service pod
	echo "Waiting for the splunk service pod to be running"
	checking_pod_running
	kubectl -n keptn logs -f -c splunk-sli-provider $(get_splunk_pod)
fi
