# Current-client exchange 557 observer

This is an **optional reference-server observation tool**, not a gameplay fix or an official-server bypass. Its purpose is to record the eleven `config_str` fields passed to `ExchangeItemManager::InitCurExchangeData` when an authorized, compatible server actually sends the exchange-form response (S2C custom message 557).

Pinned binary authority:

- `fxgame.exe` SHA256 `c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3`
- `fxgamelogic.dll` SHA256 `16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8`
- `InitCurExchangeData` RVA `0x00B15560`; entry ABI RCX = manager, RDX = NUL-terminated `config_str`
- function anchor `4055565741544155415641574881ec8007000048c7442458feffffff48899c24`

**Critical prerequisite:** the connected server must already send a genuine `BindStatus>0` 557 response. The current recovery server deliberately blocks the 388 unresolved binding-sensitive shop rows. Running this observer against that server **cannot reveal their missing `ShowBind`/`ExchangeBind` values**. A form response manufactured from guessed values also cannot establish the reference behavior. No official-server access or live observation is asserted by this repository.

The observer does not call game functions or modify inventory, packets, database or game files. Frida `Interceptor.attach` **temporarily patches process code** to install a trampoline: use it only in an environment where attachment is permitted. Do not attempt to bypass client protections if attachment is denied; stop and use an authorized existing packet trace or a server-side reference implementation instead.

## Capture procedure (only after the prerequisite is met)

1. Start the exact verified current client, with `fxgamelogic.dll` loaded, connected to a compatible reference server that genuinely supports a `BindStatus>0` exchange form.
2. In PowerShell, from this folder, run `./INSTALL.ps1` once (requires Python and permission to install the pinned Frida version).
3. Run `./RUN.ps1` or `./RUN.ps1 -ProcessId <actual PID>`. Avoid a parameter named `-Pid`: `$PID` is a read-only PowerShell automatic variable.
4. Open one such exchange form **without purchasing or exchanging any items**, then return to PowerShell and press Enter.
5. The launcher allocates a fresh `exchange557_capture/run_*` directory, validates the captured eleven fields with `stage37_exchange557_bind_probe.py`, and fails if there is no capture or no `BindStatus>0` sample.

The `.txt` output contains `JIUYIN_EXCHANGE557_CONFIG=` followed by eleven pipe-delimited fields in the exact order `Type|AddValue|BindStatus|Item|ShowBind|ExchangeBind|ConditionType|Condition|Condition2|Filters|Prop`. `parsed_557.json` contains decoded fields and preview interpretations only, **not proof of the final persisted item binding**. Review the files before sharing. The `.jsonl` diagnostic includes local process/file paths and is not required for routine comparison.

If the server does not send such a response, this tool cannot make one appear. In that case leave production `BindStatus>0` rows and C2S `0x4f` fail-closed; continue searching server-side reference evidence instead of inventing `ShowBind=1` or `ExchangeBind`.
