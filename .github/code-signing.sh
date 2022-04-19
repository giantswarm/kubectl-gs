#!/bin/sh

APPLICATION=$1
VERSION=$2

echo "APPLICATION=${APPLICATION}"
echo "VERSION=${VERSION}"
echo "USERID=${USERID}"
echo "GROUPID=${GROUPID}"
echo "PWD=${PWD}"

if [ "${CODE_SIGNING_CERT_BUNDLE_PASSWORD}" = "" ]; then
    echo "Variable CODE_SIGNING_CERT_BUNDLE_PASSWORD not set."
	exit 1
fi;

if [ "${CODE_SIGNING_CERT_BUNDLE_BASE64}" = "" ]; then
    echo "Variable CODE_SIGNING_CERT_BUNDLE_BASE64 not set."
	exit 1
fi;

echo "Signing the Windows binary"

mkdir -p certs

echo "${CODE_SIGNING_CERT_BUNDLE_BASE64}" | base64 -d > certs/code-signing.p12

mv ${APPLICATION}-v${VERSION}-windows-amd64.exe ${APPLICATION}-v${VERSION}-windows-amd64-unsigned.exe

docker run --rm \
	-v ${PWD}/certs:/mnt/certs \
	-v ${PWD}:/mnt/binaries \
	--user ${USERID}:${GROUPID} \
	quay.io/giantswarm/signcode-util:1.1.1 \
	sign \
	-pkcs12 /mnt/certs/code-signing.p12 \
	-n "Giant Swarm CLI tool ${APPLICATION}" \
	-i https://github.com/giantswarm/${APPLICATION} \
	-t http://timestamp.digicert.com -verbose \
	-in /mnt/binaries/${APPLICATION}-v${VERSION}-windows-amd64-unsigned.exe \
	-out /mnt/binaries/${APPLICATION}-v${VERSION}-windows-amd64.exe \
	-pass "${CODE_SIGNING_CERT_BUNDLE_PASSWORD}"

echo "Verifying the signed binary"

docker run --rm \
  -v ${PWD}:/mnt/binaries \
  quay.io/giantswarm/signcode-util:latest \
  verify \
  /mnt/binaries/${APPLICATION}-v${VERSION}-windows-amd64.exe
