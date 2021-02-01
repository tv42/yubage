# `age` Yubikey/PIV formats for PIV-P256 keys

## Recipient ("public key")

`age1yubikey1qds33lxxw9gaj82vqedjulgtedqeqxhv3tnu5f28zq3lpwpp25j4u9fu8kg`

- bech32 encoded
- compressed secp2561r1 curve point, aka public key


## Identity ("private key stub")

`AGE-PLUGIN-YUBIKEY-1QSPSYQVZ0DJFDPGWQ2RKZ`

- bech32 encoded
- data contains

  - 0x04 0x03 0x02 0x01 = serial number 0x01020304=16909060, little-endian
  - 0x82 = slot
  - `e2SWhQ` = public key "tag" to decrease collisions, first 4 bytes of SHA-256 of the recipient string

## Examples

Compute tag from recipient:

```
$ printf '%s' age1yubikey1qds33lxxw9gaj82vqedjulgtedqeqxhv3tnu5f28zq3lpwpp25j4u9fu8kg|sha256sum|cut -c-8|xxd -r -p|base64|tr -d =
e2SWhQ
```

Construct identity from contained data:

```
$ printf '\x04\x03\x02\x01\x82%s' "$(echo e2SWhQ==|base64 -d)"|bech32-encode AGE-PLUGIN-YUBIKEY-
AGE-PLUGIN-YUBIKEY-1QSPSYQVZ0DJFDPGWQ2RKZ
```

For a command-line Bech32 helper, see https://github.com/tv42/bech32

