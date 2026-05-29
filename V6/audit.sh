#!/usr/bin/env bash
# System Review audit skripta (LOTL)
# Pokretanje: bash audit.sh
#
# Sekcije prema dokumentaciji:
#   - Operating System + Kernel  (Viktor Srbljin SV63/2022)
#   - Time Management            (Aleksa Šiljić SV50/2022)
#   - Packages + Logging         (Uroš Đosić SV20/2022)

# ---------------------------------------------------------------------------
# Pomoćne funkcije
# ---------------------------------------------------------------------------

section() {
  echo
  echo "=============================="
  echo " $* "
  echo "=============================="
  echo
}

subsection() {
  echo
  echo " $* "
  echo
}

ok()   { echo "[OK] $*"; }
warn() { echo "[WARN] $*"; }

run_or_na() {
  local label=$1
  shift
  echo "[*] ${label}:"
  "$@" 2>/dev/null || echo "n/a"
}

# ---------------------------------------------------------------------------
# OPERATING SYSTEM + KERNEL — Viktor Srbljin SV63/2022
# ---------------------------------------------------------------------------

check_operating_system() {
  subsection "OPERATING SYSTEM CHECK"

  if [[ -f /etc/debian_version ]]; then
    echo "[*] cat /etc/debian_version:"
    cat /etc/debian_version
  elif [[ -f /etc/redhat-release ]]; then
    echo "[*] cat /etc/redhat-release:"
    cat /etc/redhat-release
  elif [[ -f /etc/fedora-release ]]; then
    echo "[*] cat /etc/fedora-release:"
    cat /etc/fedora-release
  elif command -v lsb_release >/dev/null 2>&1; then
    echo "[*] lsb_release -a:"
    lsb_release -a 2>/dev/null
  else
    echo "OS: nepoznato"
  fi

  echo "[*] Proveri da li distro još dobija security patch-eve (EOL/support)."
}

check_kernel() {
  subsection "KERNEL CHECK"

  run_or_na "uname -a" uname -a

  echo
  run_or_na "pokrenuti kernel (uname -r)" uname -r

  echo
  echo "[*] instalirani kernel paketi (dpkg -l):"
  dpkg -l 2>/dev/null | grep linux-image || echo "n/a"

  echo
  run_or_na "poslednji boot (who -b)" who -b

  echo
  run_or_na "uptime" uptime

  local uptime_days
  uptime_days=$(uptime -p 2>/dev/null | grep -oE '[0-9]+ day' | grep -oE '[0-9]+' | head -1)
  uptime_days=${uptime_days:-0}

  if [[ "$uptime_days" -ge 50 ]]; then
    warn "Sistem je up ${uptime_days} dana — verovatno nije bilo kernel reboot-a posle patch-a."
  else
    ok "Uptime: ${uptime_days} dana."
  fi

  echo
  echo "--- REZIME KERNEL ---"
  echo "Pokrenuti kernel: $(uname -r 2>/dev/null)"
  echo "Uporedi uname -r sa dpkg linux-image paketima."
  echo "Ako je instaliran noviji kernel ili je uptime dug -> verovatno treba reboot."
}

# ---------------------------------------------------------------------------
# TIME MANAGEMENT — Aleksa Šiljić SV50/2022
# ---------------------------------------------------------------------------

check_timezone() {
  echo "[*] Trenutna vremenska zona (cat /etc/timezone):"

  if [[ -f /etc/timezone ]]; then
    local tz_val
    tz_val=$(cat /etc/timezone)
    echo "    ${tz_val}"

    if [[ "$tz_val" != "Etc/UTC" && "$tz_val" != "UTC" ]]; then
      warn "Zona nije UTC (${tz_val}). Proveri da li zona ima DST — skokovi vremena su loši za logove."
    else
      ok "Zona je UTC — nema DST, logovi će biti konzistentni."
    fi
    return
  fi

  local tz_timedatectl
  tz_timedatectl=$(timedatectl 2>/dev/null | grep "Time zone" | awk '{print $3}')
  if [[ -n "$tz_timedatectl" ]]; then
    echo "    ${tz_timedatectl} (timedatectl)"
    if [[ "$tz_timedatectl" != "UTC" ]]; then
      warn "Zona nije UTC (${tz_timedatectl}). Proveri DST."
    else
      ok "Zona je UTC."
    fi
  else
    echo "    n/a (/etc/timezone nije prisutan, timedatectl nije dostupan)"
  fi
}

check_ntp_service() {
  echo "[*] Da li NTP servis radi (ps -edf | grep ntp):"

  local ntp_proc
  ntp_proc=$(ps -edf 2>/dev/null | grep -E '[n]tp[d]?')

  if [[ -n "$ntp_proc" ]]; then
    ok "NTP proces je pronađen:"
    echo "$ntp_proc"
    return
  fi

  warn "NTP proces NIJE pronađen u listi procesa."
  echo "       Sistem možda koristi systemd-timesyncd ili chrony — proveravamo..."

  if systemctl is-active --quiet systemd-timesyncd 2>/dev/null; then
    ok "systemd-timesyncd je aktivan (alternativa za NTP)."
  elif systemctl is-active --quiet chrony 2>/dev/null \
      || systemctl is-active --quiet chronyd 2>/dev/null; then
    ok "chrony/chronyd je aktivan (alternativa za NTP)."
  else
    warn "Ni ntpd, ni systemd-timesyncd, ni chrony nisu aktivni."
    echo "       Vreme sistema se verovatno NE sinhronizuje automatski — bezbednosni rizik."
  fi
}

