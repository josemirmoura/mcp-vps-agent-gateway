#!/usr/bin/env bash

vps_agent_scope_list_contains() {
  if [ "$#" -ne 2 ]; then
    return 2
  fi
  local list="$1"
  local wanted="$2"
  local scope
  for scope in $list; do
    if [ "$scope" = "$wanted" ]; then
      return 0
    fi
  done
  return 1
}
