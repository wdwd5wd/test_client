cmd="sudo sysctl -w net.ipv4.tcp_fin_timeout=3"
echo $cmd
eval $cmd

cmd="sudo sysctl -w net.ipv4.tcp_timestamps=1"
echo $cmd
eval $cmd

cmd="sudo sysctl -w net.ipv4.tcp_tw_reuse=1"
echo $cmd
eval $cmd

cmd="sudo sysctl -w net.ipv4.ip_local_port_range=10000"
echo $cmd
eval $cmd

