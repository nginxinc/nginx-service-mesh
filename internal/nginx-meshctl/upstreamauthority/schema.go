// This file is generated. Do not edit
package upstreamauthority

var uaSchema = `{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "title": "Upstream Authority Configuration",
    "description": "Configuration for specifying upstream CAs within SPIRE",
    "type": "object",
    "properties": {
        "apiVersion": {
            "type": "string",
            "description": "Version of schema",
            "pattern": "v[0-9]+"
        },
        "upstreamAuthority": {
            "type": "string",
            "description": "Specifies which upstream authority to use",
            "enum": ["disk", "aws_pca", "awssecret", "vault", "cert-manager"]
        },
        "config": {
            "description": "Configuration of upstream authority",
            "oneOf": [
                {"$ref": "#/definitions/disk"},
                {"$ref": "#/definitions/aws_pca"},
                {"$ref": "#/definitions/awssecret"},
                {"$ref": "#/definitions/vault"},
                {"$ref": "#/definitions/cert-manager"}
            ]
        }
    },
    "required": [
        "apiVersion",
        "upstreamAuthority", 
        "config"
    ],
    "additionalProperties": false,
    "definitions": {
        "disk": {
            "type": "object",
            "description": "Upstream authority using local certificates",
            "properties": {
                "cert_file_path": {
                    "type": "string"
                },
                "key_file_path": {
                    "type": "string"
                },
                "bundle_file_path": {
                    "type": "string"
                }
            },
            "required": [
                "cert_file_path",
                "key_file_path"
            ],
            "additionalProperties": false
        },
        "aws_pca": {
            "type": "object",
            "description": "Upstream authority using Amazon Private Certificate Authority",
            "properties": {
                "region": {
                    "type": "string"
                },
                "certificate_authority_arn": {
                    "type": "string",
                    "pattern": "arn:(.*:){4}.*"
                },
                "ca_signing_template_arn": {
                    "type": "string",
                    "pattern": "arn:(.*:){4}.*"
                },
                "signing_algorithm": {
                    "type": "string"
                },
                "assume_role_arn": {
                    "type": "string",
                    "pattern": "arn:(.*:){4}.*"
                },
                "endpoint": {
                    "type": "string"
                },
                "supplemental_bundle_path": {
                    "type": "string"
                },
                "aws_access_key_id": {
                    "type": "string"
                },
                "aws_secret_access_key": {
                    "type": "string"
                }
            },
            "required": [
                "region", 
                "certificate_authority_arn"
            ],
            "additionalProperties": false
        },
        "awssecret": {
            "type": "object",
            "properties": {
                "region": {
                    "type": "string"
                },
                "cert_file_arn": {
                    "type": "string",
                    "pattern": "arn:(.*:){4}.*"
                },
                "key_file_arn": {
                    "type": "string",
                    "pattern": "arn:(.*:){4}.*"
                },
                "aws_access_key_id": {
                    "type": "string"
                },
                "aws_secret_access_key": {
                    "type": "string"
                },
                "aws_secret_token": {
                    "type": "string"
                },
                "assume_role_arn": {
                    "type": "string",
                    "pattern": "arn:(.*:){4}.*"
                }
            },
            "required": [
                "region",
                "cert_file_arn",
                "key_file_arn"
            ],
            "additionalProperties": false
        },
        "vault": {
            "type": "object",
            "properties": {
                "vault_addr": {
                    "type": "string"
                },
                "namespace": {
                    "type": "string"
                },
                "pki_mount_point": {
                    "type": "string",
                    "default": "pki"
                },
                "ca_cert_path": {
                    "type": "string"
                },
                "insecure_skip_verify": {
                    "type": "boolean",
                    "default": false
                },
                "cert_auth": {
                    "type": "object",
                    "properties": {
                        "cert_auth_mount_point": {
                            "type": "string",
                            "default": "cert"
                        },
                        "cert_auth_role_name": {
                            "type": "string"
                        },
                        "client_cert_path": {
                            "type": "string"
                        },
                        "client_key_path": {
                            "type": "string"
                        }
                    },
                    "required": [
                        "client_cert_path",
                        "client_key_path"
                    ],
                    "additionalProperties": false
                },
                "token_auth": {
                    "type": "object",
                    "properties": {
                        "token": {
                            "type": "string"
                        }
                    },
                    "required": [
                        "token"
                    ],
                    "additionalProperties": false
                },
                "approle_auth": {
                    "type": "object",
                    "properties": {
                        "approle_auth_mount_point": {
                            "type": "string",
                            "default": "approle"
                        },
                        "approle_id": {
                            "type": "string"
                        },
                        "approle_secret_id": {
                            "type": "string"
                        }
                    },
                    "required": [
                        "approle_id",
                        "approle_secret_id"
                    ],
                    "additionalProperties": false
                }
            },
            "required": [
                "vault_addr",
                "namespace",
                "ca_cert_path"
            ],
            "oneOf": [
                {"required": ["cert_auth"]},
                {"required": ["token_auth"]},
                {"required": ["approle_auth"]}
            ],
            "additionalProperties": false
        },
        "cert-manager": {
            "type": "object",
            "properties": {
                "namespace": {
                    "type": "string",
                    "minLength": 1
                },
                "issuer_name": {
                    "type": "string",
                    "minLength": 1
                },
                "issuer_kind": {
                    "type": "string",
                    "default": "Issuer"
                },
                "issuer_group": {
                    "type": "string",
                    "default": "cert-manager.io"
                },
                "kube_config_file": {
                    "type": "string"
                }
              },
              "required": ["namespace", "issuer_name"]
        }
    }
}`
