# Current-client exchange 557 observer

Purpose: capture the exact 11-field `config_str` delivered to the current client `ExchangeItemManager::InitCurExchangeData` when S2C custom message 557 opens a shop exchange form.

Exact authority pinned by the tool:

- `fxgame.exe` SHA256 `c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3`
- `fxgamelogic.dll` SHA256 `16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8`
- `InitCurExchangeData` RVA `0x00B15560`
- entry ABI: RCX = manager object, RDX = NUL-terminated `config_str`
- exact function anchor begins `4055565741544155415641574881ec8007000048c7442458feffffff48899c24`

The observer does not call game functions or write packets, inventory, DB, or game files. **Frida `Interceptor.attach` does install a temporary trampoline in process memory while attached**, so this is observability-only rather than literally zero-memory-modification. Do not use it if the client/protection environment forbids debugger/instrumentation attachment.

## Run

1. Start the exact current client and reach the point where `fxgamelogic.dll` is loaded.
2. In PowerShell once: `./INSTALL.ps1`
3. Start: `./RUN.ps1`
4. Open one exchange form whose current ExchangeData has `BindStatus=1`.
5. Return to PowerShell and press Enter.

The `.txt` output contains only lines of the form:

`JIUYIN_EXCHANGE557_CONFIG=Type|AddValue|BindStatus|Item|ShowBind|ExchangeBind|ConditionType|Condition|Condition2|Filters|Prop`

Feed that file to `tools/stage37_exchange557_bind_probe.py --input <file>`.

No exchange purchase is required for this capture; opening the form is sufficient.
