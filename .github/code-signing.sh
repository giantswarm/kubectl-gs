#!/bin/sh

APPLICATION=$1
VERSION=$2

if [[ "${CODE_SIGNING_CERT_BUNDLE_PASSWORD}" != "" ]]; then
	echo "Signing the Windows binary"
	mkdir -p certs
	echo "${CODE_SIGNING_CERT_BUNDLE_BASE64}" | base64 -d > certs/code-signing.p12
	mv ${APPLICATION}-v${VERSION}-windows-amd64.exe ${APPLICATION}-v${VERSION}-windows-amd64-unsigned.exe
	docker run --rm -ti \
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
fi
