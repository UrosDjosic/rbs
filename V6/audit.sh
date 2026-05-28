#!/usr/bin/env bash
# System Review audit skripta (LOTL)
# Pokretanje: bash audit.sh

echo "=== System Review Audit ==="
echo "Host: $(hostname 2>/dev/null)"
echo "Datum: $(date 2>/dev/null)"
echo


# OPERATING SYSTEM + KERNEL  Viktor Srbljin SV63/2022

echo " OPERATING SYSTEM CHECK "

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

echo "[*] Proveri da li distro jos dobija security patch-eve (EOL/support)."
echo

echo " KERNEL CHECK "

echo "[*] uname -a:"
uname -a 2>/dev/null || echo "n/a"

echo
echo "[*] pokrenuti kernel (uname -r):"
uname -r 2>/dev/null || echo "n/a"

echo
echo "[*] instalirani kernel paketi (dpkg -l):"
dpkg -l 2>/dev/null | grep linux-image || echo "n/a"

echo
echo "[*] poslednji boot (who -b):"
who -b 2>/dev/null || echo "n/a"

echo
echo "[*] uptime:"
uptime 2>/dev/null || echo "n/a"

UPTIME_DAYS=$(uptime -p 2>/dev/null | grep -oE '[0-9]+ day' | grep -oE '[0-9]+' | head -1)
UPTIME_DAYS=${UPTIME_DAYS:-0}
if [[ "$UPTIME_DAYS" -ge 50 ]]; then
    echo "[WARN] Sistem je up ${UPTIME_DAYS} dana - verovatno nije bilo kernel reboot-a posle patch-a."
else
    echo "[OK] Uptime: ${UPTIME_DAYS} dana."
fi

echo
echo "--- REZIME KERNEL ---"
echo "Pokrenuti kernel: $(uname -r 2>/dev/null)"
echo "Uporedi uname -r sa dpkg linux-image paketima."
echo "Ako je instaliran noviji kernel ili je uptime dug -> verovatno treba reboot."
echo


# TIME MANAGEMENT  CLAN 3
echo "=============================="
echo " TIME MANAGEMENT CHECK "
echo "=============================="
echo
 
# --- 1. Provera vremenske zone ---
# Zasto: Osetljivi sistemi bi trebalo da koriste UTC ili zonu bez letnjeg racunanja vremena
#        (daylight saving time - DST). DST moze da prouzrokuje "skokove" u logovima,
#        sto otezava korelaciju dogadjaja iz razlicitih izvora.
echo "[*] Trenutna vremenska zona (cat /etc/timezone):"
if [[ -f /etc/timezone ]]; then
    TZ_VAL=$(cat /etc/timezone)
    echo "    $TZ_VAL"
    # Upozorenje ako zona nije UTC - ne mora da znaci problem, ali vredi proveriti
    if [[ "$TZ_VAL" != "Etc/UTC" && "$TZ_VAL" != "UTC" ]]; then
        echo "[WARN] Zona nije UTC ($TZ_VAL). Proveri da li zona ima DST - skokovi vremena su losi za logove."
    else
        echo "[OK] Zona je UTC - nema DST, logovi ce biti konzistentni."
    fi
else
    # Na novijim sistemima zona se cita drugacije
    TZ_TIMEDATECTL=$(timedatectl 2>/dev/null | grep "Time zone" | awk '{print $3}')
    if [[ -n "$TZ_TIMEDATECTL" ]]; then
        echo "    $TZ_TIMEDATECTL (timedatectl)"
        if [[ "$TZ_TIMEDATECTL" != "UTC" ]]; then
            echo "[WARN] Zona nije UTC ($TZ_TIMEDATECTL). Proveri DST."
        else
            echo "[OK] Zona je UTC."
        fi
    else
        echo "    n/a (/etc/timezone nije prisutan, timedatectl nije dostupan)"
    fi
fi
echo
 
# --- 2. Provera da li NTP servis radi ---
# Zasto: Bez NTP-a vreme na sistemu moze da se razlikuje od stvarnog, sto kvari:
#        - SSL verifikaciju sertifikata (expired/not yet valid greske),
#        - autentifikaciju zasnovanu na vremenu (npr. Kerberos),
#        - korelaciju log zapisa sa razlicitih masina.
#        Rucno podesavanje vremena (ntpdate jednom) nije dovoljno - NTP drzis sinhronizaciju stalnom.
echo "[*] Da li NTP servis radi (ps -edf | grep ntp):"
NTP_PROC=$(ps -edf 2>/dev/null | grep -E '[n]tp[d]?')
if [[ -n "$NTP_PROC" ]]; then
    echo "[OK] NTP proces je pronadjen:"
    echo "$NTP_PROC"
else
    echo "[WARN] NTP proces NIJE pronadjen u listi procesa."
    echo "       Sistem mozda koristi systemd-timesyncd ili chrony - proveravamo..."
 
    # --- 2a. Alternativa: systemd-timesyncd (uobicajeno na Ubuntu/Debian bez ntpd) ---
    # Zasto: Moderni Debian/Ubuntu sistemi cesto koriste systemd-timesyncd umesto ntpd.
    #        Ako ni ovo nije aktivno, sistem verovatno uopste ne sinhronizuje vreme.
    if systemctl is-active --quiet systemd-timesyncd 2>/dev/null; then
        echo "[OK] systemd-timesyncd je aktivan (alternativa za NTP)."
    elif systemctl is-active --quiet chrony 2>/dev/null || systemctl is-active --quiet chronyd 2>/dev/null; then
        echo "[OK] chrony/chronyd je aktivan (alternativa za NTP)."
    else
        echo "[WARN] Ni ntpd, ni systemd-timesyncd, ni chrony nisu aktivni."
        echo "       Vreme sistema se verovatno NE sinhronizuje automatski - bezbednosni rizik."
    fi
