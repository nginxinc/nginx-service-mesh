#!/bin/bash

var=$(cat $1)

cat << End
// This file is generated. Do not edit
package upstreamauthority

var uaSchema = \`${var}\`
End