check_timedatectl_fallback() {
  if ! command -v timedatectl >/dev/null 2>&1; then
    return
  fi

  echo "[*] timedatectl status (alternativa za proveru sinhronizacije):"
  timedatectl 2>/dev/null | grep -E 'NTP|synchronized|time zone|RTC' || echo "    n/a"

  if timedatectl 2>/dev/null | grep -q 'synchronized: yes'; then
    ok "Sistem je sinhronizovan (timedatectl)."
  else
    warn "timedatectl ne pokazuje uspešnu sinhronizaciju."
  fi
}

check_ntp_peers() {
  echo "[*] NTP peer status (ntpq -p -n) — proverava da li NTP može da dosegne servere:"

  if ! command -v ntpq >/dev/null 2>&1; then
    echo "    ntpq nije dostupan na ovom sistemu."
    check_timedatectl_fallback
    return
  fi

  local ntpq_out
  ntpq_out=$(ntpq -p -n 2>/dev/null)

  if [[ -z "$ntpq_out" ]]; then
    echo "    ntpq nije vratio izlaz (možda NTP nije pokrenut ili je blokiran)."
    return
  fi

  echo "$ntpq_out"

  if echo "$ntpq_out" | grep -q '^\*'; then
    ok "Postoji aktivan NTP peer (označen sa '*') — sinhronizacija funkcioniše."
  else
    warn "Nema aktivnog NTP peer-a (nijedan red ne počinje sa '*')."
    echo "       NTP radi ali možda ne može da sinhronizuje vreme — proveri mrežu/firewall."
  fi
}

check_time_management() {
  section "TIME MANAGEMENT CHECK"

  check_timezone
  echo
  check_ntp_service
  echo
  check_ntp_peers
  echo

  echo "--- REZIME TIME MANAGEMENT ---"
  echo "Vremenska zona: $(cat /etc/timezone 2>/dev/null || timedatectl 2>/dev/null | grep 'Time zone' | awk '{print $3}')"
  echo "Preporuke:"
  echo "  - Koristi UTC zonu na serverima (nema DST skokova u logovima)."
  echo "  - NTP mora biti aktivan i sinhronizovan sa dostupnim peer-ovima."
  echo "  - Ne koristi ntpdate za ručno podešavanje — koristi stalni NTP servis."
  echo "  - Posle promene firewall pravila, provjeri da NTP i dalje može pristupiti serverima."
}

# ---------------------------------------------------------------------------
# PACKAGES + LOGGING — Uroš Đosić SV20/2022
# ---------------------------------------------------------------------------

check_packages() {
  subsection "PACKAGES CHECK"

  echo "[*] broj instaliranih paketa:"
  dpkg -l 2>/dev/null | wc -l || echo "n/a"

  echo
  echo "[*] lista instaliranih paketa (prvih 40):"
  dpkg -l 2>/dev/null | head -40 || echo "n/a"

  echo
  echo "[*] GUI / desktop paketi:"
  if dpkg -l 2>/dev/null | egrep -q 'gnome|kde|xfce|xorg|xserver'; then
    dpkg -l 2>/dev/null | egrep 'gnome|kde|xfce|xorg|xserver'
  else
    ok "Nisu pronađeni GUI paketi."
  fi

  echo
  echo "[*] game paketi:"
  if dpkg -l 2>/dev/null | egrep -q 'game|steam|supertux'; then
    dpkg -l 2>/dev/null | egrep 'game|steam|supertux'
  else
    ok "Nisu pronađeni game paketi."
  fi

  echo
  echo "[*] proveri nepotrebne servise i pakete za ovu ulogu sistema."
  echo "Web server ne bi trebalo da ima GUI, igre ili development alate ako nisu potrebni."

  echo
  echo "--- REZIME PAKETA ---"
  echo "Minimalan broj paketa = manja attack surface."
  echo "Potrebno proveriti zastarele i nepotrebne pakete."
}

check_logging() {
  subsection "LOGGING CHECK"

  echo "[*] provera syslog procesa:"
  ps -edf 2>/dev/null | grep syslog | grep -v grep || echo "n/a"

  echo
  echo "[*] provera rsyslog konfiguracije:"
  if [[ -f /etc/rsyslog.conf ]]; then
    grep -E 'imudp|UDPServerRun|imtcp|InputTCPServerRun|@' \
      /etc/rsyslog.conf 2>/dev/null || echo "nema relevantnih stavki"
  else
    echo "rsyslog.conf nije pronađen"
  fi

  echo
  echo "[*] permisije log fajlova:"
  grep -E 'FileCreateMode|DirCreateMode|Umask' \
    /etc/rsyslog.conf 2>/dev/null || echo "n/a"

  echo
  echo "[*] provera remote logging:"
  if grep -q '@' /etc/rsyslog.conf 2>/dev/null; then
    ok "Remote logging je konfigurisan."
  else
    warn "Remote logging nije konfigurisan."
  fi

  echo
  echo "[*] provera postojanja glavnih log fajlova:"
  ls -l /var/log/syslog /var/log/auth.log 2>/dev/null || echo "n/a"

  echo
  echo "--- REZIME LOGGING ---"
  echo "Potrebno proveriti:"
  echo "- da li logging servis radi"
  echo "- da li su permisije log fajlova ograničene"
  echo "- da li postoji remote logging"
  echo "- da li su UDP/TCP syslog receiver-i uključeni nepotrebno"
}

# ---------------------------------------------------------------------------
# Glavni tok
# ---------------------------------------------------------------------------

main() {
  echo "=== System Review Audit ==="
  echo "Host: $(hostname 2>/dev/null)"
  echo "Datum: $(date 2>/dev/null)"

  check_operating_system
  check_kernel
  check_time_management
  check_packages
  check_logging

  echo
  echo " KRAJ SKRIPTE "
}

main "$@"