fi
echo
 
# --- 3. Provera sinhronizacije NTP peer-ova ---
# Zasto: NTP servis moze da radi ali da ne moze da dosegne svoje servere (firewall, DNS problem).
#        ntpq -p -n prikazuje listu peer-ova i da li je sinhronizacija uspesna.
#        Zvezda (*) ispred adrese znaci aktivni izvor; plus (+) znaci kandidat.
#        Ako nema ni jednog obelezenog peer-a, sinhronizacija nije funkcionalna.
echo "[*] NTP peer status (ntpq -p -n) - proverava da li NTP moze da dosegne servere:"
if command -v ntpq >/dev/null 2>&1; then
    NTPQ_OUT=$(ntpq -p -n 2>/dev/null)
    if [[ -n "$NTPQ_OUT" ]]; then
        echo "$NTPQ_OUT"
        # Trazimo aktivni peer - red koji pocinje sa '*'
        if echo "$NTPQ_OUT" | grep -q '^\*'; then
            echo "[OK] Postoji aktivan NTP peer (oznacen sa '*') - sinhronizacija funkcionise."
        else
            echo "[WARN] Nema aktivnog NTP peer-a (nijedan red ne pocinje sa '*')."
            echo "       NTP radi ali mozda ne moze da sinhronizuje vreme - proveri mrezu/firewall."
        fi
    else
        echo "    ntpq nije vratio izlaz (mozda NTP nije pokrenut ili je blokiran)."
    fi
else
    echo "    ntpq nije dostupan na ovom sistemu."
    # Pokusaj sa timedatectl ako je dostupan
    if command -v timedatectl >/dev/null 2>&1; then
        echo "[*] timedatectl status (alternativa za proveru sinhronizacije):"
        timedatectl 2>/dev/null | grep -E 'NTP|synchronized|time zone|RTC' || echo "    n/a"
        # Zasto: timedatectl pokazuje da li je NTP sinhronizacija ukljucena i da li je uspesna
        if timedatectl 2>/dev/null | grep -q 'synchronized: yes'; then
            echo "[OK] Sistem je sinhronizovan (timedatectl)."
        else
            echo "[WARN] timedatectl ne pokazuje uspesnu sinhronizaciju."
        fi
    fi
fi
echo
 
# --- 4. Rezime Time Management ---
echo "--- REZIME TIME MANAGEMENT ---"
echo "Vremenska zona: $(cat /etc/timezone 2>/dev/null || timedatectl 2>/dev/null | grep 'Time zone' | awk '{print $3}')"
echo "Preporuke:"
echo "  - Koristi UTC zonu na serverima (nema DST skokova u logovima)."
echo "  - NTP mora biti aktivan i sinhronizovan sa dostupnim peer-ovima."
echo "  - Ne koristi ntpdate za rucno podesavanje - koristi stalni NTP servis."
echo "  - Posle promene firewall pravila, provjeri da NTP i dalje moze pristupiti serverima."
echo
 
 
# PACKAGES + LOGGING  Uros Djosic SV20/2022

echo " PACKAGES CHECK "

echo "[*] broj instaliranih paketa:"
dpkg -l 2>/dev/null | wc -l || echo "n/a"

echo
echo "[*] lista instaliranih paketa (prvih 40):"
dpkg -l 2>/dev/null | head -40 || echo "n/a"

echo
echo "[*] GUI / desktop paketi:"
dpkg -l 2>/dev/null | egrep 'gnome|kde|xfce|xorg|xserver' || \
echo "[OK] Nisu pronadjeni GUI paketi."

echo
echo "[*] game paketi:"
dpkg -l 2>/dev/null | egrep 'game|steam|supertux' || \
echo "[OK] Nisu pronadjeni game paketi."

echo
echo "[*] proveri nepotrebne servise i pakete za ovu ulogu sistema."
echo "Web server ne bi trebalo da ima GUI, igre ili development alate ako nisu potrebni."

echo
echo "--- REZIME PAKETA ---"
echo "Minimalan broj paketa = manja attack surface."
echo "Potrebno proveriti zastarele i nepotrebne pakete."
echo


echo " LOGGING CHECK "

echo "[*] provera syslog procesa:"
ps -edf 2>/dev/null | grep syslog | grep -v grep || echo "n/a"

echo
echo "[*] provera rsyslog konfiguracije:"
if [[ -f /etc/rsyslog.conf ]]; then
    grep -E 'imudp|UDPServerRun|imtcp|InputTCPServerRun|@' \
    /etc/rsyslog.conf 2>/dev/null || echo "nema relevantnih stavki"
else
    echo "rsyslog.conf nije pronadjen"
fi

echo
echo "[*] permisije log fajlova:"
grep -E 'FileCreateMode|DirCreateMode|Umask' \
/etc/rsyslog.conf 2>/dev/null || echo "n/a"

echo
echo "[*] provera remote logging:"
if grep -q '@' /etc/rsyslog.conf 2>/dev/null; then
    echo "[OK] Remote logging je konfigurisan."
else
    echo "[WARN] Remote logging nije konfigurisan."
fi

echo
echo "[*] provera postojanja glavnih log fajlova:"
ls -l /var/log/syslog /var/log/auth.log 2>/dev/null || echo "n/a"

echo
echo "--- REZIME LOGGING ---"
echo "Potrebno proveriti:"
echo "- da li logging servis radi"
echo "- da li su permisije log fajlova ogranicene"
echo "- da li postoji remote logging"
echo "- da li su UDP/TCP syslog receiver-i ukljuceni nepotrebno"
echo


echo " KRAJ SKRIPTE "