# PowerShell 5.1+ Required
# Optimized MCP Server Setup & Installer for Windows
# Run with: powershell -ExecutionPolicy Bypass -File install.ps1

$ErrorActionPreference = "Stop"

# Colors via Write-Host
function Write-Blue   { param($msg) Write-Host $msg -ForegroundColor Cyan }
function Write-Green  { param($msg) Write-Host $msg -ForegroundColor Green }
function Write-Yellow { param($msg) Write-Host $msg -ForegroundColor Yellow }
function Write-Red    { param($msg) Write-Host $msg -ForegroundColor Red }

Write-Blue "===================================================="
Write-Blue "     Optimized MCP Server Setup & Installer         "
Write-Blue "     (Windows Edition)                              "
Write-Blue "===================================================="

# ─────────────────────────────────────────────
# [1/7] OS Detection
# ─────────────────────────────────────────────
Write-Blue "`n[1/7] Detecting operating system..."
$osInfo = [System.Environment]::OSVersion
$winVer = (Get-CimInstance Win32_OperatingSystem).Caption
Write-Green "Detected: $winVer"

# Check PowerShell version
if ($PSVersionTable.PSVersion.Major -lt 5) {
    Write-Red "Error: PowerShell 5.1 or higher is required."
    exit 1
}
Write-Green "PowerShell version: $($PSVersionTable.PSVersion)"

# Check winget
$wingetAvailable = $false
try {
    $null = Get-Command winget -ErrorAction Stop
    $wingetAvailable = $true
    Write-Green "winget is available."
} catch {
    Write-Yellow "winget not found. Will try alternative Python installation."
}

# ─────────────────────────────────────────────
# [2/7] Dependency Installation
# ─────────────────────────────────────────────
Write-Blue "`n[2/7] Checking and installing dependencies..."

# --- Python ---
$pythonCmd = $null
foreach ($cmd in @("python", "python3", "py")) {
    try {
        $ver = & $cmd --version 2>&1
        if ($ver -match "Python 3") {
            $pythonCmd = $cmd
            Write-Green "Python found: $ver (command: $cmd)"
            break
        }
    } catch {}
}

