# !/bin/sh
for ip in `cat ip.txt`
do
  ping -c 1 -w 3 $ip >/dev/nul 2>&1 && echo "$ip pass" || echo "$ip loss"
done 
