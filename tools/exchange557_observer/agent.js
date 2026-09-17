'use strict';

// Current-client Age of Wushu shop-exchange observer.
// Observation only: no NativeFunction calls, no packet writes, no state writes.
// Frida Interceptor.attach itself installs a trampoline while attached.

const MODULE_NAME = 'fxgamelogic.dll';
const INIT_EXCHANGE_RVA = 0x00B15560;
const EXPECTED_IMAGE_SIZE = 0x0322E000;
const EXPECTED_ANCHOR_HEX = '4055565741544155415641574881ec8007000048c7442458feffffff48899c24';
const SCHEMA = 'nineyin-current-exchange557-observer-v1';

function bytesToHex(buf) {
  const a = new Uint8Array(buf);
  let out = '';
  for (let i = 0; i < a.length; i++) out += a[i].toString(16).padStart(2, '0');
  return out;
}

function emit(name, fields) {
  send({
    kind: 'event',
    payload: Object.assign({
      schema: SCHEMA,
      event: name,
      pid: Process.id,
      thread_id: Process.getCurrentThreadId(),
      target_unix_ms: Date.now()
    }, fields || {})
  });
}

if (Process.platform !== 'windows' || Process.arch !== 'x64' || Process.pointerSize !== 8) {
  throw new Error('unsupported runtime: ' + Process.platform + '/' + Process.arch + '/ptr' + Process.pointerSize);
}

const mod = Process.getModuleByName(MODULE_NAME);
if (mod.size !== EXPECTED_IMAGE_SIZE) {
  throw new Error('FxGameLogic image-size mismatch: got=0x' + mod.size.toString(16) + ' expected=0x' + EXPECTED_IMAGE_SIZE.toString(16));
}

const target = mod.base.add(INIT_EXCHANGE_RVA);
const anchor = bytesToHex(target.readByteArray(EXPECTED_ANCHOR_HEX.length / 2));
if (anchor !== EXPECTED_ANCHOR_HEX) {
  throw new Error('InitCurExchangeData anchor mismatch at ' + target + ': got=' + anchor + ' expected=' + EXPECTED_ANCHOR_HEX);
}

let eventSeq = 0;
let listener = null;

try {
  listener = Interceptor.attach(target, {
    onEnter(args) {
      // Windows x64 ABI at this exact current function: RCX=this, RDX=config_str.
      // Frida maps those to args[0] and args[1].
      try {
        const configPtr = args[1];
        if (configPtr.isNull()) {
          emit('EXCHANGE557_ANOMALY', { event_seq: ++eventSeq, error: 'null config_str pointer' });
          return;
        }
        const raw = configPtr.readUtf8String();
        if (raw === null) {
          emit('EXCHANGE557_ANOMALY', { event_seq: ++eventSeq, error: 'readUtf8String returned null' });
          return;
        }
        if (raw.length > 32768) {
          emit('EXCHANGE557_ANOMALY', { event_seq: ++eventSeq, error: 'config_str exceeds 32768 characters' });
          return;
        }
        const fields = raw.split('|');
        if (fields.length !== 11) {
          emit('EXCHANGE557_ANOMALY', {
            event_seq: ++eventSeq,
            error: 'expected exactly 11 pipe fields',
            field_count: fields.length,
            raw: raw
          });
          return;
        }
        emit('EXCHANGE557_CONFIG', {
          event_seq: ++eventSeq,
          module: MODULE_NAME,
          module_base: mod.base.toString(),
          function_rva: '0x00B15560',
          raw: raw,
          Type: fields[0],
          AddValue: fields[1],
          BindStatus: fields[2],
          Item: fields[3],
          ShowBind: fields[4],
          ExchangeBind: fields[5],
          ConditionType: fields[6],
          Condition: fields[7],
          Condition2: fields[8],
          Filters: fields[9],
          Prop: fields[10]
        });
      } catch (e) {
        emit('EXCHANGE557_ANOMALY', { event_seq: ++eventSeq, error: String(e) });
      }
    }
  });
  emit('OBSERVER_READY', {
    event_seq: ++eventSeq,
    module: MODULE_NAME,
    module_path: mod.path,
    module_base: mod.base.toString(),
    module_size: mod.size,
    function_rva: '0x00B15560',
    function_address: target.toString(),
    anchor_hex: anchor,
    observer_only: true
  });
} catch (e) {
  if (listener !== null) {
    try { listener.detach(); } catch (_) {}
  }
  throw e;
}
