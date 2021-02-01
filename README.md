# yubage -- a `age-plugin-yubikey` implementation in Go

```
go install eagain.net/go/yubage/cmd/age-plugin-yubikey
```

This is an [age](http://age-encryption.org/) plugin for PIV cards/Yubikey.
Your secret is kept safe on the tamperproof hardware, while letting you use
the `age` command-line.

**WARNING: UNSTABLE** Age plugins are still in flux. Consider the format
unstable, you might need to re-encrypt all your data, and this software
might accidentally delete your data, or eat your cat.


## Generating keys

At this time, this software doesn't help you generate the crypto keys.
However, this should work:

```
yubico-piv-tool --slot=82 --algorithm=ECCP256 --touch-policy=always --pin-policy=once -a generate -o MY_YUBIKEY_FILENAME.pub
yubico-piv-tool --slot=82 -a verify-pin -a selfsign-certificate --subject='/CN=MY YUBIKEY NAME HERE/O=age-plugin-yubikey/' --valid-days=3650 -i MY_YUBIKEY_FILENAME.pub -o MY_YUBIKEY_FILENAME.cert
# enter pin, touch when lights blink
yubico-piv-tool --slot=82 -a import-certificate -i MY_YUBIKEY_FILENAME.cert
```

Replace `MY_YUBIKEY_FILENAME` and `MY YUBIKEY NAME HERE` as you wish.

If you use a "management key" with your Yubikey, add the `-k` flag to first and last command (actions `generate` and `import-certificate`).

Keys are stored in the "retired slots", available starting with Yubikey series 5. Funny name, but it's 20 slots that can be used without stepping on anyone's toes.

TODO we don't at this point have code to make `age` recipient and identity strings from the above.
You can use https://github.com/str4d/age-plugin-yubikey branch [twitch](https://github.com/str4d/age-plugin-yubikey/tree/twitch), for now.

## Using

[`filippo.io/age`](https://filippo.io/age), the Go reference implementation, does not support plugins as of 2021-02-01.

[`rage`](https://github.com/str4d/rage), a Rust implementation, supports plugins in a post-v0.5.0 commit [9f824625195583c5cff0f48e5bba9b216e1fa3f6](https://github.com/str4d/rage/commit/9f824625195583c5cff0f48e5bba9b216e1fa3f6) or so.

## Background on `age` plugins & Yubikey

[AGE-PLUGIN-PROTOCOL](AGE-PLUGIN-PROTOCOL.md): My notes and links on the `age` plugin protocol.

[PIV-P256-PROTOCOL](PIV-P256-PROTOCOL.md): My notes on the PIV-P256 ECHDE encryption format used for Yubikeys with `age`.