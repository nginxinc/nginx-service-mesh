#!/bin/bash

ID=custom%2B5618%2Fgithub.com%2Fnginxinc%2Fnginx-service-mesh

get_revision() {
    curl -s "https://app.fossa.com/api/revisions?projectId=${ID}" -H "Authorization: Bearer ${FOSSA_TOKEN}" | jq -ec ".[] | select(.locator | contains(\"${COMMIT_SHA}\"))"
}

echo -n "waiting for revision ${COMMIT_SHA} to exist..."
until get_revision > /dev/null; do
    sleep 10
done
echo "done"

REV_ID="${ID}%24${COMMIT_SHA}"

get_attributions() {
    curl -s "https://app.fossa.com/api/revisions/${REV_ID}/attribution/full/SPDX_JSON" -H "Authorization: Bearer ${FOSSA_TOKEN}"
}

echo -n "waiting for attributions to be populated..."
while 
OUTPUT=$(get_attributions)
LEN=$(jq '.packages | length' <<< "$OUTPUT")
[[ $LEN -le 1 ]]
do
  sleep 10
done
echo "done"

echo $OUTPUT | jq > nsm.sbom.json
echo "SBOM report generated"
