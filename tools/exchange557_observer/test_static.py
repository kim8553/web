from pathlib import Path
import hashlib

ROOT = Path(__file__).resolve().parent
agent = (ROOT / 'agent.js').read_text(encoding='utf-8')
host = (ROOT / 'run_observer.py').read_text(encoding='utf-8')
launcher = (ROOT / 'RUN.ps1').read_text(encoding='utf-8')
readme = (ROOT / 'README.md').read_text(encoding='utf-8')

assert "0x00B15560" in agent
assert "args[1]" in agent
assert "Interceptor.attach" in agent
for forbidden in ("Interceptor.replace", "Memory.patchCode", ".writePointer(", ".writeByteArray(", "NativeFunction("):
    assert forbidden not in agent, forbidden
assert "c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3" in host
assert "16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8" in host
assert '[int]$ProcessId = 0' in launcher and '[int]$Pid = 0' not in launcher
assert 'run_{0}_{1}' in launcher  # no stale captures may be confused with a fresh run
assert 'stage37_exchange557_bind_probe.py' in launcher
assert 'BindStatus -gt 0' in launcher
assert 'cannot reveal' in readme and 'fail-closed' in readme
print('PASS static observer policy/authority/launcher checks')
print('agent_sha256', hashlib.sha256(agent.encode()).hexdigest())
