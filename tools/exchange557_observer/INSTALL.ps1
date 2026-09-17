$ErrorActionPreference = 'Stop'
python -m pip install --upgrade "frida==17.18.0"
$ver = (& python -c "import frida; print(frida.__version__)" | Select-Object -Last 1).Trim()
if ($ver -ne '17.18.0') { throw "frida version mismatch: $ver" }
Write-Host "PASS: frida $ver"
