#!/bin/bash

# This script builds and pushes a new docker image of the splunk service and releases a new helm chart of the splunk service.
# It takes the following arguments:
# -d: the path to the splunk service directory
# -u: the docker registry username
# -v: the version of the service to build and push
# -l: show the logs of the splunk service pod
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

# Parse script parameters
while getopts ":d:u:v:l" opt; do
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

# Check if required options are provided
if [[ -z "$dockerUsername" ]]; then
	echo -e "Missing required arguments.\n"
	show_usage
fi

cd $servicePath # go to the splunk service directory

# build and push a new docker image of the service
docker build . -t $dockerUsername/splunk-sli-provider:$serviceVersion --network=host && docker push $dockerUsername/splunk-sli-provider:$serviceVersion

# remove an existing helm chart of the splunk service
# if [[ $(get_splunk_pod) ]]; then
# 	helm uninstall -n keptn splunk-sli-provider
# 	echo "Waiting for the previous splunk pods to be terminated"
# 	checking_pod_termination
# fi

# release the new chart
chartName=splunk-sli-provider.tgz
tar -czvf $chartName chart/
helm upgrade --install -n keptn splunk-sli-provider $chartName --set splunkservice.existingSecret=splunk-sli-provider-secret --set splunkservice.image.tag=$serviceVersion

if $show_logs; then
	# show the logs of the splunk service pod
	echo "Waiting for the splunk service pod to be running"
	checking_pod_running
	kubectl -n keptn logs -f -c splunk-sli-provider $(get_splunk_pod)
fi
