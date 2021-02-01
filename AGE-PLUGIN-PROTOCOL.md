# The `age` plugin protocol, as seen in `rage`

[`rage`](https://github.com/str4d/rage) (post-v0.5.0 commit [9f824625195583c5cff0f48e5bba9b216e1fa3f6](https://github.com/str4d/rage/commit/9f824625195583c5cff0f48e5bba9b216e1fa3f6) or so) has a [plugin protocol](https://github.com/str4d/rage/blob/master/age-plugin/README.md) that describes how it finds plugins to run as subprocesses.

The same author has an experimental [Yubikey(/PIV) plugin](https://github.com/str4d/age-plugin-yubikey), see branch [twitch](https://github.com/str4d/age-plugin-yubikey/tree/twitch).

That documentation covers `rage`'s identity/recipient naming and plugin discovery. It, however, does not say anything about the actual communication protocol, and the code is... not gentle on fresh eyes ([lib.rs], [identity.rs], [recipient.rs]). Here's what I've been able to reverse engineer.

[lib.rs]: https://github.com/str4d/rage/blob/main/age-plugin/src/lib.rs

[identity.rs]: https://github.com/str4d/rage/blob/main/age-plugin/src/identity.rs

[recipient.rs]: https://github.com/str4d/rage/blob/main/age-plugin/src/recipient.rs

## Plugin communication

- plugin runs as subprocess of `rage`, see [age-plugin/README](https://github.com/str4d/rage/blob/master/age-plugin/README.md) for discovery
- command line flag `--age-plugin=` decides mode for plugin to run in
- unclear: what to do about unrecognized mode
- commands on `stdin`, called *stanzas*, have the form:
  ```
  -> COMMAND [ARGS..]
  BASE64_BODY
  ```
- body ends implicitly after first base64 line that doesn't contain 64 characters (partial or empty line)
- trailing empty lines are here written as `\n` for the sake of clarity
- responses have the same form
- TODO what's the convention for errors
- ignore unrecognized commands; parent sends noise to ensure this is done
- only some commands need a response (TODO?)
- not reporting unrecognized commands as errors is weird; future commands can never know if they went unrecognized, or if the plugin is slow at responding

## Mode `recipient-v1`

This mode encrypts the ephemeral file key to the given recipients.

**Phase 1**: Accumulate data. Commands may be seen multiple times in any order(TODO?).

```
-> add-recipient RECIPIENT
\n
```

Recipients that are not necessarily handled by this plugin may be seen, and should be responded to with TODO some error.

```
-> wrap-file-key
FILE_KEY
```

Body is a short 16-byte secret, to be encoded to the recipients.

The order of each kind of commands matters, they are later referred to by their 0-based index.

Phase 1 ends with

```
-> done
\n
```

**Phase 2**: Produce results. Plugin writes stanzas

```
-> recipient-stanza FILE_KEY_INDEX ARGS
WRAPPED_KEY
```

Repeated for all recognized recipients (TODO?).

Recipient stanza `ARGS` will be stored in plaintext in the age message, and available at decryption time (plugin mode identity).

or

```
-> error recipient RECIPIENT_INDEX
MESSAGE
```

or

```
-> error [ANYTHING_EXCEPT_recipient_OR_NOTHING]
MESSAGE
```

(TODO clearer meaning of errors, what scope are they reporting on.)

Phase 2 ends with plugin writing

```
-> done
\n
```

and exiting.


## Mode `identity-v1`

**Phase 1**: Accumulate data. Commands may be seen multiple times in any order(TODO?).

```
-> add-identity IDENTITY
\n
```

`IDENTITY` is a Bech32 encoded string with the type `AGE-PLUGIN-X-`, where `X` is the name of your plugin.

```
-> recipient-stanza KEY_INDEX ARGS..
WRAPPED_KEY
```

Receives arguments and body returned from mode `recipient-v1` command `wrap-file-key` with `recipient-stanza`.

Phase 1 ends with

```
-> done
\n
```

**Phase 2**: Produce results. Plugin writes stanzas

```
-> file-key KEY_INDEX
KEY
```

or

```
-> error identity INDEX
MESSAGE
```

TODO no way to indicate per-recipient, per-file error? e.g. can't get entropy.

Phase 2 can include callbacks, messages from plugin to parent and responses to those messages, see below.

Phase 2 ends with plugin writing

```
-> done
\n
```

and exiting.


## Callbacks

```
-> msg
MESSAGE_TO_USER
```

```
-> request-secret
MESSAGE_TO_USER
```

gets a response

```
-> ok
RESPONSE
```

or

```
-> error
MESSAGE
```

## TODO

`-> unsupported` if you send command not expected in that phase