if (-not $pythonCmd) {
    Write-Yellow "Python 3 not found. Installing via winget..."
    if ($wingetAvailable) {
        winget install --id Python.Python.3.12 --silent --accept-package-agreements --accept-source-agreements
        # Refresh PATH
        $env:PATH = [System.Environment]::GetEnvironmentVariable("PATH", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("PATH", "User")
        $pythonCmd = "python"
    } else {
        Write-Red "Error: Python 3 not found and winget is not available."
        Write-Red "Please install Python from https://www.python.org/downloads/ and re-run this script."
        exit 1
    }
}

# --- Git ---
try {
    $gitVer = git --version 2>&1
    Write-Green "Git found: $gitVer"
} catch {
    Write-Yellow "git not found. Installing via winget..."
    if ($wingetAvailable) {
        winget install --id Git.Git --silent --accept-package-agreements --accept-source-agreements
        $env:PATH = [System.Environment]::GetEnvironmentVariable("PATH", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("PATH", "User")
    } else {
        Write-Yellow "Warning: git not installed. Continuing without git..."
    }
}

# --- pipx ---
$pipxAvailable = $false
try {
    $null = pipx --version 2>&1
    $pipxAvailable = $true
    Write-Green "pipx is already installed."
} catch {
    Write-Yellow "pipx not found. Installing..."
    & $pythonCmd -m pip install --user pipx
    & $pythonCmd -m pipx ensurepath
    # Refresh PATH
    $userLocalBin = "$env:USERPROFILE\AppData\Local\Programs\Python\Python312\Scripts"
    $pipxBin = "$env:USERPROFILE\.local\bin"
    $env:PATH = "$env:PATH;$userLocalBin;$pipxBin"
    $pipxAvailable = $true
}

# Ensure pipx path
try {
    & $pythonCmd -m pipx ensurepath | Out-Null
} catch {}

# --- uv ---
try {
    $null = uv --version 2>&1
    Write-Green "uv is already installed."
} catch {
    Write-Yellow "uv not found. Installing via official script..."
    $uvInstallScript = "$env:TEMP\install_uv.ps1"
    Invoke-WebRequest -Uri "https://astral.sh/uv/install.ps1" -OutFile $uvInstallScript
    & powershell -ExecutionPolicy Bypass -File $uvInstallScript
    # Add uv to PATH for this session
    $uvPath = "$env:USERPROFILE\.local\bin"
    if (Test-Path $uvPath) {
        $env:PATH = "$env:PATH;$uvPath"
    }
    $cargoPath = "$env:USERPROFILE\.cargo\bin"
    if (Test-Path $cargoPath) {
        $env:PATH = "$env:PATH;$cargoPath"
    }
}

# ─────────────────────────────────────────────
# [3/7] Install CodeGraphContext & Graphify
# ─────────────────────────────────────────────
Write-Blue "`n[3/7] Installing CodeGraphContext and Graphify..."

Write-Blue "Installing/Updating codegraphcontext via pipx..."
try {
    $pipxList = & $pythonCmd -m pipx list 2>&1
    if ($pipxList -match "codegraphcontext") {
        Write-Blue "codegraphcontext already installed. Upgrading..."
        & $pythonCmd -m pipx upgrade codegraphcontext
    } else {
        & $pythonCmd -m pipx install codegraphcontext
    }
} catch {
    Write-Yellow "Warning: codegraphcontext install had issues: $_"
}

Write-Blue "Installing/Updating graphifyy[mcp] via uv..."
try {
    uv tool install --force "graphifyy[mcp]"
} catch {
    Write-Yellow "Warning: graphifyy install had issues: $_"
}

# ─────────────────────────────────────────────
# [4/7] Patch CodeGraphContext
# ─────────────────────────────────────────────
Write-Blue "`n[4/7] Applying patches to CodeGraphContext..."

$patchScript = @'
import os
import sys
import re
from pathlib import Path

# On Windows, pipx venvs are in %USERPROFILE%\.local\share\pipx\venvs
# or %LOCALAPPDATA%\pipx\pipx\venvs depending on version
possible_venv_dirs = [
    Path.home() / ".local" / "share" / "pipx" / "venvs" / "codegraphcontext",
    Path(os.environ.get("LOCALAPPDATA", "")) / "pipx" / "pipx" / "venvs" / "codegraphcontext",
    Path(os.environ.get("USERPROFILE", "")) / ".local" / "pipx" / "venvs" / "codegraphcontext",
]

venv_dir = None
for d in possible_venv_dirs:
    if d.exists():
        venv_dir = d
        break

if not venv_dir:
    # Try to find via pipx list
    import subprocess
    try:
        result = subprocess.run(["pipx", "list", "--json"], capture_output=True, text=True)
        import json
        data = json.loads(result.stdout)
        for name, info in data.get("venvs", {}).items():
            if "codegraphcontext" in name.lower():
                venv_dir = Path(info.get("metadata", {}).get("main_package", {}).get("package_or_url", ""))
                break
    except Exception:
        pass

if not venv_dir or not venv_dir.exists():
    print(f"\033[0;31mError: pipx venv for codegraphcontext not found.\033[0m")
    print("Searched paths:")
    for d in possible_venv_dirs:
        print(f"  {d}")
    sys.exit(1)

print(f"Found venv at: {venv_dir}")

# Patch A: server.py
server_py_paths = list(venv_dir.glob("**/codegraphcontext/server.py"))
if not server_py_paths:
    print("\033[0;31mError: server.py not found in venv.\033[0m")
    sys.exit(1)

server_py_path = server_py_paths[0]
print(f"Target server file: {server_py_path}")

with open(server_py_path, 'r', encoding='utf-8') as f:
    content = f.read()

patched = False

# 1. CGC_RUNTIME_DB_PATH database isolation
target = "self.db_manager = get_database_manager(db_path=ctx.db_path)"
replacement = """db_path = os.getenv("CGC_RUNTIME_DB_PATH") or ctx.db_path
            self.db_manager = get_database_manager(db_path=db_path)"""
if target in content:
    content = content.replace(target, replacement)
    patched = True

# 2. Protocol Version Negotiation
if '"protocolVersion": "2025-03-26"' in content:
    content = content.replace('"protocolVersion": "2025-03-26"', '"protocolVersion": params.get("protocolVersion", "2024-11-05")')
    patched = True

# 3. Empty instructions string
if '"instructions": LLM_SYSTEM_PROMPT' in content:
    content = content.replace('"instructions": LLM_SYSTEM_PROMPT', '"instructions": ""')
    patched = True

if patched:
    with open(server_py_path, 'w', encoding='utf-8') as f:
        f.write(content)
    print("\033[0;32mserver.py patches applied successfully!\033[0m")
else:
    print("\033[0;32mserver.py patches already applied.\033[0m")

# Patch B: Console(stderr=True)
patched_console_count = 0
for py_path in venv_dir.glob("**/codegraphcontext/**/*.py"):
    try:
        with open(py_path, 'r', encoding='utf-8') as f:
            py_content = f.read()
        if "Console()" in py_content:
            py_content = py_content.replace("Console()", "Console(stderr=True)")
            with open(py_path, 'w', encoding='utf-8') as f:
                f.write(py_content)
            patched_console_count += 1
    except Exception as e:
        print(f"Warning: Failed to patch console in {py_path}: {e}")
print(f"\033[0;32mPatched Console() in {patched_console_count} files.\033[0m")

# Patch C: database_kuzu.py read-only fallback
kuzu_py_paths = list(venv_dir.glob("**/codegraphcontext/core/database_kuzu.py"))
if kuzu_py_paths:
    kuzu_py_path = kuzu_py_paths[0]
    with open(kuzu_py_path, 'r', encoding='utf-8') as f:
        kuzu_content = f.read()

    kuzu_patched = False

    init_target = "self.db_path = new_db_path"
    init_replacement = "self.db_path = new_db_path\n        self._is_read_only = False"
    if init_target in kuzu_content and "self._is_read_only = False" not in kuzu_content:
        kuzu_content = kuzu_content.replace(init_target, init_replacement)
        kuzu_patched = True

    db_pattern = r'(\s+)self\._db = kuzu\.Database\(self\.db_path\)'
    db_replacement = (
        r'\1try:\n'
        r'\1    self._db = kuzu.Database(self.db_path)\n'
        r'\1    self._is_read_only = False\n'
        r'\1except RuntimeError as re:\n'
        r'\1    if "lock" in str(re).lower():\n'
        r'\1        info_logger("KuzuDB is locked. Opening in read_only mode.")\n'
        r'\1        self._db = kuzu.Database(self.db_path, read_only=True)\n'
        r'\1        self._is_read_only = True\n'
        r'\1    else:\n'
        r'\1        raise'
    )
    if re.search(db_pattern, kuzu_content) and "except RuntimeError as re:" not in kuzu_content:
        kuzu_content = re.sub(db_pattern, db_replacement, kuzu_content)
        kuzu_patched = True

    if kuzu_patched:
        with open(kuzu_py_path, 'w', encoding='utf-8') as f:
            f.write(kuzu_content)
        print("\033[0;32mdatabase_kuzu.py patched successfully!\033[0m")
    else:
        print("\033[0;32mdatabase_kuzu.py patches already applied.\033[0m")
else:
    print("\033[0;31mError: database_kuzu.py not found.\033[0m")
    sys.exit(1)
'@

$patchFile = "$env:TEMP\cgc_patch.py"
Set-Content -Path $patchFile -Value $patchScript -Encoding UTF8
& $pythonCmd $patchFile
if ($LASTEXITCODE -ne 0) {
    Write-Red "Error: Patch script failed!"
    exit 1
}

# ─────────────────────────────────────────────
# [5/7] Deploy Wrapper Scripts
# ─────────────────────────────────────────────
Write-Blue "`n[5/7] Deploying wrapper scripts..."

$binDir = "$env:USERPROFILE\.local\bin"
New-Item -ItemType Directory -Force -Path $binDir | Out-Null

# CodeGraphContext wrapper
Write-Blue "Writing codegraphcontext-mcp-wrapper.py..."
$cgcWrapper = @'
#!/usr/bin/env python3
import sys
import os
import json
import subprocess
import threading
from urllib.parse import urlparse, unquote
from datetime import datetime

def read_first_message(stream, log_file):
    while True:
        char = stream.read(1)
        if not char:
            return None, None, False
        if char not in (b'\r', b'\n', b' ', b'\t'):
            break

    if char == b'{':
        content = bytearray(char)
        brace_count = 1
        in_string = False
        escaped = False
        while brace_count > 0:
            next_char = stream.read(1)
            if not next_char:
                break
            content.extend(next_char)
            if in_string:
                if escaped:
                    escaped = False
                elif next_char == b'\\':
                    escaped = True
                elif next_char == b'"':
                    in_string = False
            else:
                if next_char == b'"':
                    in_string = True
                elif next_char == b'{':
                    brace_count += 1
                elif next_char == b'}':
                    brace_count -= 1
        return None, bytes(content), False
    elif char.lower() in (b'c', b'l', b'o', b'n', b't', b'e'):
        rest_of_line = stream.readline()
        first_line = char + rest_of_line
        headers = first_line
        content_length = None
        line_str = first_line.decode('ascii', errors='ignore')
        if line_str.lower().startswith('content-length:'):
            try:
                content_length = int(line_str.split(':')[1].strip())
            except ValueError:
                pass
        while True:
            line = stream.readline()
            if not line:
                break
            headers += line
            if line == b'\r\n' or line == b'\n':
                break
            line_str = line.decode('ascii', errors='ignore')
            if line_str.lower().startswith('content-length:'):
                try:
                    content_length = int(line_str.split(':')[1].strip())
                except ValueError:
                    pass
        if content_length is None:
            return headers, None, True
        content = stream.read(content_length)
        return headers, content, True
    else:
        return None, char, False

def main():
    log_dir = os.path.expanduser("~/.cache")
    os.makedirs(log_dir, exist_ok=True)
    log_file = open(os.path.join(log_dir, "codegraphcontext-mcp.log"), "a", encoding="utf-8")
    print(f"[{datetime.now().isoformat()}] Wrapper started.", file=log_file, flush=True)

    headers, content, is_framed = read_first_message(sys.stdin.buffer, log_file)

    if content:
        try:
            print(f"Intercepted: {content.decode('utf-8', errors='ignore')}", file=log_file, flush=True)
        except Exception:
            pass

    root_path = None
    if content:
        try:
            data = json.loads(content.decode('utf-8'))
            if data.get("method") == "initialize":
                params = data.get("params", {})
                root_path = params.get("rootPath")
                if not root_path and "rootUri" in params:
                    uri = params["rootUri"]
                    parsed = urlparse(uri)
                    if parsed.scheme == "file":
                        path = unquote(parsed.path)
                        # Windows: remove leading slash before drive letter
                        if path.startswith('/') and len(path) > 2 and path[2] == ':':
                            path = path[1:]
                        root_path = path
        except Exception as e:
            print(f"Error parsing initialize: {e}", file=log_file, flush=True)

    env = os.environ.copy()
    if root_path:
        db_path = os.path.join(root_path, ".codegraphcontext_db")
        env["CGC_RUNTIME_DB_PATH"] = db_path
        print(f"Workspace root: {root_path}", file=log_file, flush=True)
        print(f"DB path: {db_path}", file=log_file, flush=True)
    else:
        env["CGC_RUNTIME_DB_PATH"] = ".codegraphcontext_db"
        print("No workspace root. Using relative fallback.", file=log_file, flush=True)

    # Find codegraphcontext binary
    possible_bins = [
        os.path.expanduser("~/.local/bin/codegraphcontext.exe"),
        os.path.expanduser("~/.local/bin/codegraphcontext"),
        os.path.join(os.environ.get("LOCALAPPDATA", ""), "Programs", "Python", "Python312", "Scripts", "codegraphcontext.exe"),
    ]
    cgc_bin = None
    for b in possible_bins:
        if os.path.exists(b):
            cgc_bin = b
            break
    if not cgc_bin:
        import shutil
        cgc_bin = shutil.which("codegraphcontext")
    if not cgc_bin:
        print("ERROR: codegraphcontext not found!", file=log_file, flush=True)
        sys.exit(1)

    cmd = [cgc_bin, "mcp", "start"] + sys.argv[1:]

    try:
        proc = subprocess.Popen(
            cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=log_file,
            env=env,
            cwd=root_path
        )
    except Exception as e:
        print(f"Failed to start subprocess: {e}", file=log_file, flush=True)
        sys.exit(1)

    if is_framed:
        if headers:
            proc.stdin.write(headers)
        if content:
            proc.stdin.write(content)
    else:
        if content:
            if not content.endswith(b'\n'):
                content += b'\n'
            proc.stdin.write(content)
    proc.stdin.flush()

    def forward(source, dest):
        try:
            while True:
                data = getattr(source, "read1", source.read)(4096)
                if not data:
                    break
                dest.write(data)
                dest.flush()
        except Exception:
            pass
        finally:
            try:
                dest.close()
            except Exception:
                pass

    t1 = threading.Thread(target=forward, args=(sys.stdin.buffer, proc.stdin))
    t2 = threading.Thread(target=forward, args=(proc.stdout, sys.stdout.buffer))
    t1.start()
    t2.start()
    proc.wait()
    t1.join()
    t2.join()

if __name__ == "__main__":
    main()
'@

Set-Content -Path "$binDir\codegraphcontext-mcp-wrapper.py" -Value $cgcWrapper -Encoding UTF8

# Graphify wrapper
Write-Blue "Writing graphify-mcp-wrapper.py..."
$graphifyWrapper = @'
#!/usr/bin/env python3
import sys
import os
import json
import subprocess
import threading
from urllib.parse import urlparse, unquote
from datetime import datetime

def read_first_message(stream, log_file):
    while True:
        char = stream.read(1)
        if not char:
            return None, None, False
        if char not in (b'\r', b'\n', b' ', b'\t'):
            break

    if char == b'{':
        content = bytearray(char)
        brace_count = 1
        in_string = False
        escaped = False
        while brace_count > 0:
            next_char = stream.read(1)
            if not next_char:
                break
            content.extend(next_char)
            if in_string:
                if escaped:
                    escaped = False
                elif next_char == b'\\':
                    escaped = True
                elif next_char == b'"':
                    in_string = False
            else:
                if next_char == b'"':
                    in_string = True
                elif next_char == b'{':
                    brace_count += 1
                elif next_char == b'}':
                    brace_count -= 1
        return None, bytes(content), False
    elif char.lower() in (b'c', b'l', b'o', b'n', b't', b'e'):
        rest_of_line = stream.readline()
        first_line = char + rest_of_line
        headers = first_line
        content_length = None
        line_str = first_line.decode('ascii', errors='ignore')
        if line_str.lower().startswith('content-length:'):
            try:
                content_length = int(line_str.split(':')[1].strip())
            except ValueError:
                pass
        while True:
            line = stream.readline()
            if not line:
                break
            headers += line
            if line == b'\r\n' or line == b'\n':
                break
            line_str = line.decode('ascii', errors='ignore')
            if line_str.lower().startswith('content-length:'):
                try:
                    content_length = int(line_str.split(':')[1].strip())
                except ValueError:
                    pass
        if content_length is None:
            return headers, None, True
        content = stream.read(content_length)
        return headers, content, True
    else:
        return None, char, False

CACHE_FILE = os.path.expanduser("~/.cache/graphify-mcp-cache.json")

def load_cache():
    if os.path.exists(CACHE_FILE):
        try:
            with open(CACHE_FILE, "r") as f:
                return json.load(f)
        except Exception:
            pass
    return {}

def save_cache(cache_data):
    try:
        os.makedirs(os.path.dirname(CACHE_FILE), exist_ok=True)
        with open(CACHE_FILE, "w") as f:
            json.dump(cache_data, f, indent=2)
    except Exception:
        pass

def find_graph(root_path, log_file):
    if root_path:
        candidate = os.path.join(root_path, "graphify-out", "graph.json")
        if os.path.exists(candidate):
            cache = load_cache()
            cache[root_path] = candidate
            save_cache(cache)
            print(f"Workspace local graph found: {candidate}", file=log_file, flush=True)
            return candidate

    env_path = os.environ.get("GRAPHIFY_GRAPH_PATH")
    if env_path and os.path.exists(env_path):
        return env_path

    cache = load_cache()
    valid_cached_paths = []
    for cached_dir, cached_file in list(cache.items()):
        if os.path.exists(cached_file):
            valid_cached_paths.append(cached_file)
        else:
            cache.pop(cached_dir, None)
    save_cache(cache)

    if valid_cached_paths:
        valid_cached_paths.sort(key=lambda p: os.path.getmtime(p), reverse=True)
        return valid_cached_paths[0]

    print("Error: No graphify-out/graph.json found.", file=log_file, flush=True)
    return None

def main():
    log_dir = os.path.expanduser("~/.cache")
    os.makedirs(log_dir, exist_ok=True)
    log_file = open(os.path.join(log_dir, "graphify-mcp.log"), "a", encoding="utf-8")
    print(f"[{datetime.now().isoformat()}] Wrapper started.", file=log_file, flush=True)

    headers, content, is_framed = read_first_message(sys.stdin.buffer, log_file)

    root_path = None
    if content:
        try:
            data = json.loads(content.decode('utf-8'))
            if data.get("method") == "initialize":
                params = data.get("params", {})
                root_path = params.get("rootPath")
                if not root_path and "rootUri" in params:
                    uri = params["rootUri"]
                    parsed = urlparse(uri)
                    if parsed.scheme == "file":
                        path = unquote(parsed.path)
                        if path.startswith('/') and len(path) > 2 and path[2] == ':':
                            path = path[1:]
                        root_path = path
        except Exception as e:
            print(f"Error parsing initialize: {e}", file=log_file, flush=True)

    graph_path = find_graph(root_path, log_file)
    if not graph_path:
        print("No graph found. Server cannot start.", file=log_file, flush=True)
        sys.exit(1)

    # Find python in uv tools
    possible_pythons = [
        os.path.expanduser("~/.local/share/uv/tools/graphifyy/Scripts/python.exe"),
        os.path.expanduser("~/.local/share/uv/tools/graphifyy/bin/python"),
        os.path.join(os.environ.get("LOCALAPPDATA", ""), "uv", "tools", "graphifyy", "Scripts", "python.exe"),
    ]
    python_bin = None
    for p in possible_pythons:
        if os.path.exists(p):
            python_bin = p
            break
    if not python_bin:
        import shutil
        python_bin = shutil.which("python") or "python"

    cmd = [python_bin, "-m", "graphify.serve", graph_path] + sys.argv[1:]

    try:
        proc = subprocess.Popen(
            cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=log_file,
            cwd=root_path
        )
    except Exception as e:
        print(f"Failed to start graphify: {e}", file=log_file, flush=True)
        sys.exit(1)

    if is_framed:
        if headers:
            proc.stdin.write(headers)
        if content:
            proc.stdin.write(content)
    else:
        if content:
            if not content.endswith(b'\n'):
                content += b'\n'
            proc.stdin.write(content)
    proc.stdin.flush()

    def forward(source, dest):
        try:
            while True:
                data = getattr(source, "read1", source.read)(4096)
                if not data:
                    break
                dest.write(data)
                dest.flush()
        except Exception:
            pass
        finally:
            try:
                dest.close()
            except Exception:
                pass

    t1 = threading.Thread(target=forward, args=(sys.stdin.buffer, proc.stdin))
    t2 = threading.Thread(target=forward, args=(proc.stdout, sys.stdout.buffer))
    t1.start()
    t2.start()
    proc.wait()
    t1.join()
    t2.join()

if __name__ == "__main__":
    main()
'@

Set-Content -Path "$binDir\graphify-mcp-wrapper.py" -Value $graphifyWrapper -Encoding UTF8
Write-Green "Wrappers written to $binDir"

# ─────────────────────────────────────────────
# [6/7] Configure MCP Clients
# ─────────────────────────────────────────────
Write-Blue "`n[6/7] Updating MCP configuration files..."

$pythonBin = (Get-Command $pythonCmd).Source
$cgcWrapperPath = "$binDir\codegraphcontext-mcp-wrapper.py"
$graphifyWrapperPath = "$binDir\graphify-mcp-wrapper.py"

$mcpConfigScript = @"
import json, sys, os
from pathlib import Path

python_bin = r'$pythonBin'
cgc_wrapper = r'$cgcWrapperPath'
graphify_wrapper = r'$graphifyWrapperPath'

appdata = os.environ.get('APPDATA', '')
localappdata = os.environ.get('LOCALAPPDATA', '')
userprofile = os.environ.get('USERPROFILE', str(Path.home()))

configs = [
    Path(userprofile) / '.gemini' / 'config' / 'mcp_config.json',
    Path(appdata) / 'Claude' / 'claude_desktop_config.json',
    Path(appdata) / 'Cursor' / 'User' / 'globalStorage' / 'saoudrizwan.claude-dev' / 'settings' / 'cline_mcp_settings.json',
    Path(localappdata) / 'Programs' / 'cursor' / 'resources' / 'app' / 'extensions' / 'saoudrizwan.claude-dev' / 'settings' / 'cline_mcp_settings.json',
]

for cfg_path in configs:
    is_gemini = 'gemini' in str(cfg_path)
    if not cfg_path.exists() and not is_gemini:
        continue
    try:
        cfg_path.parent.mkdir(parents=True, exist_ok=True)
        data = {'mcpServers': {}}
        if cfg_path.exists():
            try:
                with open(cfg_path, 'r') as f:
                    data = json.load(f)
            except Exception:
                pass
        if 'mcpServers' not in data:
            data['mcpServers'] = {}
        data['mcpServers']['CodeGraphContext'] = {
            'command': python_bin,
            'args': [cgc_wrapper]
        }
        data['mcpServers']['Graphify'] = {
            'command': python_bin,
            'args': [graphify_wrapper]
        }
        with open(cfg_path, 'w') as f:
            json.dump(data, f, indent=2)
        print(f'\033[0;32mUpdated: {cfg_path}\033[0m')
    except Exception as e:
        print(f'\033[0;31mError: {cfg_path}: {e}\033[0m')
"@

$mcpConfigFile = "$env:TEMP\mcp_config.py"
Set-Content -Path $mcpConfigFile -Value $mcpConfigScript -Encoding UTF8
& $pythonCmd $mcpConfigFile

# ─────────────────────────────────────────────
# Yap Skill Installation
# ─────────────────────────────────────────────
Write-Blue "Installing 'yap' Antigravity Skill..."
$skillDir = "$env:USERPROFILE\.gemini\config\skills\yap"
New-Item -ItemType Directory -Force -Path $skillDir | Out-Null

$skillContent = @'
---
name: yap
description: Proje genelinde CodeGraphContext ve Graphify MCP sunucularını kullanarak derinlemesine analiz, optimizasyon ve kod incelemesi yapar. Kullanıcı projeyi analiz etmek istediğinde (yap, yap-ai vb.) bu beceriyi çağırır.
---

# Yap AI Performance Analyzer

Bu beceri çağrıldığında bir "Uzman Yapay Zeka Performans ve Kod Analisti" gibi davranmalısın.

## Görevlerin:
1. **Araçları Kullan**: Bağlı olan `CodeGraphContext` ve `Graphify` MCP sunucu araçlarını (tools) kullanarak projenin kod yapısını, ilişkilerini ve karmaşıklığını (complexity) analiz et.
2. **Derinlemesine Raporlama**: Kullanıcı "yap" dediğinde, projeyi baştan sona tara, önemli bileşenleri belirle ve mimari/performans iyileştirme tavsiyelerinde bulun.
3. **Proaktif Ol**: Kullanıcı sadece "projeyi başlat" veya "yap" dediğinde, ona projenin en çok dikkat gerektiren kısımlarını (en karmaşık fonksiyonlar, en çok bağlantısı olan düğümler vb.) listele.

## Kurallar:
- Daima kullanıcıyı bilgilendirmeden önce MCP araçlarından veri çek.
- Raporlarını her zaman okunaklı tablolar ve listeler halinde sun.
'@

Set-Content -Path "$skillDir\SKILL.md" -Value $skillContent -Encoding UTF8
Write-Green "✓ 'yap' skill installed to $skillDir"

# ─────────────────────────────────────────────
# [7/7] Verification
# ─────────────────────────────────────────────
Write-Blue "`n[7/7] Verifying setup..."
$verified = $true

if (Test-Path "$binDir\codegraphcontext-mcp-wrapper.py") {
    Write-Green "✓ codegraphcontext-mcp-wrapper.py exists."
} else {
    Write-Red "✗ codegraphcontext-mcp-wrapper.py missing!"
    $verified = $false
}

if (Test-Path "$binDir\graphify-mcp-wrapper.py") {
    Write-Green "✓ graphify-mcp-wrapper.py exists."
} else {
    Write-Red "✗ graphify-mcp-wrapper.py missing!"
    $verified = $false
}

if ($verified) {
    Write-Host ""
    Write-Green "======================================================"
    Write-Green "  Setup successfully completed! (Windows Edition)"
    Write-Green "======================================================"
    Write-Host "You can now run your MCP clients (Gemini CLI, Cursor, Claude Desktop)."
    Write-Host "CodeGraphContext uses workspace-isolated DB files under '.codegraphcontext_db'."
    Write-Host "Graphify server is configured to prevent startup timeouts."
    Write-Host ""
    Write-Host "NOTE: You may need to restart your terminal or MCP client for changes to take effect."
} else {
    Write-Red "Setup completed with errors. Please check the log above."
    exit 1
}
