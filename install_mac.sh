#!/usr/bin/env bash

# Exit immediately if a command exits with a non-zero status
set -e

# Setup colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
BOLD='\033[1m'

echo -e "${BLUE}${BOLD}====================================================${NC}"
echo -e "${BLUE}${BOLD}     Optimized MCP Server Setup & Installer         ${NC}"
echo -e "${BLUE}${BOLD}     (macOS Edition)                                ${NC}"
echo -e "${BLUE}${BOLD}====================================================${NC}"

# ─────────────────────────────────────────────
# [1/7] OS Detection
# ─────────────────────────────────────────────
echo -e "\n${BLUE}[1/7] Detecting operating system...${NC}"

if [[ "$(uname)" != "Darwin" ]]; then
    echo -e "${RED}Error: This script is for macOS only.${NC}"
    echo -e "For Linux, use: bash install.sh"
    echo -e "For Windows, use: powershell -ExecutionPolicy Bypass -File install.ps1"
    exit 1
fi

MAC_VER=$(sw_vers -productVersion)
MAC_NAME=$(sw_vers -productName)
ARCH=$(uname -m)
echo -e "Detected: ${GREEN}$MAC_NAME $MAC_VER ($ARCH)${NC}"

# macOS version check (require 12+)
MAC_MAJOR=$(echo "$MAC_VER" | cut -d. -f1)
if [ "$MAC_MAJOR" -lt 12 ]; then
    echo -e "${YELLOW}Warning: macOS 12 (Monterey) or newer is recommended.${NC}"
fi

# ─────────────────────────────────────────────
# [2/7] Dependency Installation
# ─────────────────────────────────────────────
echo -e "\n${BLUE}[2/7] Checking and installing dependencies...${NC}"

# --- Xcode Command Line Tools ---
if ! xcode-select -p &>/dev/null; then
    echo -e "${YELLOW}Xcode Command Line Tools not found. Installing...${NC}"
    xcode-select --install
    echo -e "${YELLOW}Please complete the Xcode CLT installation dialog, then re-run this script.${NC}"
    exit 0
fi
echo -e "${GREEN}Xcode Command Line Tools: OK${NC}"

# --- Homebrew ---
if ! command -v brew &>/dev/null; then
    echo -e "${YELLOW}Homebrew not found. Installing...${NC}"
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    # Add brew to PATH for Apple Silicon
    if [[ "$ARCH" == "arm64" ]]; then
        eval "$(/opt/homebrew/bin/brew shellenv)"
        echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> "$HOME/.zprofile"
    else
        eval "$(/usr/local/bin/brew shellenv)"
    fi
else
    echo -e "${GREEN}Homebrew is already installed.${NC}"
fi

# Ensure brew is on PATH
if command -v brew &>/dev/null; then
    eval "$(brew shellenv)"
fi

# --- Python 3 ---
if ! command -v python3 &>/dev/null; then
    echo -e "${YELLOW}Python 3 not found. Installing via Homebrew...${NC}"
    brew install python
else
    PYTHON_VER=$(python3 --version)
    echo -e "${GREEN}Python found: $PYTHON_VER${NC}"
fi

# --- Git ---
if ! command -v git &>/dev/null; then
    echo -e "${YELLOW}git not found. Installing via Homebrew...${NC}"
    brew install git
else
    echo -e "${GREEN}git: $(git --version)${NC}"
fi

# --- pipx ---
if ! command -v pipx &>/dev/null; then
    echo -e "${YELLOW}pipx not found. Installing via Homebrew...${NC}"
    brew install pipx
    pipx ensurepath
    # Source updated PATH
    export PATH="$HOME/.local/bin:$PATH"
else
    echo -e "${GREEN}pipx is already installed.${NC}"
fi

# Ensure pipx paths are configured
pipx ensurepath --force 2>/dev/null || true

