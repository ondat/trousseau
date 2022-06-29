# Local development

This document describes how to develop Trousseau AWS KMS provider on your local machine.

Please follow base documentation at [localdev.md](/../../localdev.md)

## Login to AWS

Log in and create profile file at `~/.aws/credentials`.

Edit config file at [awskms.yaml](/../../tests/e2e/kuttl/kube-v1.23/awskms.yaml):

```yaml
profile: profile
keyArn: keyArn
# Optional fields
roleArn: roleArn
mfaToken: token
encryptionContext:
  foo: bar
```

### Using Localstack

Aternatively you should spin up a Localstack on your machine to test AWS KMS without using AWS itself.

Edit profile file at `~/.aws/credentials`:

```ini
[trousseau-local-aws]
aws_access_key_id=000000000000
aws_secret_access_key=XXX
```

Start services:

```bash
docker run --name trousseau-local-aws --rm -d -e SERVICES=kms -e HOSTNAME=localhost.localstack.cloud -e HOSTNAME_EXTERNAL=localhost.localstack.cloud -e DEFAULT_REGION=eu-west-1 -e KMS_PROVIDER=kms-local -p 4566:4566 -p 4510-4559:4510-4559 localstack/localstack:0.14.4
docker exec trousseau-local-aws awslocal kms create-key
```

Output:
```json
{
    "KeyMetadata": {
        "AWSAccountId": "000000000000",
        "KeyId": "c720e1b5-a113-44ed-9f7b-1cf0c1f61ee8",
        "Arn": "arn:aws:kms:eu-west-1:000000000000:key/c720e1b5-a113-44ed-9f7b-1cf0c1f61ee8",
        "CreationDate": 1656405914,
        "Enabled": true,
        "KeyUsage": "ENCRYPT_DECRYPT",
        "KeyState": "Enabled",
        "Origin": "AWS_KMS",
        "KeyManager": "CUSTOMER",
        "CustomerMasterKeySpec": "SYMMETRIC_DEFAULT",
        "KeySpec": "SYMMETRIC_DEFAULT",
        "EncryptionAlgorithms": [
            "SYMMETRIC_DEFAULT"
        ]
    }
}
```

Edit config file based on output at [awskms.yaml](/../../tests/e2e/kuttl/kube-v1.23/awskms.yaml):

```yaml
endpoint: https://localhost.localstack.cloud:4566
profile: trousseau-local-aws
keyArn: arn:aws:kms:eu-west-1:000000000000:key/c720e1b5-a113-44ed-9f7b-1cf0c1f61ee8
```

## Run Trousseau components

Use command line or our favorite IDE to start Trousseau components on your machine:

```bash
task go:run:proxy
task go:run:awskms
ENABLED_PROVIDERS="--enabled-providers awskms" task go:run:trousseau
```
