#!/usr/bin/env python3
"""
hcm-client: Host Credential Manager SSH Client (fzf-powered)

Interactive SSH launcher that connects to the Host Credential Manager (HCM) server,
fetches hosts having SSH in their AccessList, presents a searchable menu using fzf,
retrieves the password securely using masterpassword, and connects via SSH.
"""

import os
import sys
import argparse
import getpass
import shutil
import subprocess
from pathlib import Path

try:
    import requests
    from requests.adapters import HTTPAdapter
except ImportError:
    print("Error: 'requests' library is required. Please run: pip install requests", file=sys.stderr)
    sys.exit(1)

import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)


class HostnameIgnoreAdapter(HTTPAdapter):
    """Adapter to ignore hostname mismatches while still verifying against CA certificate."""
    def init_poolmanager(self, *args, **kwargs):
        kwargs['assert_hostname'] = False
        return super().init_poolmanager(*args, **kwargs)


def resolve_cert_path(specified_cert: str = None) -> str | None:
    """Find this project's CA or server certificate."""
    if specified_cert:
        p = Path(specified_cert).expanduser().resolve()
        if p.exists():
            return str(p)
        print(f"Warning: Specified cert file not found: {specified_cert}", file=sys.stderr)

    script_dir = Path(__file__).resolve().parent
    candidates = [
        script_dir / "cert" / "cacert.pem",
        script_dir / "cert" / "cert.pem",
        Path("cert/cacert.pem").resolve(),
        Path("cert/cert.pem").resolve(),
    ]
    for c in candidates:
        if c.exists():
            return str(c)
    return None


def get_session(cert_path: str | None) -> tuple[requests.Session, str | bool]:
    session = requests.Session()
    session.mount('https://', HostnameIgnoreAdapter())
    verify = cert_path if cert_path else False
    return session, verify


def fetch_ssh_targets(server_url: str, session: requests.Session, verify: str | bool) -> list[dict]:
    url = f"{server_url.rstrip('/')}/api/ssh-fzf"
    try:
        resp = session.get(url, verify=verify, timeout=10)
        resp.raise_for_status()
        data = resp.json()
        if isinstance(data, list):
            return data
        if isinstance(data, dict) and "targets" in data:
            return data["targets"]
        return []
    except requests.exceptions.SSLError as e:
        print(f"\n[SSL/TLS Error] Failed to verify certificate connecting to {url}", file=sys.stderr)
        print(f"Details: {e}", file=sys.stderr)
        print("Tip: Specify certificate path using --cert or check cert/cacert.pem in this project.\n", file=sys.stderr)
        sys.exit(1)
    except requests.exceptions.ConnectionError as e:
        print(f"\n[Connection Error] Could not connect to HCM server at {server_url}", file=sys.stderr)
        print("Please verify the server is running and accessible.", file=sys.stderr)
        print(f"Details: {e}\n", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"\n[Error] Failed to fetch SSH targets: {e}\n", file=sys.stderr)
        sys.exit(1)


def pick_target(targets: list[dict]) -> dict | None:
    if not shutil.which("fzf"):
        print("Error: 'fzf' is not installed or not found in PATH.", file=sys.stderr)
        sys.exit(1)

    # Format entries for fzf
    display_map = {}
    display_lines = []
    for t in targets:
        hostname = t.get("hostname", "")
        username = t.get("username", "")
        ip = t.get("ip", "")
        port = str(t.get("port", "22"))
        port_suffix = f":{port}" if port != "22" else ""
        line = f"{hostname:<30} {username:<15} {ip}{port_suffix}"
        display_map[line] = t
        display_lines.append(line)

    fzf_cmd = [
        "fzf",
        "--prompt=Select SSH Host > ",
        "--header=HOSTNAME                       USERNAME        IP:PORT",
        "--height=40%",
        "--reverse",
        "--border",
    ]

    try:
        proc = subprocess.run(
            fzf_cmd,
            input="\n".join(display_lines).encode("utf-8"),
            stdout=subprocess.PIPE,
            check=False
        )
    except KeyboardInterrupt:
        return None

    if proc.returncode != 0:
        return None

    selected = proc.stdout.decode("utf-8").strip()
    if not selected:
        return None

    return display_map.get(selected)


def get_ssh_password(server_url: str, session: requests.Session, verify: str | bool, hostname: str, username: str, masterpassword: str) -> str:
    url = f"{server_url.rstrip('/')}/api/ssh-fzf"
    payload = {
        "masterpassword": masterpassword,
        "hostname": hostname,
        "username": username,
    }
    try:
        resp = session.post(url, json=payload, verify=verify, timeout=10)
        resp.raise_for_status()
        data = resp.json()
        return data.get("value", "")
    except Exception as e:
        print(f"Failed to query credentials: {e}", file=sys.stderr)
        return ""


def main():
    parser = argparse.ArgumentParser(description="HCM SSH FZF Client")
    parser.add_argument("--url", default=os.environ.get("HCM_URL", "https://127.0.0.1:8080"), help="HCM server base URL (default: https://127.0.0.1:8080)")
    parser.add_argument("--cert", default=os.environ.get("HCM_CERT"), help="Path to CA/server certificate (default: ./cert/cacert.pem)")
    parser.add_argument("--list", action="store_true", help="Print SSH targets list and exit")
    args = parser.parse_args()

    server_url = args.url
    cert_path = resolve_cert_path(args.cert)
    session, verify = get_session(cert_path)

    targets = fetch_ssh_targets(server_url, session, verify)
    if not targets:
        print("No SSH hosts found in HCM database (no hosts with 'ssh' in AccessList).")
        return

    if args.list:
        print(f"Found {len(targets)} SSH targets on {server_url}:")
        print(f"{'HOSTNAME':<30} {'USERNAME':<15} {'IP':<18} {'PORT'}")
        print("-" * 70)
        for t in targets:
            print(f"{t.get('hostname', ''):<30} {t.get('username', ''):<15} {t.get('ip', ''):<18} {t.get('port', '22')}")
        return

    target = pick_target(targets)
    if not target:
        print("Selection canceled.")
        return

    hostname = target.get("hostname", "")
    username = target.get("username", "")
    ip = target.get("ip", "")
    port = str(target.get("port", "22"))

    masterpassword = getpass.getpass("masterpassword: ")
    if not masterpassword:
        print("Masterpassword cannot be empty.", file=sys.stderr)
        sys.exit(1)

    password = get_ssh_password(server_url, session, verify, hostname, username, masterpassword)
    if not password:
        print(f"Error: Invalid masterpassword or no credentials found for {hostname} ({username}).", file=sys.stderr)
        sys.exit(1)

    if not shutil.which("sshpass"):
        print("Error: 'sshpass' is required for automatic password login. Please install sshpass.", file=sys.stderr)
        sys.exit(1)

    os.environ["SSHPASS"] = password

    ssh_cmd = ["sshpass", "-e", "ssh", "-o", "StrictHostKeyChecking=accept-new"]
    if port != "22":
        ssh_cmd.extend(["-p", port])
    ssh_cmd.append(f"{username}@{ip}")

    port_info = f" (port {port})" if port != "22" else ""
    print(f"Connecting to {username}@{ip}{port_info} ...")
    sys.stdout.flush()
    sys.stderr.flush()

    os.execvp("sshpass", ssh_cmd)


if __name__ == "__main__":
    main()