# --- uv ---
if ! command -v uv &>/dev/null; then
    echo -e "${YELLOW}uv not found. Installing via official script...${NC}"
    curl -LsSf https://astral.sh/uv/install.sh | sh
    # Source env
    if [ -f "$HOME/.local/bin/env" ]; then
        source "$HOME/.local/bin/env"
    elif [ -f "$HOME/.cargo/env" ]; then
        source "$HOME/.cargo/env"
    fi
    export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$PATH"
else
    echo -e "${GREEN}uv: $(uv --version)${NC}"
fi

# ─────────────────────────────────────────────
# [3/7] Install CodeGraphContext & Graphify
# ─────────────────────────────────────────────
echo -e "\n${BLUE}[3/7] Installing CodeGraphContext and Graphify...${NC}"

echo -e "Installing/Updating ${BOLD}codegraphcontext${NC} via pipx..."
if pipx list | grep -q "codegraphcontext"; then
    echo -e "codegraphcontext already installed. Upgrading..."
    pipx upgrade codegraphcontext || true
else
    pipx install codegraphcontext || true
fi

echo -e "Installing/Updating ${BOLD}graphifyy[mcp]${NC} via uv..."
uv tool install --force "graphifyy[mcp]"

# ─────────────────────────────────────────────
# [4/7] Patch CodeGraphContext
# ─────────────────────────────────────────────
echo -e "\n${BLUE}[4/7] Applying patches to CodeGraphContext...${NC}"
python3 - << 'EOF'
import os
import sys
import re
from pathlib import Path

venv_dir = Path.home() / ".local/share/pipx/venvs/codegraphcontext"
if not venv_dir.exists():
    print(f"\033[0;31mError: pipx venv for codegraphcontext not found at {venv_dir}\033[0m")
    sys.exit(1)

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

# 1. CGC_RUNTIME_DB_PATH
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
        print(f"Warning: Failed to patch {py_path}: {e}")
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
EOF

# ─────────────────────────────────────────────
# [5/7] Deploy Wrapper Scripts
# ─────────────────────────────────────────────
echo -e "\n${BLUE}[5/7] Deploying wrapper scripts...${NC}"
mkdir -p "$HOME/.local/bin"

# CodeGraphContext wrapper
echo -e "Writing codegraphcontext-mcp-wrapper..."
cat << 'EOF' > "$HOME/.local/bin/codegraphcontext-mcp-wrapper"
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

    for k, v in sorted(os.environ.items()):
        if any(s in k.lower() for s in ["key", "secret", "password", "token"]):
            print(f"  {k}=<masked>", file=log_file, flush=True)
        else:
            print(f"  {k}={v}", file=log_file, flush=True)

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
                        root_path = unquote(parsed.path)
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

    cmd = [os.path.expanduser("~/.local/bin/codegraphcontext"), "mcp", "start"] + sys.argv[1:]

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
EOF
chmod +x "$HOME/.local/bin/codegraphcontext-mcp-wrapper"

# Graphify wrapper
echo -e "Writing graphify-mcp-wrapper..."
cat << 'EOF' > "$HOME/.local/bin/graphify-mcp-wrapper"
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
            print(f"Workspace local graph: {candidate}", file=log_file, flush=True)
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
        fallback = valid_cached_paths[0]
        print(f"Using cached graph: {fallback}", file=log_file, flush=True)
        return fallback

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
                        root_path = unquote(parsed.path)
        except Exception as e:
            print(f"Error parsing initialize: {e}", file=log_file, flush=True)

    graph_path = find_graph(root_path, log_file)
    if not graph_path:
        print("No graph found. Server cannot start.", file=log_file, flush=True)
        sys.exit(1)

    python_bin = os.path.expanduser("~/.local/share/uv/tools/graphifyy/bin/python")
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
EOF
chmod +x "$HOME/.local/bin/graphify-mcp-wrapper"

echo -e "${GREEN}Wrappers written to $HOME/.local/bin/${NC}"

# ─────────────────────────────────────────────
# [6/7] Configure MCP Clients
# ─────────────────────────────────────────────
echo -e "\n${BLUE}[6/7] Updating MCP configuration files...${NC}"
python3 - << 'EOF'
import json
import sys
from pathlib import Path

