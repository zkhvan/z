export Z_SOURCED=1

# wrapper for z command
z() {
  local output
  local exit_code

  if [[ "$1" == "project" && "$2" == "select" ]]; then
    output="$(command z "$@")"
    exit_code=$?

    if [[ $exit_code -ne 0 ]]; then
      echo "Error: 'z project select' failed with exit code $exit_code" >&2
      print -r "$output" >&2
      return $exit_code
    fi

    print -r "${output}"
    if [[ "${output##*$'\n'}" == "cd "* ]]; then
      builtin cd -- "${output##*$'\n'#cd }"
      return 0
    fi
  else
    command z "$@"
    return $?
  fi
}
