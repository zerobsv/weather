# Include this snippet in the local bashrc for WSL2

if [ $(ulimit -n) -lt 10000000 ]; then
    sudo prlimit --pid=$$ --nofile=10000000:10000000 >/dev/null 2>&1
fi