home = Path.home()

configs = [
    home / ".gemini" / "config" / "mcp_config.json",
    # Claude Desktop on macOS
    home / "Library" / "Application Support" / "Claude" / "claude_desktop_config.json",
    # Cursor (Cline extension)
    home / "Library" / "Application Support" / "Cursor" / "User" / "globalStorage" / "saoudrizwan.claude-dev" / "settings" / "cline_mcp_settings.json",
    # VS Code (Cline extension)
    home / "Library" / "Application Support" / "Code" / "User" / "globalStorage" / "saoudrizwan.claude-dev" / "settings" / "cline_mcp_settings.json",
]

for cfg_path in configs:
    is_gemini = "gemini" in str(cfg_path)
    if not cfg_path.exists() and not is_gemini:
        continue
    try:
        cfg_path.parent.mkdir(parents=True, exist_ok=True)
        data = {"mcpServers": {}}
        if cfg_path.exists():
            try:
                with open(cfg_path, 'r') as f:
                    data = json.load(f)
            except Exception:
                pass
        if "mcpServers" not in data:
            data["mcpServers"] = {}
        data["mcpServers"]["CodeGraphContext"] = {
            "command": f"{Path.home()}/.local/bin/codegraphcontext-mcp-wrapper",
            "args": []
        }
        data["mcpServers"]["Graphify"] = {
            "command": f"{Path.home()}/.local/bin/graphify-mcp-wrapper",
            "args": []
        }
        with open(cfg_path, 'w') as f:
            json.dump(data, f, indent=2)
        print(f"\033[0;32mUpdated: {cfg_path}\033[0m")
    except Exception as e:
        print(f"\033[0;31mError: {cfg_path}: {e}\033[0m")
EOF

# ─────────────────────────────────────────────
# Yap Skill Installation
# ─────────────────────────────────────────────
echo -e "\n${BLUE}Installing 'yap' Antigravity Skill...${NC}"
SKILL_DIR="$HOME/.gemini/config/skills/yap"
mkdir -p "$SKILL_DIR"
cat << 'SKILL_EOF' > "$SKILL_DIR/SKILL.md"
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
SKILL_EOF

echo -e "${GREEN}✓ 'yap' skill installed to $SKILL_DIR${NC}"

# ─────────────────────────────────────────────
# [7/7] Verification
# ─────────────────────────────────────────────
echo -e "\n${BLUE}[7/7] Verifying setup...${NC}"
VERIFIED=true

if [ -f "$HOME/.local/bin/codegraphcontext-mcp-wrapper" ]; then
    echo -e "${GREEN}✓ codegraphcontext-mcp-wrapper exists.${NC}"
else
    echo -e "${RED}✗ codegraphcontext-mcp-wrapper missing!${NC}"
    VERIFIED=false
fi

if [ -f "$HOME/.local/bin/graphify-mcp-wrapper" ]; then
    echo -e "${GREEN}✓ graphify-mcp-wrapper exists.${NC}"
else
    echo -e "${RED}✗ graphify-mcp-wrapper missing!${NC}"
    VERIFIED=false
fi

if [ "$VERIFIED" = true ]; then
    echo -e "\n${GREEN}${BOLD}======================================================${NC}"
    echo -e "${GREEN}${BOLD}  Setup successfully completed! (macOS Edition)${NC}"
    echo -e "${GREEN}${BOLD}======================================================${NC}"
    echo -e "You can now run your MCP clients (Gemini CLI, Cursor, Claude Desktop)."
    echo -e "CodeGraphContext uses workspace-isolated DB files under '.codegraphcontext_db'."
    echo -e "Graphify server is configured to prevent startup timeouts."
    echo -e "\n${YELLOW}NOTE: You may need to restart your terminal or MCP client for changes to take effect.${NC}"
else
    echo -e "\n${RED}${BOLD}Setup completed with errors. Please check the log above.${NC}"
    exit 1
fi
