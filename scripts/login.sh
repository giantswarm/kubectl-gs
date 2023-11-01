#!/bin/bash

# This script can be used to log i to workload clusters
# running in aws, azure or kvm installations
#
# Usage: ./login.sh <base-domain> <organization-name> <workload-cluster-name> <ttl>
#
# Mandatory: <base-domain>, <organization-name> and <workload-cluster-name>
# Optional: <ttl>

baseUrl=$1
wcOrg=$2
wcName=$3
ttl="1h"

if [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ]
then
  echo "Missing required arguments"
  echo "Usage:"
  echo "./login.sh <base-domain> <organization-name> <workload-cluster-name>"
  echo "./login.sh <base-domain> <organization-name> <workload-cluster-name> <ttl>"
  exit 1
fi

if [ -n "$4" ]
then
  ttl=$4
fi

athenaResponse=$(curl "https://athena.$baseUrl/graphql" -s \
  -H "Content-Type: application/json" \
  -d '{"operationName":"GetInfo","query":"query GetInfo { identity { provider codename } kubernetes { apiUrl authUrl caCert} }"}')

caCert=$(echo "$athenaResponse" | jq -r .data.kubernetes.caCert | base64)
apiUrl=$(echo "$athenaResponse" | jq -r .data.kubernetes.apiUrl)
dexUrl=$(echo "$athenaResponse" | jq -r .data.kubernetes.authUrl)

if [ -z "$caCert" ] || [ "$caCert" == 'null' ] || [ -z "$apiUrl" ] || [ "$apiUrl" == 'null' ] || [ -z "$dexUrl" ] || [ "$dexUrl" == 'null' ]
then
  echo "Failed to retrieve info from https://athena.$baseUrl"
  exit 1
fi

deviceCodeResponse=$(curl "$dexUrl/device/code" \
  -d client_id=zQiFLUnrTFQwrybYzeY53hWWfhOKWRAU \
  -d client_id="dex-k8s-authenticator" \
  -d scope="openid profile email offline_access groups audience:server:client_id:dex-k8s-authenticator" -s)

deviceCode=$(echo "$deviceCodeResponse" | jq -r .device_code)
verificationUrl=$(echo "$deviceCodeResponse" | jq -r .verification_uri_complete)

echo "Open this URL to log in: $verificationUrl"
echo "Press ENTER when ready to continue"
read -r continue

deviceTokenResponse=$(curl "$dexUrl/device/token" -d device_code="$deviceCode" -d grant_type="urn:ietf:params:oauth:grant-type:device_code" -s)
idToken=$(echo "$deviceTokenResponse" | jq -r .id_token)

if [ -z "$idToken" ] || [ "$idToken" == 'null' ]
then
  echo "Failed to retrieve ID token from $dexUrl"
  exit 1
fi

wcNamespace="org-$wcOrg"
certIdHash=$(date +"%s" | shasum -a 256)
certId=${certIdHash::16}

certName="$wcName-$certId"
wcBaseUrl=${baseUrl:1}

payload="{\"apiVersion\":\"core.giantswarm.io/v1alpha1\",\"kind\":\"CertConfig\",\"metadata\":{\"labels\":{\"cert-operator.giantswarm.io/version\":\"2.0.1\",\"giantswarm.io/certificate\":\"$certId\",\"giantswarm.io/cluster\":\"$wcName\",\"giantswarm.io/managed-by\":\"cluster-operator\",\"giantswarm.io/organization\":\"$wcOrg\"},\"name\":\"$certName\",\"namespace\":\"$wcNamespace\"},\"spec\":{\"cert\":{\"allowBareDomains\":true,\"clusterComponent\":\"$certId\",\"clusterID\":\"$wcName\",\"commonName\":\"$certId.$wcName.k$wcBaseUrl\",\"disableRegeneration\":false,\"organizations\":[\"system:masters\"],\"ttl\":\"$ttl\"},\"versionBundle\":{\"version\":\"2.0.1\"}}}"

certConfigResponse=$(curl -X POST "$apiUrl/apis/core.giantswarm.io/v1alpha1/namespaces/$wcNamespace/certconfigs" \
  -d "$payload" -s -H "Content-Type: application/json" \
  --header "Authorization: Bearer $idToken" \
  --insecure)

certConfigResponseKind=$(echo "$certConfigResponse" | jq -r .kind)

if [ "$certConfigResponseKind" != "CertConfig" ]
then
  echo "Failed to generate certificate for $wcName in organization $wcOrg"
  exit 1
fi

wcCa=""
wcCrt=""
wcKey=""
attempts=5

while [ $attempts -gt 0 ]
do
    sleep 3
    secretAttemptResponse=$(curl -X GET "$apiUrl/api/v1/namespaces/$wcNamespace/secrets/$certName" --header "Authorization: Bearer $idToken" --insecure -s)

    wcCa=$(echo "$secretAttemptResponse" | jq -r .data.ca)
    wcCrt=$(echo "$secretAttemptResponse" | jq -r .data.crt)
    wcKey=$(echo "$secretAttemptResponse" | jq -r .data.key)

    if [ -z "$wcCa" ] || [ "$wcCa" == 'null' ]
    then
        if [ $attempts == 5 ]
        then
          echo "Waiting for client certificate data ..."
        fi
        attempts=$(( $attempts - 1 ))
    else
        attempts=0
    fi
done

if [ "$wcCa" == 'null' ]
then
    echo "Failed to generate certificate for $wcName in organization $wcOrg"
    exit 1
fi

fileName=$(pwd)/$wcName.kubeconfig

cat > "$fileName" <<- EOF
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: $wcCa
    server: https://api.$wcName.k$wcBaseUrl:443
  name: gs-$wcName
contexts:
- context:
    cluster: gs-$wcName
    user: gs-$wcName-user
  name: gs-$wcName-clientcert
current-context: gs-$wcName-clientcert
kind: Config
preferences: {}
users:
- name: gs-$wcName-user
  user:
    client-certificate-data: $wcCrt
    client-key-data: $wcKey
EOF

echo "Generated client certificate for workload cluster '$wcName' and stored it in './$wcName.kubeconfig'"
echo "Use 'kubectl --kubeconfig ./$wcName.kubeconfig' ...' to access the cluster"
