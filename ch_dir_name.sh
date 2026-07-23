#!/bin/bash
set -e

# formato: "nome_atual|novo_numero"
# comenta a linha se decidir remover em vez de renumerar
declare -a MAPPING=(
  "4-tcp_chat_server|01"
  "8-health_check_poller_with_circuit_breaker|02"   # decisão sua -- comenta se for remover
  "12-tcp_server_worker_pools|03"
  "14-mining_pool_with_stratum_protocol|04"          # decisão sua -- comenta se for remover
  "16-websocket_server_multi_room_broadcast|05"
  "22-tcp_multiplexed_stream_broker|06"
  "23-udp_raw_client_server|07"
  "24-udp_reliable_retransmit|08"
  "26-rtp_style_packetizer_seq_timeout|09"
)

for dir in "." "templates"; do
  echo "== processando $dir =="
  for entry in "${MAPPING[@]}"; do
    old_name="${entry%%|*}"
    new_num="${entry##*|}"

    # acha a pasta real (root usa hífen depois do número, templates às vezes usa underscore)
    match=$(find "$dir" -maxdepth 1 -type d \( -name "${old_name/-/[-_]}" -o -name "$old_name" \) | head -n1)
    if [ -z "$match" ]; then
      echo "  (pular, não existe em $dir: $old_name)"
      continue
    fi

    suffix="${old_name#*-}"
    new_name="$dir/${new_num}-${suffix}"

    echo "  git mv \"$match\" \"$new_name\""
    git mv "$match" "$new_name"
  done
done

echo "done -- confere com 'git status' antes de commitar"